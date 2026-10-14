package updater

import (
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"runtime"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("signed update payload")
	signature := ed25519.Sign(privateKey, payload)
	encoded := []byte(base64.StdEncoding.EncodeToString(signature))

	if err := verifySignature(publicKey, payload, encoded); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}

	tampered := append([]byte(nil), payload...)
	tampered[0] ^= 0xff
	if err := verifySignature(publicKey, tampered, encoded); err == nil {
		t.Fatal("tampered payload accepted")
	}
}

func TestDecompressUpdate(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	want := []byte("habo update binary")
	if _, err := writer.Write(want); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := decompressUpdate(compressed.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("decompressed payload = %q, want %q", got, want)
	}
}


func TestAllowedGitHubURL(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{raw: "https://api.github.com/repos/hbnis/habo-client/releases/latest", want: true},
		{raw: "https://github.com/hbnis/habo-client/releases/download/v0.1.7/update-linux-amd64.gz", want: true},
		{raw: "https://release-assets.githubusercontent.com/example", want: true},
		{raw: "http://github.com/hbnis/habo-client", want: false},
		{raw: "https://github.com.evil.example/hbnis/habo-client", want: false},
		{raw: "https://user:pass@github.com/hbnis/habo-client", want: false},
		{raw: "https://evil.example/update.gz", want: false},
	}

	for _, tt := range tests {
		parsed, err := url.Parse(tt.raw)
		if err != nil {
			t.Fatalf("url.Parse(%q): %v", tt.raw, err)
		}
		if got := allowedGitHubURL(parsed); got != tt.want {
			t.Fatalf("allowedGitHubURL(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestUpdaterRefusesUnsignedRelease(t *testing.T) {
	u := NewUpdater("0.1.0", "hbnis", "habo-client", "update-")
	payloadName := "update-" + runtime.GOOS + "-" + runtime.GOARCH + ".gz"
	if runtime.GOOS == "windows" {
		payloadName = "update-" + runtime.GOOS + "-" + runtime.GOARCH + ".exe.gz"
	}
	u.latestRelease = &releaseResponse{
		TagName: "v0.1.1",
		Assets: []releaseAsset{{
			Name:               payloadName,
			BrowserDownloadURL: "https://github.com/hbnis/habo-client/releases/download/v0.1.1/" + payloadName,
		}},
	}

	if err := u.Update(); !errors.Is(err, ErrorNoSignature) {
		t.Fatalf("Update() error = %v, want ErrorNoSignature", err)
	}
}
