package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ao-data/albiondata-client/lib"
	"github.com/ao-data/albiondata-client/log"
)

const (
	haboUploadQueueMaxJobs       = 2000
	haboUploadQueueMaxBytes      = 25 * 1024 * 1024
	haboUploadOrderTTL           = 15 * time.Minute
	haboUploadHistoryTTL         = 24 * time.Hour
	haboUploadSendInterval       = time.Second
	haboUploadMaxEnvelopes       = 64
	haboUploadMaxOrderRows       = 500
	haboUploadMaxHistoryRows     = 2000
	haboUploadTargetBodyBytes    = 700 * 1024
	haboUploadAuthPause          = 15 * time.Minute
	haboUploadRetryAfterFallback = 60 * time.Second
)

type haboUploadJob struct {
	ID               string          `json:"id"`
	Topic            string          `json:"topic"`
	ServerID         int             `json:"serverId"`
	Identifier       string          `json:"identifier,omitempty"`
	RowCount         int             `json:"rowCount"`
	CapturedAt       string          `json:"capturedAt"`
	TokenFingerprint string          `json:"tokenFingerprint"`
	Payload          json.RawMessage `json:"payload"`

	Attempts      int       `json:"-"`
	NextAttemptAt time.Time `json:"-"`
	FilePath      string    `json:"-"`
	SizeBytes     int64     `json:"-"`
	InFlight      bool      `json:"-"`
}

type haboUploadQueueStats struct {
	SentBatches      int64
	SentRows         int64
	RateLimitRetries int64
	NetworkRetries   int64
	ServerRetries    int64
	DroppedExpired   int64
	DroppedAccount   int64
	DroppedPermanent int64
}

type haboUploadQueue struct {
	baseURL string
	client  *http.Client
	dir     string

	mu                sync.Mutex
	jobs              []*haboUploadJob
	diskBytes         int64
	wake              chan struct{}
	nextSend          time.Time
	ordersSinceHistory int
	sendInterval      time.Duration
	stats             haboUploadQueueStats
}

type queuedOrderPayload struct {
	Orders []json.RawMessage `json:"Orders"`
}

type queuedHistoryPayload struct {
	AlbionID       int               `json:"AlbionId"`
	LocationID     string            `json:"LocationId"`
	QualityLevel   int               `json:"QualityLevel"`
	Timescale      int               `json:"Timescale"`
	MarketHistories []json.RawMessage `json:"MarketHistories"`
}

type queuedOrderEnvelope struct {
	Orders     []json.RawMessage `json:"Orders"`
	CapturedAt string            `json:"CapturedAt"`
}

type queuedOrderBatch struct {
	Batches []queuedOrderEnvelope `json:"Batches"`
}

type queuedHistoryEnvelope struct {
	AlbionID        int               `json:"AlbionId"`
	LocationID      string            `json:"LocationId"`
	QualityLevel    int               `json:"QualityLevel"`
	Timescale       int               `json:"Timescale"`
	MarketHistories []json.RawMessage `json:"MarketHistories"`
	CapturedAt      string            `json:"CapturedAt"`
}

type queuedHistoryBatch struct {
	Batches []queuedHistoryEnvelope `json:"Batches"`
}

func haboUploadQueueDir() (string, error) {
	configPath, err := haboConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), "upload-queue"), nil
}

func haboTokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func haboUploadJobID(topic, identifier string, capturedAt time.Time) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d", topic, identifier, capturedAt.UnixNano())))
	return hex.EncodeToString(sum[:12])
}

func newHaboUploadQueue(baseURL string, httpClient *http.Client, dir string, startWorker bool) *haboUploadQueue {
	q := &haboUploadQueue{
		baseURL:      strings.TrimRight(baseURL, "/"),
		client:       httpClient,
		dir:          dir,
		wake:         make(chan struct{}, 1),
		sendInterval: haboUploadSendInterval,
	}
	q.load()
	if startWorker {
		go q.run()
	}
	return q
}

