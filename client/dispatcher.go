package client

import (
	"encoding/json"
	"net/http"
	"sync"

	"strings"

	"github.com/ao-data/albiondata-client/internal/dashboard"
	"github.com/ao-data/albiondata-client/lib"
	"github.com/ao-data/albiondata-client/log"
)

type dispatcher struct{}

var (
	wsHub *WSHub
	dis   *dispatcher

	// uploaderCacheMu guards uploaderCache.
	uploaderCacheMu sync.Mutex
	// uploaderCache holds one uploader per ingest target, reused across
	// every upload. Every market order, gold price, or mail upload calls
	// createUploaders, and building a fresh uploader per call used to open
	// a brand new, never-closed NATS connection (with its own internal
	// reconnect/buffering goroutines) or a fresh http.Transport whose idle
	// keep-alive connections were never reclaimed - both leaked memory
	// continuously, and far more sharply during network instability, when
	// abandoned NATS connections would sit retrying with buffered data.
	uploaderCache = map[string]uploader{}

	uploadModeMu sync.RWMutex
	uploadMode   = UploadModePublic
)

const (
	UploadModePublic  = "public"
	UploadModePrivate = "private"
)

func GetUploadMode() string {
	uploadModeMu.RLock()
	defer uploadModeMu.RUnlock()
	return uploadMode
}

func PrivateUploadConfigured() bool {
	return strings.TrimSpace(ConfigGlobal.PrivateIngestBaseUrls) != ""
}

func SetUploadMode(mode string) bool {
	if mode != UploadModePublic && mode != UploadModePrivate {
		return false
	}
	if mode == UploadModePrivate && !PrivateUploadConfigured() {
		return false
	}
	uploadModeMu.Lock()
	uploadMode = mode
	uploadModeMu.Unlock()
	return true
}

func isHaboMarketTopic(topic string) bool {
	return topic == lib.NatsMarketOrdersIngest || topic == lib.NatsMarketHistoriesIngest
}

func createDispatcher() {
	dis = &dispatcher{}

	if ConfigGlobal.EnableWebsockets {
		wsHub = newHub()
		go wsHub.run()
		go runHTTPServer()
	}
}

func createUploaders(targets []string) []uploader {
	uploaderCacheMu.Lock()
	defer uploaderCacheMu.Unlock()

	var uploaders []uploader
	for _, target := range targets {
		if target == "" {
			continue
		}
		if len(target) < 4 {
			log.Infof("Got an ingest target that was less than 4 characters, not a valid ingest target: %v", target)
			continue
		}

		if cached, ok := uploaderCache[target]; ok {
			uploaders = append(uploaders, cached)
			continue
		}

		var u uploader
		if target[0:8] == "http+pow" || target[0:9] == "https+pow" {
			u = newHTTPUploaderPow(target)
		} else if target[0:4] == "http" || target[0:5] == "https" {
			u = newHTTPUploader(target)
		} else if target[0:4] == "nats" {
			u = newNATSUploader(target)
		} else {
			log.Infof("An invalid ingest target was specified: %v", target)
			continue
		}

		uploaderCache[target] = u
		uploaders = append(uploaders, u)
	}

	return uploaders
}

func sendMsgToPublicUploaders(upload interface{}, topic string, state *albionState, identifier string, recordCount int) {
	// Habo Client is intentionally focused on market flipping. Keep the
	// proven AODP decoder, but only forward market orders and market history.
	if !isHaboMarketTopic(topic) {
		return
	}

	dashboard.IncrementCounterBy(topic, int64(recordCount))

	data, err := json.Marshal(upload)
	if err != nil {
		log.Errorf("Error while marshalling payload for %v: %v", err, topic)
		return
	}

	switch GetUploadMode() {
	case UploadModePrivate:
		privateUploaders := createUploaders(strings.Split(ConfigGlobal.PrivateIngestBaseUrls, ","))
		if len(privateUploaders) == 0 {
			log.Warn("Private scan mode is selected but no Habo private ingest is configured.")
			return
		}
		sendMsgToUploaders(data, topic, privateUploaders, state, identifier)

	default:
		publicIngestBaseUrls := ConfigGlobal.PublicIngestBaseUrls
		// https+pow://albion-online-data.com is the AODP placeholder for every realm.
		if strings.Contains(publicIngestBaseUrls, "https+pow://albion-online-data.com") {
			publicIngestBaseUrls = strings.Replace(publicIngestBaseUrls, "https+pow://albion-online-data.com", state.AODataIngestBaseURL, -1)
		}
		publicUploaders := createUploaders(strings.Split(publicIngestBaseUrls, ","))
		sendMsgToUploaders(data, topic, publicUploaders, state, identifier)
	}

	if ConfigGlobal.EnableWebsockets {
		sendMsgToWebSockets(data, topic)
	}
}

func sendMsgToPrivateUploaders(upload lib.PersonalizedUpload, topic string, state *albionState, identifier string) {
	if !isHaboMarketTopic(topic) {
		return
	}
	if ConfigGlobal.DisableUpload {
		log.Info("Upload is disabled.")
		return
	}

	// TODO: Re-enable this when issue #14 is fixed
	// Will personalize with blanks for now in order to allow people to see the format
	// if state.CharacterName == "" || state.CharacterId == "" {
	// 	log.Error("The player name or id has not been set. Please restart the game and make sure the client is running.")
	// 	notification.Push("The player name or id has not been set. Please restart the game and make sure the client is running.")
	// 	return
	// }

	upload.Personalize(state.CharacterId, state.CharacterName)

	data, err := json.Marshal(upload)
	if err != nil {
		log.Errorf("Error while marshalling payload for %v: %v", err, topic)
		return
	}

	var privateUploaders = createUploaders(strings.Split(ConfigGlobal.PrivateIngestBaseUrls, ","))
	if len(privateUploaders) > 0 {
		sendMsgToUploaders(data, topic, privateUploaders, state, identifier)
	}

	// If websockets are enabled, send the data there too
	if ConfigGlobal.EnableWebsockets {
		sendMsgToWebSockets(data, topic)
	}
}

func sendMsgToUploaders(msg []byte, topic string, uploaders []uploader, state *albionState, identifier string) {
	if ConfigGlobal.DisableUpload {
		log.Info("Upload is disabled.")
		return
	}

	for _, u := range uploaders {
		u.sendToIngest(msg, topic, state, identifier)
	}
}

func runHTTPServer() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(wsHub, w, r)
	})

	err := http.ListenAndServe(":8099", nil)

	if err != nil {
		log.Panic("ListenAndServe: ", err)
	}
}

func sendMsgToWebSockets(msg []byte, topic string) {
	// TODO (gradius): send JSON data with topic string
	// TODO (gradius): this seems super hacky, and I'm sure there's a better way.
	var result string
	result = "{\"topic\": \"" + topic + "\", \"data\": " + string(msg) + "}"
	wsHub.broadcast <- []byte(result)
}
