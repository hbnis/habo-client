package client

import (
	"net/http"
	"strings"
	"time"

	"github.com/ao-data/albiondata-client/log"
)

type haboUploader struct {
	queue *haboUploadQueue
}

func newHaboUploader(target string) uploader {
	baseURL := strings.TrimPrefix(target, "habo+")
	httpClient := &http.Client{Transport: &http.Transport{}, Timeout: httpUploadTimeout}
	queueDir, err := haboUploadQueueDir()
	if err != nil {
		log.Warnf("Could not resolve Habo upload queue directory; uploads will remain in memory only: %v", err)
		queueDir = ""
	}
	return &haboUploader{
		queue: newHaboUploadQueue(strings.TrimRight(baseURL, "/"), httpClient, queueDir, true),
	}
}

func (u *haboUploader) sendToIngest(body []byte, topic string, state *albionState, identifier string) {
	token := GetHaboToken()
	if token == "" {
		log.Warn("Habo private upload skipped because this client is not connected to The Habo Hub.")
		return
	}

	serverID := state.ServerIDForUpload()
	if serverID == 0 {
		log.Warnf("Habo private %s upload skipped because the Albion server could not be identified yet.", topic)
		return
	}

	if err := u.queue.enqueue(body, topic, serverID, identifier, token, time.Now()); err != nil {
		log.Errorf("Could not queue Habo private %s upload: %v", topic, err)
	}
}
