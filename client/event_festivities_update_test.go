package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/ao-data/albiondata-client/lib"
)

const (
	activitiesStartTime int64 = 639211752000000000
	activitiesEndTime   int64 = 639212616000000000
	gatheringEndTime    int64 = 639213480000000000
)

func testFestivitiesUpdate() eventFestivitiesUpdate {
	return eventFestivitiesUpdate{
		Kinds:       []uint8{2, 1},
		Categories:  []string{"GENERAL", "GATHERING"},
		UniqueNames: []string{"COMMON_CROSSBOW", "ORE"},
		StartTimes:  []int64{activitiesStartTime, activitiesStartTime},
		EndTimes:    []int64{activitiesEndTime, gatheringEndTime},
	}
}

func TestDecodeFestivitiesUpdate(t *testing.T) {
	operation, err := decodeEvent(map[uint8]interface{}{
		0:   []uint8{2, 1},
		1:   []string{"GENERAL", "GATHERING"},
		2:   []string{"COMMON_CROSSBOW", "ORE"},
		3:   []int64{activitiesStartTime, activitiesStartTime},
		4:   []int64{activitiesEndTime, gatheringEndTime},
		252: int16(evFestivitiesUpdate),
	})
	if err != nil {
		t.Fatal(err)
	}

	event, ok := operation.(*eventFestivitiesUpdate)
	if !ok {
		t.Fatalf("unexpected operation type: %T", operation)
	}
	if !reflect.DeepEqual(*event, testFestivitiesUpdate()) {
		t.Fatalf("unexpected decoded event: %#v", event)
	}
}

func TestFestivitiesUpdateUpload(t *testing.T) {
	upload, err := testFestivitiesUpdate().upload()
	if err != nil {
		t.Fatal(err)
	}

	want := lib.FestivitiesUpload{
		Events: []*lib.Festivity{
			{
				Kind:       2,
				Category:   "GENERAL",
				UniqueName: "COMMON_CROSSBOW",
				StartTime:  activitiesStartTime,
				EndTime:    activitiesEndTime,
			},
			{
				Kind:       1,
				Category:   "GATHERING",
				UniqueName: "ORE",
				StartTime:  activitiesStartTime,
				EndTime:    gatheringEndTime,
			},
		},
	}
	if !reflect.DeepEqual(upload, want) {
		t.Fatalf("unexpected upload: %#v", upload)
	}
}

func TestFestivitiesUpdateRejectsInvalidPayload(t *testing.T) {
	event := testFestivitiesUpdate()
	event.EndTimes = event.EndTimes[:1]
	if _, err := event.upload(); err == nil {
		t.Fatal("expected inconsistent arrays to fail")
	}

	event = testFestivitiesUpdate()
	event.EndTimes[0] = event.StartTimes[0]
	if _, err := event.upload(); err == nil {
		t.Fatal("expected invalid interval to fail")
	}
}

func TestFestivitiesUpdateIsNotForwardedByHaboClient(t *testing.T) {
	requests := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests <- struct{}{}
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	oldDisableUpload := ConfigGlobal.DisableUpload
	oldPublicIngestBaseUrls := ConfigGlobal.PublicIngestBaseUrls
	oldPrivateIngestBaseUrls := getPrivateIngestBaseURLs()
	oldEnableWebsockets := ConfigGlobal.EnableWebsockets
	t.Cleanup(func() {
		ConfigGlobal.DisableUpload = oldDisableUpload
		ConfigGlobal.PublicIngestBaseUrls = oldPublicIngestBaseUrls
		SetPrivateIngestBaseURLs(oldPrivateIngestBaseUrls)
		ConfigGlobal.EnableWebsockets = oldEnableWebsockets
	})

	ConfigGlobal.DisableUpload = false
	ConfigGlobal.PublicIngestBaseUrls = server.URL
	SetPrivateIngestBaseURLs("")
	ConfigGlobal.EnableWebsockets = false

	state := &albionState{AODataServerID: 3}
	event := testFestivitiesUpdate()
	event.Process(state)

	select {
	case <-requests:
		t.Fatal("Habo Client forwarded festivities data; only market data should be uploaded")
	case <-time.After(100 * time.Millisecond):
	}
}
