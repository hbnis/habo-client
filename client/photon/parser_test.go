package photon

import (
	"encoding/binary"
	"testing"
)

// buildFragmentPacket wraps a single SendFragment command into a minimal
// valid Photon UDP packet (12-byte header + 12-byte command header +
// 20-byte fragment header + data).
func buildFragmentPacket(startSeq, fragCount, fragNum, totalLen, fragOffset uint32, data []byte) []byte {
	hdr := []byte{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0}

	fragHdr := make([]byte, fragmentHeaderLength)
	binary.BigEndian.PutUint32(fragHdr[0:], startSeq)
	binary.BigEndian.PutUint32(fragHdr[4:], fragCount)
	binary.BigEndian.PutUint32(fragHdr[8:], fragNum)
	binary.BigEndian.PutUint32(fragHdr[12:], totalLen)
	binary.BigEndian.PutUint32(fragHdr[16:], fragOffset)

	cmdData := append(fragHdr, data...)

	cmdHdr := make([]byte, commandHeaderLength)
	cmdHdr[0] = cmdSendFragment
	binary.BigEndian.PutUint32(cmdHdr[4:], uint32(commandHeaderLength+len(cmdData)))

	pkt := append(append([]byte{}, hdr...), cmdHdr...)
	pkt = append(pkt, cmdData...)
	return pkt
}

// TestPendingSegments_BoundedByCap verifies that incomplete fragment
// reassembly (e.g. because a fragment was lost to packet loss during a
// network hiccup) doesn't grow pendingSegments without bound - each
// abandoned start-of-fragment allocates a totalLen-sized buffer that is
// never released otherwise.
func TestPendingSegments_BoundedByCap(t *testing.T) {
	parser := NewPhotonParser(nil, nil, nil)

	for i := 0; i < maxPendingSegments*4; i++ {
		// Send only the first fragment of a 2-fragment message, so it never
		// completes and stays in pendingSegments.
		pkt := buildFragmentPacket(uint32(i), 2, 0, 20, 0, make([]byte, 10))
		parser.ReceivePacket(pkt)
	}

	if len(parser.pendingSegments) > maxPendingSegments {
		t.Fatalf("pendingSegments grew to %d entries, want capped at %d", len(parser.pendingSegments), maxPendingSegments)
	}
}

// TestPendingSegments_RejectsOversizedTotalLength verifies a corrupt or
// hostile totalLen field (read straight off the wire) can't force a huge
// allocation.
func TestPendingSegments_RejectsOversizedTotalLength(t *testing.T) {
	parser := NewPhotonParser(nil, nil, nil)

	pkt := buildFragmentPacket(1, 2, 0, maxSegmentTotalLength+1, 0, make([]byte, 10))
	parser.ReceivePacket(pkt)

	if len(parser.pendingSegments) != 0 {
		t.Fatalf("pendingSegments has %d entries, want 0 for an oversized totalLen", len(parser.pendingSegments))
	}
}

// A duplicated fragment must not count twice and cause a partially filled
// market response to be dispatched as a complete Photon message.
func TestPendingSegments_DuplicateDoesNotCompleteMessage(t *testing.T) {
	parser := NewPhotonParser(nil, nil, nil)
	first := buildFragmentPacket(111, 2, 0, 12, 0, []byte("123456"))
	parser.ReceivePacket(first)
	parser.ReceivePacket(first)
	seg := parser.pendingSegments[111]
	if seg == nil || seg.bytesWritten != 6 || len(seg.fragments) != 1 {
		t.Fatalf("duplicate inflated completion: %+v", seg)
	}
	parser.ReceivePacket(buildFragmentPacket(111, 2, 1, 12, 6, []byte("abcdef")))
	if _, found := parser.pendingSegments[111]; found {
		t.Fatal("complete message retained after receiving both unique fragments")
	}
}

func TestPendingSegments_OutOfOrderAndOverlap(t *testing.T) {
	parser := NewPhotonParser(nil, nil, nil)
	parser.ReceivePacket(buildFragmentPacket(222, 2, 1, 12, 6, []byte("abcdef")))
	parser.ReceivePacket(buildFragmentPacket(222, 2, 0, 12, 0, []byte("123456")))
	if _, found := parser.pendingSegments[222]; found {
		t.Fatal("out-of-order complete message was not released")
	}

	parser.ReceivePacket(buildFragmentPacket(333, 2, 0, 12, 0, []byte("123456")))
	parser.ReceivePacket(buildFragmentPacket(333, 2, 1, 12, 4, []byte("abcdef")))
	if _, found := parser.pendingSegments[333]; found {
		t.Fatal("overlapping fragments should discard a corrupt segment")
	}
}

func TestPendingSegments_RejectsInvalidFragmentBounds(t *testing.T) {
	parser := NewPhotonParser(nil, nil, nil)
	parser.ReceivePacket(buildFragmentPacket(444, 2, 0, 12, 11, []byte("ab")))
	parser.ReceivePacket(buildFragmentPacket(445, 2, 2, 12, 0, []byte("ab")))
	if len(parser.pendingSegments) != 0 {
		t.Fatalf("invalid fragments allocated incomplete segments: %+v", parser.pendingSegments)
	}
}
