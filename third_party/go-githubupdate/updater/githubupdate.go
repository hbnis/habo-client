package updater

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/minio/selfupdate"
)

const (
	updatePublicKeyBase64   = "8EieXkARx1MVEUmPwAFA9uIunzxZKk3l5W1lTJFBptM="
	maxReleaseResponseSize  = 2 << 20
	maxCompressedUpdateSize = 100 << 20
	maxExecutableSize       = 250 << 20
)

var (
	ErrorNoBinary    = errors.New("No binary for the update found")
	ErrorNoSignature = errors.New("No signature for the update found")
)

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type releaseResponse struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

// Updater preserves the original go-githubupdate API used by Habo Client, but
// adds detached Ed25519 signature verification before self-update is allowed.
type Updater struct {
	CurrentVersion string
	GithubOwner    string
	GithubRepo     string
	FilePrefix     string
	latestRelease  *releaseResponse
	publicKey      ed25519.PublicKey
	httpClient     *http.Client
}

func NewUpdater(currentVersion, githubOwner, githubRepo, filePrefix string) *Updater {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(updatePublicKeyBase64)
	if err != nil || len(publicKeyBytes) != ed25519.PublicKeySize {
		panic("invalid Habo Client update public key")
	}

	client := &http.Client{
		Timeout: 90 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects while downloading update")
			}
			if !allowedGitHubURL(req.URL) {
				return fmt.Errorf("refusing update redirect to untrusted URL: %s", req.URL.Redacted())
			}
			return nil
		},
	}

	return &Updater{
		CurrentVersion: currentVersion,
		GithubOwner:    githubOwner,
		GithubRepo:     githubRepo,
		FilePrefix:     filePrefix,
		publicKey:      ed25519.PublicKey(publicKeyBytes),
		httpClient:     client,
	}
}

func (u *Updater) BackgroundUpdater() (bool, error) {
	available, err := u.CheckUpdateAvailable()
	if err != nil {
		return false, err
	}
	if available == "" {
		return false, nil
	}
	fmt.Printf("Version %s available, verifying signed update now.\n", available)
	if err := u.Update(); err != nil {
		return false, err
	}
	return true, nil
}

func normalizeVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}

func (u *Updater) CheckUpdateAvailable() (string, error) {
	if !validGitHubName(u.GithubOwner) || !validGitHubName(u.GithubRepo) {
		return "", errors.New("invalid GitHub update repository")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", u.GithubOwner, u.GithubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "habo-client-secure-updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub release check returned HTTP %d", resp.StatusCode)
	}

	var release releaseResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxReleaseResponseSize)).Decode(&release); err != nil {
		return "", fmt.Errorf("decode GitHub release: %w", err)
	}
	if strings.TrimSpace(release.TagName) == "" {
		return "", errors.New("latest GitHub release has no tag")
	}
	u.latestRelease = &release

	current, err := semver.Make(normalizeVersion(u.CurrentVersion))
	if err != nil {
		return "", fmt.Errorf("parse current version: %w", err)
	}
	latest, err := semver.Make(normalizeVersion(release.TagName))
	if err != nil {
		return "", fmt.Errorf("parse release version: %w", err)
	}
	if current.LT(latest) {
		return release.TagName, nil
	}
	return "", nil
}

func (u *Updater) Update() error {
	if u.latestRelease == nil {
		return errors.New("update called before release metadata was loaded")
	}

	payloadName := u.FilePrefix + runtime.GOOS + "-" + runtime.GOARCH + ".gz"
	if runtime.GOOS == "windows" {
		payloadName = u.FilePrefix + runtime.GOOS + "-" + runtime.GOARCH + ".exe.gz"
	}
	signatureName := payloadName + ".sig"

	var payloadAsset, signatureAsset *releaseAsset
	for i := range u.latestRelease.Assets {
		asset := &u.latestRelease.Assets[i]
		switch asset.Name {
		case payloadName:
			payloadAsset = asset
		case signatureName:
			signatureAsset = asset
		}
	}
	if payloadAsset == nil {
		return ErrorNoBinary
	}
	if signatureAsset == nil {
		return ErrorNoSignature
	}

	payload, err := u.fetchAsset(payloadAsset.BrowserDownloadURL, maxCompressedUpdateSize)
	if err != nil {
		return fmt.Errorf("download update payload: %w", err)
	}
	signatureText, err := u.fetchAsset(signatureAsset.BrowserDownloadURL, 4096)
	if err != nil {
		return fmt.Errorf("download update signature: %w", err)
	}
	if err := verifySignature(u.publicKey, payload, signatureText); err != nil {
		return err
	}

	binary, err := decompressUpdate(payload)
	if err != nil {
		return err
	}
	if err := selfupdate.Apply(bytes.NewReader(binary), selfupdate.Options{}); err != nil {
		if rollbackErr := selfupdate.RollbackError(err); rollbackErr != nil {
			return fmt.Errorf("update failed and rollback also failed: %v", rollbackErr)
		}
		return fmt.Errorf("update failed: %w", err)
	}
	fmt.Println("Signed update installed successfully.")
	return nil
}

func verifySignature(publicKey ed25519.PublicKey, payload, signatureText []byte) error {
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signatureText)))
	if err != nil {
		return fmt.Errorf("decode update signature: %w", err)
	}
	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid update signature length: got %d, want %d", len(signature), ed25519.SignatureSize)
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return errors.New("update signature verification failed")
	}
	return nil
}

func decompressUpdate(payload []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("open update gzip: %w", err)
	}
	defer gz.Close()

	binary, err := io.ReadAll(io.LimitReader(gz, maxExecutableSize+1))
	if err != nil {
		return nil, fmt.Errorf("decompress update: %w", err)
	}
	if int64(len(binary)) > maxExecutableSize {
		return nil, errors.New("decompressed update exceeds size limit")
	}
	return binary, nil
}

func (u *Updater) fetchAsset(rawURL string, maxBytes int64) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || !allowedGitHubURL(parsed) {
		return nil, errors.New("refusing untrusted update asset URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "habo-client-secure-updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("asset download returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errors.New("update asset exceeds size limit")
	}
	return data, nil
}

func validGitHubName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func allowedGitHubURL(u *url.URL) bool {
	if u == nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "github.com" || host == "api.github.com" || host == "objects.githubusercontent.com" || host == "release-assets.githubusercontent.com" || strings.HasSuffix(host, ".githubusercontent.com")
}
