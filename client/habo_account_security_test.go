package client

import (
	"os"
	"testing"
)

func TestValidateHaboBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "production", input: "https://habonis.com", wantErr: false},
		{name: "production subdomain", input: "https://api.habonis.com", wantErr: false},
		{name: "production http rejected", input: "http://habonis.com", wantErr: true},
		{name: "foreign host rejected", input: "https://example.com", wantErr: true},
		{name: "foreign suffix rejected", input: "https://habonis.com.example.com", wantErr: true},
		{name: "credentials rejected", input: "https://user:pass@habonis.com", wantErr: true},
		{name: "path rejected", input: "https://habonis.com/other", wantErr: true},
		{name: "localhost http allowed", input: "http://localhost:3000", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateHaboBaseURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateHaboBaseURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestTrustedHaboResponseURLRejectsOriginChange(t *testing.T) {
	t.Setenv("HABO_HUB_URL", "https://habonis.com")

	if _, err := trustedHaboResponseURL("https://evil.example/api/client/ingest", "/api/client/ingest", false); err == nil {
		t.Fatal("accepted foreign ingest origin")
	}
	if _, err := trustedHaboResponseURL("https://habonis.com/api/client/ingest", "/api/client/ingest", false); err != nil {
		t.Fatalf("rejected trusted ingest URL: %v", err)
	}
	if _, err := trustedHaboResponseURL("https://habonis.com/client/connect?code=ABCDE-23456", "/client/connect", true); err != nil {
		t.Fatalf("rejected trusted pairing URL: %v", err)
	}
}


func TestRestoreHaboTokenFromRestartEnvironment(t *testing.T) {
	token := "abcdefghijklmnopqrstuvwxyzABCDEFGH1234567890"
	t.Setenv(haboRestartTokenEnvKey, token)

	var saved string
	restored, err := restoreHaboTokenFromRestartEnvironment(func(value string) error {
		saved = value
		return nil
	})
	if err != nil {
		t.Fatalf("restore returned error: %v", err)
	}
	if !restored {
		t.Fatal("expected restart token to be restored")
	}
	if saved != token {
		t.Fatalf("saved token = %q, want %q", saved, token)
	}
	if got := os.Getenv(haboRestartTokenEnvKey); got != "" {
		t.Fatalf("restart token environment was not cleared: %q", got)
	}
}

func TestRestoreHaboTokenFromRestartEnvironmentRejectsInvalidToken(t *testing.T) {
	t.Setenv(haboRestartTokenEnvKey, "short")

	called := false
	restored, err := restoreHaboTokenFromRestartEnvironment(func(value string) error {
		called = true
		return nil
	})
	if err == nil {
		t.Fatal("expected invalid restart token error")
	}
	if restored {
		t.Fatal("invalid restart token must not be reported restored")
	}
	if called {
		t.Fatal("invalid restart token must not be persisted")
	}
	if got := os.Getenv(haboRestartTokenEnvKey); got != "" {
		t.Fatalf("invalid restart token environment was not cleared: %q", got)
	}
}

func TestHaboUpdateRestartEnvironmentUsesCurrentToken(t *testing.T) {
	setTestHaboToken(t, "restart-token-abcdefghijklmnopqrstuvwxyz123456")
	got := HaboUpdateRestartEnvironment()
	want := haboRestartTokenEnvKey + "=restart-token-abcdefghijklmnopqrstuvwxyz123456"
	if got != want {
		t.Fatalf("restart environment = %q, want %q", got, want)
	}
}
