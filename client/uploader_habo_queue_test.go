package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ao-data/albiondata-client/lib"
)

func setTestHaboToken(t *testing.T, token string) {
	t.Helper()
	haboMu.Lock()
	old := haboToken
	haboToken = token
	haboMu.Unlock()
	t.Cleanup(func() {
		haboMu.Lock()
		haboToken = old
		haboMu.Unlock()
	})
}

func testOrderPayload(t *testing.T, id int) []byte {
	t.Helper()
	body := map[string]any{
		"Orders": []map[string]any{{
			"Id":               id,
			"ItemTypeId":       "T4_BOW",
			"ItemGroupTypeId":  "",
			"LocationId":       "1002",
			"QualityLevel":     1,
			"EnchantmentLevel": 0,
			"UnitPriceSilver":  1000000,
			"Amount":           1,
			"AuctionType":      "offer",
			"Expires":          "",
		}},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func testHistoryPayload(t *testing.T, albionID int) []byte {
	t.Helper()
	body := map[string]any{
		"AlbionId":     albionID,
		"LocationId":   "1002",
		"QualityLevel": 1,
		"Timescale":    0,
		"MarketHistories": []map[string]any{{
			"ItemAmount":   2,
			"SilverAmount": 1000000,
			"Timestamp":    638960000000000000,
		}},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestHaboUploadQueueBatchesOrdersAndPreservesCaptureTimes(t *testing.T) {
	setTestHaboToken(t, "queue-test-token")
	var requests atomic.Int32
	var received queuedOrderBatch

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer queue-test-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("X-Habo-Server-Id"); got != "1" {
			t.Errorf("server id header = %q, want 1", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	q := newHaboUploadQueue(server.URL, server.Client(), t.TempDir(), false)
	first := time.Now().Add(-2 * time.Second).UTC()
	second := first.Add(time.Second)

	if err := q.enqueue(testOrderPayload(t, 1), lib.NatsMarketOrdersIngest, 1, "first", "queue-test-token", first); err != nil {
		t.Fatal(err)
	}
	if err := q.enqueue(testOrderPayload(t, 2), lib.NatsMarketOrdersIngest, 1, "second", "queue-test-token", second); err != nil {
		t.Fatal(err)
	}

	q.processOnce()

	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1 batched request", got)
	}
	if len(received.Batches) != 2 {
		t.Fatalf("batch envelopes = %d, want 2", len(received.Batches))
	}
	if received.Batches[0].CapturedAt != first.Format(time.RFC3339Nano) {
		t.Fatalf("first CapturedAt = %q", received.Batches[0].CapturedAt)
	}
	if received.Batches[1].CapturedAt != second.Format(time.RFC3339Nano) {
		t.Fatalf("second CapturedAt = %q", received.Batches[1].CapturedAt)
	}
	if q.depth() != 0 {
		t.Fatalf("queue depth = %d, want 0 after success", q.depth())
	}
	stats := q.snapshotStats()
	if stats.SentBatches != 1 || stats.SentRows != 2 {
		t.Fatalf("stats = %+v, want one batch and two rows", stats)
	}
}

func TestHaboUploadQueueBatchesHistoryEnvelopes(t *testing.T) {
	setTestHaboToken(t, "history-token")
	var received queuedHistoryBatch

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	q := newHaboUploadQueue(server.URL, server.Client(), t.TempDir(), false)
	now := time.Now().UTC()
	if err := q.enqueue(testHistoryPayload(t, 101), lib.NatsMarketHistoriesIngest, 3, "history-1", "history-token", now.Add(-2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := q.enqueue(testHistoryPayload(t, 102), lib.NatsMarketHistoriesIngest, 3, "history-2", "history-token", now.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}

	q.processOnce()

	if len(received.Batches) != 2 {
		t.Fatalf("history envelopes = %d, want 2", len(received.Batches))
	}
	if received.Batches[0].AlbionID != 101 || received.Batches[1].AlbionID != 102 {
		t.Fatalf("history AlbionIds = %d, %d", received.Batches[0].AlbionID, received.Batches[1].AlbionID)
	}
	if q.depth() != 0 {
		t.Fatalf("queue depth = %d, want 0", q.depth())
	}
}

func TestHaboUploadQueueRespectsRetryAfterAndRetries429(t *testing.T) {
	setTestHaboToken(t, "retry-token")
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requests.Add(1)
		if count == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	q := newHaboUploadQueue(server.URL, server.Client(), t.TempDir(), false)
	if err := q.enqueue(testOrderPayload(t, 5), lib.NatsMarketOrdersIngest, 1, "retry", "retry-token", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	q.processOnce()
	if q.depth() != 1 {
		t.Fatalf("queue depth after 429 = %d, want 1", q.depth())
	}
	q.mu.Lock()
	job := q.jobs[0]
	if job.Attempts != 1 {
		t.Fatalf("attempts = %d, want 1", job.Attempts)
	}
	if time.Until(job.NextAttemptAt) < 500*time.Millisecond {
		t.Fatalf("Retry-After was not respected: next=%v", job.NextAttemptAt)
	}
	job.NextAttemptAt = time.Time{}
	q.nextSend = time.Time{}
	q.mu.Unlock()

	q.processOnce()
	if got := requests.Load(); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
	if q.depth() != 0 {
		t.Fatalf("queue depth after retry success = %d, want 0", q.depth())
	}
	if q.snapshotStats().RateLimitRetries != 1 {
		t.Fatalf("rate limit retry stats = %+v", q.snapshotStats())
	}
}

func TestHaboUploadQueueRetriesServerErrorWithBackoff(t *testing.T) {
	setTestHaboToken(t, "server-retry-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	q := newHaboUploadQueue(server.URL, server.Client(), t.TempDir(), false)
	if err := q.enqueue(testOrderPayload(t, 6), lib.NatsMarketOrdersIngest, 1, "server-retry", "server-retry-token", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	q.processOnce()

	if q.depth() != 1 {
		t.Fatalf("queue depth = %d, want retry retained", q.depth())
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.jobs[0].Attempts != 1 {
		t.Fatalf("attempts = %d, want 1", q.jobs[0].Attempts)
	}
	if time.Until(q.jobs[0].NextAttemptAt) < time.Second {
		t.Fatalf("backoff too short: %v", q.jobs[0].NextAttemptAt)
	}
}

func TestHaboUploadQueueDropsNonRetryable400(t *testing.T) {
	setTestHaboToken(t, "bad-request-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	q := newHaboUploadQueue(server.URL, server.Client(), t.TempDir(), false)
	if err := q.enqueue(testOrderPayload(t, 7), lib.NatsMarketOrdersIngest, 1, "bad", "bad-request-token", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	q.processOnce()

	if q.depth() != 0 {
		t.Fatalf("queue depth = %d, want 0 after non-retryable 400", q.depth())
	}
	if q.snapshotStats().DroppedPermanent != 1 {
		t.Fatalf("drop stats = %+v", q.snapshotStats())
	}
}

func TestHaboUploadQueueDropsJobsAfterAccountTokenChanges(t *testing.T) {
	setTestHaboToken(t, "old-token")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	q := newHaboUploadQueue(server.URL, server.Client(), t.TempDir(), false)
	if err := q.enqueue(testOrderPayload(t, 8), lib.NatsMarketOrdersIngest, 1, "old-account", "old-token", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	haboMu.Lock()
	haboToken = "new-token"
	haboMu.Unlock()
	q.processOnce()

	if got := requests.Load(); got != 0 {
		t.Fatalf("requests = %d, old-account job must not upload under new token", got)
	}
	if q.depth() != 0 {
		t.Fatalf("queue depth = %d, mismatched-account job should be dropped", q.depth())
	}
	if q.snapshotStats().DroppedAccount != 1 {
		t.Fatalf("account drop stats = %+v", q.snapshotStats())
	}
}

func TestHaboUploadQueuePersistsAndReloadsJobs(t *testing.T) {
	dir := t.TempDir()
	q := newHaboUploadQueue("https://example.invalid", &http.Client{}, dir, false)
	captured := time.Now().UTC()
	if err := q.enqueue(testOrderPayload(t, 9), lib.NatsMarketOrdersIngest, 2, "persist", "persist-token", captured); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".json") {
		t.Fatalf("queue files = %v, want one json file", entries)
	}

	reloaded := newHaboUploadQueue("https://example.invalid", &http.Client{}, dir, false)
	if reloaded.depth() != 1 {
		t.Fatalf("reloaded depth = %d, want 1", reloaded.depth())
	}
	reloaded.mu.Lock()
	job := reloaded.jobs[0]
	reloaded.mu.Unlock()
	if job.ServerID != 2 || job.TokenFingerprint != haboTokenFingerprint("persist-token") {
		t.Fatalf("reloaded job lost routing/account context: %+v", job)
	}
	if job.CapturedAt != captured.Format(time.RFC3339Nano) {
		t.Fatalf("CapturedAt = %q", job.CapturedAt)
	}
}

func TestHaboUploadQueueExpiresStaleOrdersButKeepsHistory(t *testing.T) {
	dir := t.TempDir()
	q := newHaboUploadQueue("https://example.invalid", &http.Client{}, dir, false)
	now := time.Now().UTC()

	if err := q.enqueue(testOrderPayload(t, 10), lib.NatsMarketOrdersIngest, 1, "stale-order", "ttl-token", now.Add(-16*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if q.depth() != 0 {
		t.Fatalf("stale order depth = %d, want 0", q.depth())
	}

	if err := q.enqueue(testHistoryPayload(t, 103), lib.NatsMarketHistoriesIngest, 1, "history", "ttl-token", now.Add(-16*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if q.depth() != 1 {
		t.Fatalf("history depth = %d, want 1", q.depth())
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("queue files = %d, want one history job", len(files))
	}
}

func TestHaboRetryBackoffCapsAtOneMinute(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 32 * time.Second},
		{6, time.Minute},
		{20, time.Minute},
	}
	for _, tc := range cases {
		if got := haboRetryBackoff(tc.attempt); got != tc.want {
			t.Fatalf("attempt %d backoff = %v, want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestHaboQueueSelectsThreeOrderBatchesThenHistory(t *testing.T) {
	q := newHaboUploadQueue("https://example.invalid", &http.Client{}, "", false)
	now := time.Now().UTC()
	token := "priority-token"
	fingerprint := haboTokenFingerprint(token)

	for i := 0; i < 4; i++ {
		if err := q.enqueue(testOrderPayload(t, 20+i), lib.NatsMarketOrdersIngest, 1, "order", token, now.Add(time.Duration(i)*time.Millisecond)); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.enqueue(testHistoryPayload(t, 200), lib.NatsMarketHistoriesIngest, 1, "history", token, now.Add(5*time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	// Use a row limit effect to force one order job into each synthetic turn.
	q.mu.Lock()
	for _, job := range q.jobs {
		if job.Topic == lib.NatsMarketOrdersIngest {
			job.RowCount = haboUploadMaxOrderRows
		}
	}
	for turn := 1; turn <= 3; turn++ {
		selected := q.selectBatchLocked(time.Now(), fingerprint)
		if len(selected) != 1 || selected[0].Topic != lib.NatsMarketOrdersIngest {
			q.mu.Unlock()
			t.Fatalf("turn %d selected %+v, want one order job", turn, selected)
		}
		selected[0].InFlight = false
		selected[0].NextAttemptAt = time.Now().Add(time.Hour)
	}
	selected := q.selectBatchLocked(time.Now(), fingerprint)
	q.mu.Unlock()

	if len(selected) != 1 || selected[0].Topic != lib.NatsMarketHistoriesIngest {
		t.Fatalf("fourth turn selected %+v, want history", selected)
	}
}