func (q *haboUploadQueue) load() {
	if strings.TrimSpace(q.dir) == "" {
		return
	}
	if err := os.MkdirAll(q.dir, 0700); err != nil {
		log.Warnf("Could not create Habo upload queue directory: %v", err)
		return
	}

	entries, err := os.ReadDir(q.dir)
	if err != nil {
		log.Warnf("Could not read Habo upload queue: %v", err)
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	q.mu.Lock()
	defer q.mu.Unlock()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(q.dir, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var job haboUploadJob
		if json.Unmarshal(data, &job) != nil || !validQueuedTopic(job.Topic) || job.ServerID < 1 || job.ServerID > 3 || job.TokenFingerprint == "" {
			_ = os.Remove(path)
			continue
		}
		if _, parseErr := time.Parse(time.RFC3339Nano, job.CapturedAt); parseErr != nil {
			_ = os.Remove(path)
			continue
		}
		job.FilePath = path
		job.SizeBytes = int64(len(data))
		q.jobs = append(q.jobs, &job)
		q.diskBytes += job.SizeBytes
	}
	q.sortJobsLocked()
	q.pruneLocked(time.Now())
	if len(q.jobs) > 0 {
		log.Infof("Loaded %d pending Habo upload jobs from disk.", len(q.jobs))
	}
}

func validQueuedTopic(topic string) bool {
	return topic == lib.NatsMarketOrdersIngest || topic == lib.NatsMarketHistoriesIngest
}

func queuedPayloadRowCount(topic string, body []byte) (int, error) {
	switch topic {
	case lib.NatsMarketOrdersIngest:
		var payload queuedOrderPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return 0, err
		}
		return len(payload.Orders), nil
	case lib.NatsMarketHistoriesIngest:
		var payload queuedHistoryPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return 0, err
		}
		return len(payload.MarketHistories), nil
	default:
		return 0, fmt.Errorf("unsupported Habo upload topic %q", topic)
	}
}

func (q *haboUploadQueue) enqueue(body []byte, topic string, serverID int, identifier, token string, capturedAt time.Time) error {
	if !validQueuedTopic(topic) {
		return fmt.Errorf("unsupported Habo upload topic %q", topic)
	}
	if serverID < 1 || serverID > 3 {
		return fmt.Errorf("invalid Albion server id %d", serverID)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("missing Habo token")
	}
	rowCount, err := queuedPayloadRowCount(topic, body)
	if err != nil || rowCount < 1 {
		if err == nil {
			err = fmt.Errorf("payload contains no rows")
		}
		return err
	}

	job := &haboUploadJob{
		ID:               haboUploadJobID(topic, identifier, capturedAt),
		Topic:            topic,
		ServerID:         serverID,
		Identifier:       identifier,
		RowCount:         rowCount,
		CapturedAt:       capturedAt.UTC().Format(time.RFC3339Nano),
		TokenFingerprint: haboTokenFingerprint(token),
		Payload:          append(json.RawMessage(nil), body...),
	}

	if err := q.persist(job); err != nil {
		// Keep the job in memory even if the local disk is temporarily
		// unavailable. We lose restart durability, not the live upload.
		log.Warnf("Could not persist Habo upload job; keeping it in memory: %v", err)
	}

	q.mu.Lock()
	q.jobs = append(q.jobs, job)
	q.diskBytes += job.SizeBytes
	q.sortJobsLocked()
	q.pruneLocked(time.Now())
	depth := len(q.jobs)
	q.mu.Unlock()

	if depth == 1 || depth%100 == 0 {
		log.Infof("Habo upload queue depth: %d", depth)
	}
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return nil
}

