package ocpi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPSourcePrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prices/H100" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Price{SKU: "H100", USDPerHour: 2.5, AsOf: time.Now()})
	}))
	defer srv.Close()

	src := NewHTTPSource(srv.URL)
	p, err := src.Price(context.Background(), "H100")
	if err != nil {
		t.Fatal(err)
	}
	if p.USDPerHour != 2.5 || p.SKU != "H100" {
		t.Fatalf("got %+v", p)
	}
}

func TestOrnnDataSourcePrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ocpi/H100/spot" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("auth = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"price": 3.25,
			"asOf":  time.Now().UTC().Format(time.RFC3339),
		})
	}))
	defer srv.Close()

	src := NewOrnnDataSource(OrnnDataConfig{
		BaseURL:   srv.URL,
		Token:     "secret-token",
		PricePath: "/v1/ocpi/{sku}/spot",
	})
	p, err := src.Price(context.Background(), "H100")
	if err != nil {
		t.Fatal(err)
	}
	if p.USDPerHour != 3.25 {
		t.Fatalf("price = %v", p.USDPerHour)
	}
}

func TestOrnnDataSourceRequiresToken(t *testing.T) {
	src := NewOrnnDataSource(OrnnDataConfig{BaseURL: "http://example", Token: ""})
	if _, err := src.Price(context.Background(), "H100"); err == nil {
		t.Fatal("expected error without token")
	}
}

func TestPriceStale(t *testing.T) {
	fresh := Price{AsOf: time.Now()}
	if fresh.Stale(time.Minute) {
		t.Fatal("fresh price should not be stale")
	}
	old := Price{AsOf: time.Now().Add(-2 * time.Hour)}
	if !old.Stale(time.Minute) {
		t.Fatal("old price should be stale")
	}
	if !(Price{}).Stale(time.Minute) {
		t.Fatal("zero AsOf should be stale")
	}
}
