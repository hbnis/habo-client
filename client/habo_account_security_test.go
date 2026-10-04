package client

import "testing"

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
