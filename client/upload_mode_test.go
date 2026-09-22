package client

import (
	"testing"

	"github.com/ao-data/albiondata-client/lib"
)

func TestHaboUploadModeIsPrivateOnly(t *testing.T) {
	originalPrivate := ConfigGlobal.PrivateIngestBaseUrls
	defer func() {
		ConfigGlobal.PrivateIngestBaseUrls = originalPrivate
		SetUploadMode(UploadModePrivate)
	}()

	ConfigGlobal.PrivateIngestBaseUrls = ""
	if !SetUploadMode(UploadModePrivate) {
		t.Fatal("private mode should remain the Habo Client scan mode even before an ingest target is available")
	}
	if got := GetUploadMode(); got != UploadModePrivate {
		t.Fatalf("upload mode = %q, want %q", got, UploadModePrivate)
	}

	if SetUploadMode(UploadModePublic) {
		t.Fatal("public mode must not be enabled in the Habo build")
	}
	if got := GetUploadMode(); got != UploadModePrivate {
		t.Fatalf("upload mode changed to %q after public-mode request, want %q", got, UploadModePrivate)
	}

	ConfigGlobal.PrivateIngestBaseUrls = "https://habonis.com/api/client/ingest"
	if !PrivateUploadConfigured() {
		t.Fatal("private upload target should be detected when configured")
	}
}

func TestHaboMarketTopicFilter(t *testing.T) {
	if !isHaboMarketTopic(lib.NatsMarketOrdersIngest) {
		t.Fatal("market orders must be allowed")
	}
	if !isHaboMarketTopic(lib.NatsMarketHistoriesIngest) {
		t.Fatal("market histories must be allowed")
	}
	if isHaboMarketTopic(lib.NatsGoldPricesIngest) {
		t.Fatal("gold prices must not be forwarded by Habo Client")
	}
}
