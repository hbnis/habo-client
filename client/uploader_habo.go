package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ao-data/albiondata-client/log"
)

type haboUploader struct {
	baseURL string
	client  *http.Client
}

func newHaboUploader(target string) uploader {
	baseURL := strings.TrimPrefix(target, "habo+")
	return &haboUploader{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Transport: &http.Transport{}, Timeout: httpUploadTimeout},
	}
}

func (u *haboUploader) sendToIngest(body []byte, topic string, state *albionState, identifier string) {
	token := GetHaboToken()
	if token == "" {
		log.Warn("Habo private upload skipped because this client is not connected to The Habo Hub.")
		return
	}

	req, err := http.NewRequest("POST", u.baseURL+"/"+topic, bytes.NewReader(body))
	if err != nil {
		log.Errorf("Could not create Habo private ingest request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("habo-client/%s", version))
	req.Header.Set("X-Habo-Server-Id", fmt.Sprintf("%d", state.AODataServerID))
	if identifier != "" {
		req.Header.Set("X-Habo-Identifier", identifier)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		log.Errorf("Habo private market upload failed: %v", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Errorf("Habo private market upload returned HTTP %d", resp.StatusCode)
		return
	}

	log.Infof("Sent private %s data to The Habo Hub.", topic)
}
