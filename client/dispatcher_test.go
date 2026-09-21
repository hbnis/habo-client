package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ao-data/albiondata-client/internal/dashboard"
	"github.com/ao-data/albiondata-client/lib"
)

// TestCreateUploaders_ReusesUploaderForSameTarget guards against the
// memory leak where every upload call constructed brand new uploaders
// (a fresh NATS connection that was never closed, a fresh http.Transport
// whose idle connections were never reclaimed) instead of reusing one
// per ingest target.
func TestCreateUploaders_ReusesUploaderForSameTarget(t *testing.T) {
	target := "https://example.com/TestCreateUploaders_ReusesUploaderForSameTarget"

	first := createUploaders([]string{target})
	second := createUploaders([]string{target})

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("got %d and %d uploaders, want 1 and 1", len(first), len(second))
	}
	if first[0] != second[0] {
		t.Fatal("createUploaders returned a different uploader instance for the same target")
	}
}

func TestSendMsgToPublicUploaders_IncrementsCounterByRecordCount(t *testing.T) {
	// No configured ingest targets: createUploaders returns nothing for
	// both public and private, so this exercises the counter increment
	// without making any network calls.
	ConfigGlobal.PublicIngestBaseUrls = ""
	ConfigGlobal.PrivateIngestBaseUrls = ""

	before := dashboard.GetUploadCounts()["marketorders.ingest"]

	sendMsgToPublicUploaders(struct{}{}, lib.NatsMarketOrdersIngest, &albionState{}, "test-id", 50)

	after := dashboard.GetUploadCounts()["marketorders.ingest"]
	if after != before+50 {
		t.Fatalf("marketorders.ingest counter = %d, want %d", after, before+50)
	}
}


func TestPrivateModeDoesNotSendMarketDataToPublicAODP(t *testing.T) {
	publicRequests := make(chan struct{}, 1)
	privateRequests := make(chan struct{}, 1)

	publicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		publicRequests <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer publicServer.Close()

	privateServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		privateRequests <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer privateServer.Close()

	oldPublic := ConfigGlobal.PublicIngestBaseUrls
	oldPrivate := getPrivateIngestBaseURLs()
	oldDisable := ConfigGlobal.DisableUpload
	oldMode := GetUploadMode()
	t.Cleanup(func() {
		ConfigGlobal.PublicIngestBaseUrls = oldPublic
		SetPrivateIngestBaseURLs(oldPrivate)
		ConfigGlobal.DisableUpload = oldDisable
		SetUploadMode(UploadModePublic)
		if oldMode == UploadModePrivate && oldPrivate != "" {
			SetUploadMode(UploadModePrivate)
		}
	})

	ConfigGlobal.DisableUpload = false
	ConfigGlobal.PublicIngestBaseUrls = publicServer.URL
	SetPrivateIngestBaseURLs(privateServer.URL)
	if !SetUploadMode(UploadModePrivate) {
		t.Fatal("private mode should enable with a configured private target")
	}

	sendMsgToPublicUploaders(
		lib.MarketUpload{Orders: []*lib.MarketOrder{{ID: 1, ItemID: "T4_BOW", LocationID: "1", QualityLevel: 1, Price: 100, Amount: 1, AuctionType: "offer"}}},
		lib.NatsMarketOrdersIngest,
		&albionState{AODataServerID: 3},
		"private-routing-test",
		1,
	)

	select {
	case <-privateRequests:
	case <-time.After(time.Second):
		t.Fatal("private market data was not sent to the private target")
	}

	select {
	case <-publicRequests:
		t.Fatal("private mode leaked market data to the public AODP target")
	case <-time.After(100 * time.Millisecond):
	}
}
