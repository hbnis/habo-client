package client

import (
	"testing"

	"github.com/ao-data/albiondata-client/client/photon"
)

// Real Protocol18 responses have a typed debug value followed by a parameter
// table. The market order array belongs at key 0 and the operation code at
// key 253. Both must survive decoding without replacing the wire code.
func buildMarketResponsePacket(outerCode, wireCode byte, debugOrders, tableOrders []string) []byte {
	body := []byte{0, 0} // successful return code, little-endian
	if debugOrders == nil {
		body = append(body, photon.TypeNull)
	} else {
		body = append(body, encodeTestStringArray(debugOrders)...)
	}
	count := uint32(1)
	if tableOrders != nil {
		count = 2
	}
	body = append(body, writeVarint(count)...)
	body = append(body, 253, 11, wireCode) // Protocol18 Int1
	if tableOrders != nil {
		body = append(body, 0) // parameter key 0: orders
		body = append(body, encodeTestStringArray(tableOrders)...)
	}
	return buildPhotonPacket(photon.MsgResponse, outerCode, body)
}

func encodeTestStringArray(orders []string) []byte {
	out := []byte{0x47} // Protocol18 typed string array
	out = append(out, writeVarint(uint32(len(orders)))...)
	for _, order := range orders {
		out = append(out, writeVarint(uint32(len(order)))...)
		out = append(out, order...)
	}
	return out
}

func receiveDecodedOperation(t *testing.T, pkt []byte) operation {
	t.Helper()
	l := newListener(newRouter())
	if !l.parser.ReceivePacket(pkt) {
		t.Fatal("Photon parser rejected valid response")
	}
	select {
	case op := <-l.router.newOperation:
		return op
	default:
		t.Fatal("no decoded operation was dispatched")
		return nil
	}
}

func TestMarketResponsePreservesBuyVersusSellOperationCode(t *testing.T) {
	const orderJSON = `{"ItemTypeId":"T4_BAG"}`
	for _, tc := range []struct {
		name string
		code byte
	}{
		{"sell offers", 81},
		{"buy requests", 82},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Outer byte is intentionally not authoritative; the wire
			// operation code in param 253 determines the response kind.
			op := receiveDecodedOperation(t, buildMarketResponsePacket(0, tc.code, nil, []string{orderJSON}))
			switch tc.code {
			case 81:
				sell, ok := op.(*operationAuctionGetOffersResponse)
				if !ok || len(sell.MarketOrders) != 1 {
					t.Fatalf("expected one sell offer, got %T (%+v)", op, op)
				}
			case 82:
				buy, ok := op.(*operationAuctionGetRequestsResponse)
				if !ok || len(buy.MarketOrders) != 1 {
					t.Fatalf("expected one buy request, got %T (%+v)", op, op)
				}
			}
		})
	}
}

func TestUnrelatedStringArrayDoesNotBecomeSellOrders(t *testing.T) {
	// Old code discarded the parameter table and/or forced this response
	// to AuctionGetOffers solely because it contained a []string.
	l := newListener(newRouter())
	pkt := buildMarketResponsePacket(81, 1, []string{"not an order"}, nil) // Ping (code 1)
	if !l.parser.ReceivePacket(pkt) {
		t.Fatal("Photon parser rejected valid response")
	}
	if n := len(l.router.newOperation); n != 0 {
		t.Fatalf("unrelated response was incorrectly routed as a market order: %d operations", n)
	}
}

func TestDebugArrayDoesNotHideActualMarketOrderTable(t *testing.T) {
	const orderJSON = `{"ItemTypeId":"T4_BAG"}`
	pkt := buildMarketResponsePacket(0, 81, []string{}, []string{orderJSON})
	op := receiveDecodedOperation(t, pkt)
	sell, ok := op.(*operationAuctionGetOffersResponse)
	if !ok || len(sell.MarketOrders) != 1 || sell.MarketOrders[0] != orderJSON {
		t.Fatalf("lost table order after decoding debug array: %T (%+v)", op, op)
	}
}

func TestLegacyDebugArrayNeedsExplicitMarketCode(t *testing.T) {
	const orderJSON = `{"ItemTypeId":"T4_BAG"}`
	pkt := buildMarketResponsePacket(0, 82, []string{orderJSON}, nil)
	op := receiveDecodedOperation(t, pkt)
	buy, ok := op.(*operationAuctionGetRequestsResponse)
	if !ok || len(buy.MarketOrders) != 1 {
		t.Fatalf("legacy buy response not preserved: %T (%+v)", op, op)
	}
}

func TestCaptureSnaplenIsLargeEnoughForMarketOrderDatagrams(t *testing.T) {
	// A 2048 byte snapshot can truncate coalesced UDP market order replies
	// even while shorter history responses continue decoding normally.
	if captureSnaplen < 65535 {
		t.Fatalf("capture snaplen %d may truncate full-size market datagrams", captureSnaplen)
	}
}