func (q *haboUploadQueue) persist(job *haboUploadJob) error {
	if strings.TrimSpace(q.dir) == "" {
		return nil
	}
	if err := os.MkdirAll(q.dir, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%020d-%s.json", time.Now().UnixNano(), job.ID)
	path := filepath.Join(q.dir, name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	job.FilePath = path
	job.SizeBytes = int64(len(data))
	return nil
}

func (q *haboUploadQueue) sortJobsLocked() {
	sort.SliceStable(q.jobs, func(i, j int) bool {
		left, _ := time.Parse(time.RFC3339Nano, q.jobs[i].CapturedAt)
		right, _ := time.Parse(time.RFC3339Nano, q.jobs[j].CapturedAt)
		return left.Before(right)
	})
}

func queuedJobTTL(topic string) time.Duration {
	if topic == lib.NatsMarketOrdersIngest {
		return haboUploadOrderTTL
	}
	return haboUploadHistoryTTL
}

func (q *haboUploadQueue) pruneLocked(now time.Time) {
	for index := len(q.jobs) - 1; index >= 0; index-- {
		job := q.jobs[index]
		if job.InFlight {
			continue
		}
		capturedAt, err := time.Parse(time.RFC3339Nano, job.CapturedAt)
		if err != nil || now.Sub(capturedAt) > queuedJobTTL(job.Topic) {
			q.removeJobLocked(index)
			q.stats.DroppedExpired++
		}
	}

	for len(q.jobs) > haboUploadQueueMaxJobs || q.diskBytes > haboUploadQueueMaxBytes {
		index := q.oldestDroppableIndexLocked(lib.NatsMarketHistoriesIngest)
		if index < 0 {
			index = q.oldestDroppableIndexLocked(lib.NatsMarketOrdersIngest)
		}
		if index < 0 {
			break
		}
		q.removeJobLocked(index)
		q.stats.DroppedPermanent++
	}
}

func (q *haboUploadQueue) oldestDroppableIndexLocked(topic string) int {
	for index, job := range q.jobs {
		if !job.InFlight && job.Topic == topic {
			return index
		}
	}
	return -1
}

func (q *haboUploadQueue) removeJobLocked(index int) {
	job := q.jobs[index]
	if job.FilePath != "" {
		_ = os.Remove(job.FilePath)
	}
	q.diskBytes -= job.SizeBytes
	if q.diskBytes < 0 {
		q.diskBytes = 0
	}
	q.jobs = append(q.jobs[:index], q.jobs[index+1:]...)
}

func (q *haboUploadQueue) removeJobsByIDLocked(ids map[string]struct{}) {
	for index := len(q.jobs) - 1; index >= 0; index-- {
		if _, ok := ids[q.jobs[index].ID]; ok {
			q.removeJobLocked(index)
		}
	}
}

func (q *haboUploadQueue) dropWrongAccountJobsLocked(fingerprint string) {
	for index := len(q.jobs) - 1; index >= 0; index-- {
		job := q.jobs[index]
		if job.InFlight || job.TokenFingerprint == fingerprint {
			continue
		}
		q.removeJobLocked(index)
		q.stats.DroppedAccount++
	}
}

func (q *haboUploadQueue) run() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			q.processOnce()
		case <-q.wake:
			q.processOnce()
		}
	}
}

func (q *haboUploadQueue) processOnce() {
	token := strings.TrimSpace(GetHaboToken())
	if token == "" {
		return
	}
	fingerprint := haboTokenFingerprint(token)
	now := time.Now()

	q.mu.Lock()
	q.pruneLocked(now)
	q.dropWrongAccountJobsLocked(fingerprint)
	if now.Before(q.nextSend) {
		q.mu.Unlock()
		return
	}
	jobs := q.selectBatchLocked(now, fingerprint)
	q.mu.Unlock()
	if len(jobs) == 0 {
		return
	}

	body, usable, malformed := buildHaboBatch(jobs)
	if len(malformed) > 0 {
		ids := make(map[string]struct{}, len(malformed))
		for _, job := range malformed {
			ids[job.ID] = struct{}{}
		}
		q.mu.Lock()
		q.removeJobsByIDLocked(ids)
		q.stats.DroppedPermanent += int64(len(malformed))
		q.mu.Unlock()
	}
	if len(usable) == 0 || len(body) == 0 {
		q.releaseInFlight(jobs)
		return
	}

	for len(body) > haboUploadTargetBodyBytes && len(usable) > 1 {
		last := usable[len(usable)-1]
		q.releaseInFlight([]*haboUploadJob{last})
		usable = usable[:len(usable)-1]
		body, usable, malformed = buildHaboBatch(usable)
		if len(malformed) > 0 {
			q.dropMalformed(malformed)
		}
	}
	if len(usable) == 0 {
		return
	}

	status, retryAfter, err := q.sendBatch(token, usable, body)
	q.finishAttempt(usable, status, retryAfter, err)
}

