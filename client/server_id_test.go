package client

import "testing"

func TestJoinResponsePreservesDetectedServerID(t *testing.T) {
	state := &albionState{
		AODataServerID: 3,
		GameServerIP:   "193.169.238.42",
	}

	operationJoinResponse{Location: "3000"}.Process(state)

	if state.AODataServerID != 3 {
		t.Fatalf("AODataServerID = %d, want 3 after join response", state.AODataServerID)
	}
}

func TestServerIDForUploadRecoversFromKnownGameServerIP(t *testing.T) {
	state := &albionState{
		AODataServerID: 0,
		GameServerIP:   "193.169.238.42",
	}

	if got := state.ServerIDForUpload(); got != 3 {
		t.Fatalf("ServerIDForUpload() = %d, want 3", got)
	}
}

func TestServerIDForUploadPrefersDetectedServerAfterSwitch(t *testing.T) {
	state := &albionState{
		AODataServerID: 3,
		GameServerIP:   "5.188.125.42",
	}

	if got := state.ServerIDForUpload(); got != 1 {
		t.Fatalf("ServerIDForUpload() = %d, want 1 from latest game-server IP", got)
	}
}

func TestServerIDForUploadKeepsLastValidServerForUnrelatedPacket(t *testing.T) {
	state := &albionState{
		AODataServerID: 2,
		GameServerIP:   "10.0.0.1",
	}

	if got := state.ServerIDForUpload(); got != 2 {
		t.Fatalf("ServerIDForUpload() = %d, want last valid server 2", got)
	}
}

func TestServerIDForUploadRejectsUnknownServer(t *testing.T) {
	state := &albionState{}

	if got := state.ServerIDForUpload(); got != 0 {
		t.Fatalf("ServerIDForUpload() = %d, want 0 when server is unknown", got)
	}
}
