package client

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const defaultHaboHubURL = "https://habonis.com"

type haboClientConfig struct {
	Token string `json:"token"`
}

type HaboAccountState struct {
	Connected        bool
	Pairing          bool
	DisplayName      string
	ConnectURL       string
	PairCode         string
	PrivateIngestURL string
	Error            string
}

type haboPairStartResponse struct {
	Connected        bool   `json:"connected"`
	Pairing          bool   `json:"pairing"`
	DisplayName      string `json:"displayName"`
	PrivateIngestURL string `json:"privateIngestUrl"`
	PairCode         string `json:"pairCode"`
	ConnectURL       string `json:"connectUrl"`
	ExpiresInSeconds int    `json:"expiresInSeconds"`
	Error            string `json:"error"`
}

type haboSessionResponse struct {
	Connected        bool   `json:"connected"`
	Pairing          bool   `json:"pairing"`
	Expired          bool   `json:"expired"`
	DisplayName      string `json:"displayName"`
	PrivateIngestURL string `json:"privateIngestUrl"`
	Error            string `json:"error"`
}

var (
	haboMu    sync.RWMutex
	haboToken string
	haboHTTP  = &http.Client{Timeout: 15 * time.Second}
)

func haboHubURL() string {
	if value := strings.TrimSpace(os.Getenv("HABO_HUB_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultHaboHubURL
}

func haboConfigPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "Habo Client", "client.json"), nil
}

func loadHaboToken() string {
	haboMu.RLock()
	if haboToken != "" {
		token := haboToken
		haboMu.RUnlock()
		return token
	}
	haboMu.RUnlock()

	path, err := haboConfigPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var config haboClientConfig
	if json.Unmarshal(data, &config) != nil {
		return ""
	}
	token := strings.TrimSpace(config.Token)
	if token == "" {
		return ""
	}

	haboMu.Lock()
	haboToken = token
	haboMu.Unlock()
	return token
}

func saveHaboToken(token string) error {
	path, err := haboConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(haboClientConfig{Token: token})
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	haboMu.Lock()
	haboToken = token
	haboMu.Unlock()
	return nil
}

func clearHaboToken() error {
	path, err := haboConfigPath()
	if err == nil {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return removeErr
		}
	}
	haboMu.Lock()
	haboToken = ""
	haboMu.Unlock()
	SetPrivateIngestBaseURLs("")
	SetUploadMode(UploadModePublic)
	return nil
}

func newHaboToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func haboPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "windows", "linux":
		return runtime.GOOS
	default:
		return "unknown"
	}
}

func haboDeviceName() string {
	if hostname, err := os.Hostname(); err == nil && strings.TrimSpace(hostname) != "" {
		return strings.TrimSpace(hostname)
	}
	return "Habo Client"
}

func GetHaboToken() string {
	return loadHaboToken()
}

func RefreshHaboAccount() HaboAccountState {
	token := loadHaboToken()
	if token == "" {
		return HaboAccountState{}
	}

	req, err := http.NewRequest("GET", haboHubURL()+"/api/client/session", nil)
	if err != nil {
		return HaboAccountState{Error: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", fmt.Sprintf("habo-client/%s", version))

	resp, err := haboHTTP.Do(req)
	if err != nil {
		return HaboAccountState{Error: "Could not reach The Habo Hub."}
	}
	defer resp.Body.Close()

	var payload haboSessionResponse
	if json.NewDecoder(resp.Body).Decode(&payload) != nil {
		return HaboAccountState{Error: "The Habo Hub returned an unreadable response."}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusGone {
		_ = clearHaboToken()
		return HaboAccountState{}
	}
	if resp.StatusCode != http.StatusOK {
		if payload.Error == "" {
			payload.Error = fmt.Sprintf("The Habo Hub returned HTTP %d.", resp.StatusCode)
		}
		return HaboAccountState{Error: payload.Error}
	}

	if payload.Connected {
		SetPrivateIngestBaseURLs("habo+" + strings.TrimRight(payload.PrivateIngestURL, "/"))
		return HaboAccountState{
			Connected:        true,
			DisplayName:      payload.DisplayName,
			PrivateIngestURL: payload.PrivateIngestURL,
		}
	}

	return HaboAccountState{Pairing: payload.Pairing}
}

func StartHaboPairing() HaboAccountState {
	token := loadHaboToken()
	if token != "" {
		state := RefreshHaboAccount()
		if state.Connected {
			return state
		}
		// Refresh may clear a revoked or expired token. Reload before deciding
		// whether the existing token can safely be reused for a new pairing.
		token = loadHaboToken()
	}
	if token == "" {
		var err error
		token, err = newHaboToken()
		if err != nil {
			return HaboAccountState{Error: "Could not create a secure device token."}
		}
		if err := saveHaboToken(token); err != nil {
			return HaboAccountState{Error: "Could not save the Habo Client device token."}
		}
	}

	body, _ := json.Marshal(map[string]string{
		"token":      token,
		"deviceName": haboDeviceName(),
		"platform":   haboPlatform(),
		"version":    version,
	})
	req, err := http.NewRequest("POST", haboHubURL()+"/api/client/pair/start", bytes.NewReader(body))
	if err != nil {
		return HaboAccountState{Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("habo-client/%s", version))

	resp, err := haboHTTP.Do(req)
	if err != nil {
		return HaboAccountState{Error: "Could not reach The Habo Hub."}
	}
	defer resp.Body.Close()

	var payload haboPairStartResponse
	if json.NewDecoder(resp.Body).Decode(&payload) != nil {
		return HaboAccountState{Error: "The Habo Hub returned an unreadable response."}
	}
	if resp.StatusCode == http.StatusOK && payload.Connected {
		SetPrivateIngestBaseURLs("habo+" + strings.TrimRight(payload.PrivateIngestURL, "/"))
		return HaboAccountState{
			Connected:        true,
			DisplayName:      payload.DisplayName,
			PrivateIngestURL: payload.PrivateIngestURL,
		}
	}
	if resp.StatusCode != http.StatusCreated {
		if payload.Error == "" {
			payload.Error = fmt.Sprintf("The Habo Hub returned HTTP %d.", resp.StatusCode)
		}
		return HaboAccountState{Error: payload.Error}
	}

	return HaboAccountState{
		Pairing:    true,
		ConnectURL: payload.ConnectURL,
		PairCode:   payload.PairCode,
	}
}

func DisconnectHaboAccount() HaboAccountState {
	token := loadHaboToken()
	if token != "" {
		req, err := http.NewRequest("DELETE", haboHubURL()+"/api/client/session", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("User-Agent", fmt.Sprintf("habo-client/%s", version))
			resp, requestErr := haboHTTP.Do(req)
			if requestErr == nil && resp != nil {
				resp.Body.Close()
			}
		}
	}
	if err := clearHaboToken(); err != nil {
		return HaboAccountState{Error: "Disconnected, but the saved device token could not be removed."}
	}
	return HaboAccountState{}
}
