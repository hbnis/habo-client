package client

import (
	"testing"

	"github.com/ao-data/albiondata-client/lib"
)

func TestHaboUploadModeRequiresPrivateTarget(t *testing.T) {
	originalPrivate := ConfigGlobal.PrivateIngestBaseUrls
	defer func() {
		ConfigGlobal.PrivateIngestBaseUrls = originalPrivate
		SetUploadMode(UploadModePublic)
	}()

	ConfigGlobal.PrivateIngestBaseUrls = ""
	SetUploadMode(UploadModePublic)

	if SetUploadMode(UploadModePrivate) {
		t.Fatal("private mode should not enable without a private ingest target")
	}
	if got := GetUploadMode(); got != UploadModePublic {
		t.Fatalf("upload mode = %q, want %q", got, UploadModePublic)
	}

	ConfigGlobal.PrivateIngestBaseUrls = "https://habonis.com/api/client/ingest"
	if !SetUploadMode(UploadModePrivate) {
		t.Fatal("private mode should enable when a private ingest target is configured")
	}
	if got := GetUploadMode(); got != UploadModePrivate {
		t.Fatalf("upload mode = %q, want %q", got, UploadModePrivate)
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
