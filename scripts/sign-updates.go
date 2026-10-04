package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fatal(errors.New("usage: go run ./scripts/sign-updates.go <file> [file ...]"))
	}

	encoded := strings.TrimSpace(os.Getenv("HABO_UPDATE_SIGNING_KEY"))
	if encoded == "" {
		fatal(errors.New("HABO_UPDATE_SIGNING_KEY is not set"))
	}
	keyBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		fatal(fmt.Errorf("decode signing key: %w", err))
	}

	var privateKey ed25519.PrivateKey
	switch len(keyBytes) {
	case ed25519.SeedSize:
		privateKey = ed25519.NewKeyFromSeed(keyBytes)
	case ed25519.PrivateKeySize:
		privateKey = ed25519.PrivateKey(keyBytes)
	default:
		fatal(fmt.Errorf("invalid signing key length: got %d bytes", len(keyBytes)))
	}

	for _, path := range os.Args[1:] {
		payload, err := os.ReadFile(path)
		if err != nil {
			fatal(fmt.Errorf("read %s: %w", path, err))
		}
		signature := ed25519.Sign(privateKey, payload)
		encodedSignature := base64.StdEncoding.EncodeToString(signature) + "\n"
		if err := os.WriteFile(path+".sig", []byte(encodedSignature), 0644); err != nil {
			fatal(fmt.Errorf("write %s.sig: %w", path, err))
		}
		fmt.Printf("signed %s\n", path)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
