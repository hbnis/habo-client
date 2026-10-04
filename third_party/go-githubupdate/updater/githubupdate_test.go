package updater

import (
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
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