func (q *haboUploadQueue) selectBatchLocked(now time.Time, fingerprint string) []*haboUploadJob {
	hasOrders := false
	hasHistory := false
	for _, job := range q.jobs {
		if job.InFlight || job.TokenFingerprint != fingerprint || now.Before(job.NextAttemptAt) {
			continue
		}
		if job.Topic == lib.NatsMarketOrdersIngest {
			hasOrders = true
		} else if job.Topic == lib.NatsMarketHistoriesIngest {
			hasHistory = true
		}
	}
	if !hasOrders && !hasHistory {
		return nil
	}

	topic := lib.NatsMarketOrdersIngest
	if hasHistory && (!hasOrders || q.ordersSinceHistory >= 3) {
		topic = lib.NatsMarketHistoriesIngest
		q.ordersSinceHistory = 0
	} else if hasOrders {
		q.ordersSinceHistory++
	}

	var seed *haboUploadJob
	for _, job := range q.jobs {
		if job.InFlight || job.Topic != topic || job.TokenFingerprint != fingerprint || now.Before(job.NextAttemptAt) {
			continue
		}
		seed = job
		break
	}
	if seed == nil {
		return nil
	}

	rowLimit := haboUploadMaxOrderRows
	if topic == lib.NatsMarketHistoriesIngest {
		rowLimit = haboUploadMaxHistoryRows
	}
	rows := 0
	estimatedBytes := 0
	selected := make([]*haboUploadJob, 0, 8)
	for _, job := range q.jobs {
		if len(selected) >= haboUploadMaxEnvelopes {
			break
		}
		if job.InFlight || job.Topic != topic || job.ServerID != seed.ServerID || job.TokenFingerprint != fingerprint || now.Before(job.NextAttemptAt) {
			continue
		}
		if rows+job.RowCount > rowLimit {
			continue
		}
		nextBytes := estimatedBytes + len(job.Payload) + 128
		if len(selected) > 0 && nextBytes > haboUploadTargetBodyBytes {
			break
		}
		job.InFlight = true
		selected = append(selected, job)
		rows += job.RowCount
		estimatedBytes = nextBytes
	}
	return selected
}

func buildHaboBatch(jobs []*haboUploadJob) ([]byte, []*haboUploadJob, []*haboUploadJob) {
	if len(jobs) == 0 {
		return nil, nil, nil
	}
	switch jobs[0].Topic {
	case lib.NatsMarketOrdersIngest:
		request := queuedOrderBatch{Batches: make([]queuedOrderEnvelope, 0, len(jobs))}
		usable := make([]*haboUploadJob, 0, len(jobs))
		var malformed []*haboUploadJob
		for _, job := range jobs {
			var payload queuedOrderPayload
			if json.Unmarshal(job.Payload, &payload) != nil || len(payload.Orders) == 0 {
				malformed = append(malformed, job)
				continue
			}
			request.Batches = append(request.Batches, queuedOrderEnvelope{Orders: payload.Orders, CapturedAt: job.CapturedAt})
			usable = append(usable, job)
		}
		body, err := json.Marshal(request)
		if err != nil {
			return nil, nil, jobs
		}
		return body, usable, malformed

	case lib.NatsMarketHistoriesIngest:
		request := queuedHistoryBatch{Batches: make([]queuedHistoryEnvelope, 0, len(jobs))}
		usable := make([]*haboUploadJob, 0, len(jobs))
		var malformed []*haboUploadJob
		for _, job := range jobs {
			var payload queuedHistoryPayload
			if json.Unmarshal(job.Payload, &payload) != nil || len(payload.MarketHistories) == 0 {
				malformed = append(malformed, job)
				continue
			}
			request.Batches = append(request.Batches, queuedHistoryEnvelope{
				AlbionID:        payload.AlbionID,
				LocationID:      payload.LocationID,
				QualityLevel:    payload.QualityLevel,
				Timescale:       payload.Timescale,
				MarketHistories: payload.MarketHistories,
				CapturedAt:      job.CapturedAt,
			})
			usable = append(usable, job)
		}
		body, err := json.Marshal(request)
		if err != nil {
			return nil, nil, jobs
		}
		return body, usable, malformed
	default:
		return nil, nil, jobs
	}
}

