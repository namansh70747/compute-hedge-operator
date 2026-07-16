package market

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestLogPublisherRoundTrip(t *testing.T) {
	p := NewLogPublisher()
	offer := Offer{ID: "a", Position: "p", GPUCount: 2, AsOf: time.Now()}
	if err := p.PublishSupply(context.Background(), offer); err != nil {
		t.Fatal(err)
	}
	got, ok := p.Last()
	if !ok || got.ID != "a" {
		t.Fatalf("last = %+v ok=%v", got, ok)
	}
	if err := p.WithdrawSupply(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.Last(); ok {
		t.Fatal("expected no last offer after withdraw")
	}
}

func TestHTTPPublisherWriteGate(t *testing.T) {
	var calls atomic.Int32
	var mode atomic.Int32 // 0=none, 1=publish, 2=withdraw

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch mode.Load() {
		case 1:
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if r.URL.Path != "/v1/marketplace/supply" {
				t.Errorf("path = %s", r.URL.Path)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer tok" {
				t.Errorf("auth = %q", got)
			}
			var body Offer
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode: %v", err)
			}
			if body.ID != "offer-1" || body.GPUCount != 4 {
				t.Errorf("body = %+v", body)
			}
			w.WriteHeader(http.StatusCreated)
		case 2:
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if r.URL.Path != "/v1/marketplace/supply/offer-1" {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	gated := NewHTTPPublisher(HTTPConfig{
		BaseURL:      srv.URL,
		Token:        "tok",
		WriteEnabled: false,
	})
	if err := gated.PublishSupply(context.Background(), Offer{ID: "x"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("write gate should block posts, got %d calls", calls.Load())
	}

	live := NewHTTPPublisher(HTTPConfig{
		BaseURL:      srv.URL,
		Path:         "/v1/marketplace/supply",
		AuthScheme:   "Bearer",
		AuthHeader:   "Authorization",
		Token:        "tok",
		WriteEnabled: true,
	})
	offer := Offer{
		ID:              "offer-1",
		Position:        "batch",
		Namespace:       "ns",
		SKU:             "H100",
		GPUCount:        4,
		PriceUSDPerHour: 2.1,
		AsOf:            time.Now().UTC(),
	}
	mode.Store(1)
	if err := live.PublishSupply(context.Background(), offer); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 publish call, got %d", calls.Load())
	}

	mode.Store(2)
	if err := live.WithdrawSupply(context.Background(), "offer-1"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls after withdraw, got %d", calls.Load())
	}
}
