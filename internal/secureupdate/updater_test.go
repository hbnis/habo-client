package secureupdate

import (
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"bytes"
)

func TestVerifySignature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("signed updater payload")
	signature := ed25519.Sign(privateKey, payload)
	signatureText := []byte(base64.StdEncoding.EncodeToString(signature))

	if err := verifySignature(publicKey, payload, signatureText); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}

	payload[0] ^= 0xff
	if err := verifySignature(publicKey, payload, signatureText); err == nil {
		t.Fatal("tampered payload accepted")
	}
}

func TestDecompressUpdate(t *testing.T) {
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	want := []byte("habo binary")
	if _, err := gz.Write(want); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := decompressUpdate(compressed.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("decompressed bytes = %q, want %q", got, want)
	}
}

func TestAllowedGitHubURL(t *testing.T) {
	updater, err := NewUpdater("1.0.0", "hbnis", "habo-client", "update-", base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		t.Fatal(err)
	}
	if updater.HTTPClient == nil {
		t.Fatal("HTTP client not configured")
	}
}