func (q *haboUploadQueue) sendBatch(token string, jobs []*haboUploadJob, body []byte) (int, time.Duration, error) {
	if len(jobs) == 0 {
		return 0, 0, fmt.Errorf("empty Habo upload batch")
	}
	first := jobs[0]
	req, err := http.NewRequest("POST", q.baseURL+"/"+first.Topic, bytes.NewReader(body))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("habo-client/%s", version))
	req.Header.Set("X-Habo-Server-Id", strconv.Itoa(first.ServerID))
	req.Header.Set("X-Habo-Batch-Count", strconv.Itoa(len(jobs)))
	if first.Identifier != "" {
		req.Header.Set("X-Habo-Identifier", first.Identifier)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	retryAfter := time.Duration(0)
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter = parseHaboRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	}
	return resp.StatusCode, retryAfter, nil
}

func parseHaboRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds)*time.Second + time.Second
	}
	if parsed, err := http.ParseTime(value); err == nil && parsed.After(now) {
		return parsed.Sub(now) + time.Second
	}
	return haboUploadRetryAfterFallback + time.Second
}

func haboRetryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	backoff := 2 * time.Second
	for index := 1; index < attempt; index++ {
		backoff *= 2
		if backoff >= time.Minute {
			return time.Minute
		}
	}
	if backoff > time.Minute {
		return time.Minute
	}
	return backoff
}

func (q *haboUploadQueue) finishAttempt(jobs []*haboUploadJob, status int, retryAfter time.Duration, requestErr error) {
	now := time.Now()
	ids := make(map[string]struct{}, len(jobs))
	rows := 0
	for _, job := range jobs {
		ids[job.ID] = struct{}{}
		rows += job.RowCount
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.nextSend = now.Add(q.sendInterval)

	switch {
	case requestErr == nil && status >= 200 && status < 300:
		q.removeJobsByIDLocked(ids)
		q.stats.SentBatches++
		q.stats.SentRows += int64(rows)
		log.Infof("Sent Habo %s batch: %d payloads, %d rows; queue=%d", jobs[0].Topic, len(jobs), rows, len(q.jobs))
		return

	case status == http.StatusTooManyRequests:
		if retryAfter <= 0 {
			retryAfter = haboUploadRetryAfterFallback + time.Second
		}
		for _, job := range jobs {
			job.InFlight = false
			job.Attempts++
			job.NextAttemptAt = now.Add(retryAfter)
		}
		q.stats.RateLimitRetries++
		log.Warnf("Habo upload rate limited; retrying %d payloads after %s.", len(jobs), retryAfter.Round(time.Second))
		return

	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		for _, job := range jobs {
			job.InFlight = false
			job.Attempts++
			job.NextAttemptAt = now.Add(haboUploadAuthPause)
		}
		log.Warnf("Habo upload returned HTTP %d; pausing queued uploads for %s.", status, haboUploadAuthPause)
		return

	case requestErr != nil || status == http.StatusRequestTimeout || status == http.StatusTooEarly || status >= 500:
		maxBackoff := time.Duration(0)
		for _, job := range jobs {
			job.InFlight = false
			job.Attempts++
			backoff := haboRetryBackoff(job.Attempts)
			job.NextAttemptAt = now.Add(backoff)
			if backoff > maxBackoff {
				maxBackoff = backoff
			}
		}
		if requestErr != nil {
			q.stats.NetworkRetries++
			log.Warnf("Habo upload failed; retrying %d payloads after %s: %v", len(jobs), maxBackoff, requestErr)
		} else {
			q.stats.ServerRetries++
			log.Warnf("Habo upload returned HTTP %d; retrying %d payloads after %s.", status, len(jobs), maxBackoff)
		}
		return

	default:
		q.removeJobsByIDLocked(ids)
		q.stats.DroppedPermanent += int64(len(jobs))
		log.Errorf("Habo upload returned non-retryable HTTP %d; dropped %d queued payloads.", status, len(jobs))
	}
}

func (q *haboUploadQueue) releaseInFlight(jobs []*haboUploadJob) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, job := range jobs {
		job.InFlight = false
	}
}

func (q *haboUploadQueue) dropMalformed(jobs []*haboUploadJob) {
	ids := make(map[string]struct{}, len(jobs))
	for _, job := range jobs {
		ids[job.ID] = struct{}{}
	}
	q.mu.Lock()
	q.removeJobsByIDLocked(ids)
	q.stats.DroppedPermanent += int64(len(jobs))
	q.mu.Unlock()
}

func (q *haboUploadQueue) depth() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

func (q *haboUploadQueue) snapshotStats() haboUploadQueueStats {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.stats
}
