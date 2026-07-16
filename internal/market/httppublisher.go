package market

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPPublisher posts idle-capacity offers to a real marketplace endpoint.
type HTTPPublisher struct {
	baseURL      string
	path         string
	authHeader   string
	authScheme   string
	token        string
	writeEnabled bool
	client       *http.Client
}

// HTTPConfig configures the live marketplace publisher.
type HTTPConfig struct {
	BaseURL      string
	Path         string // defaults to "/v1/marketplace/supply"
	AuthHeader   string // defaults to "Authorization"
	AuthScheme   string // defaults to "Bearer"
	Token        string
	WriteEnabled bool
}

// NewHTTPPublisher builds a live publisher. Writes are still gated by WriteEnabled so
// nothing is posted unless explicitly turned on.
func NewHTTPPublisher(cfg HTTPConfig) *HTTPPublisher {
	path := cfg.Path
	if path == "" {
		path = "/v1/marketplace/supply"
	}
	header := cfg.AuthHeader
	if header == "" {
		header = "Authorization"
	}
	scheme := cfg.AuthScheme
	if scheme == "" {
		scheme = "Bearer"
	}
	return &HTTPPublisher{
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		path:         path,
		authHeader:   header,
		authScheme:   scheme,
		token:        cfg.Token,
		writeEnabled: cfg.WriteEnabled,
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *HTTPPublisher) setAuth(req *http.Request) {
	if p.token == "" {
		return
	}
	if p.authScheme == "" {
		req.Header.Set(p.authHeader, p.token)
		return
	}
	req.Header.Set(p.authHeader, p.authScheme+" "+p.token)
}

// PublishSupply POSTs the offer to the marketplace.
func (p *HTTPPublisher) PublishSupply(ctx context.Context, offer Offer) error {
	if !p.writeEnabled {
		return nil
	}
	body, err := json.Marshal(offer)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+p.path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	p.setAuth(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("marketplace publish returned %d", resp.StatusCode)
	}
	return nil
}

// WithdrawSupply removes a previously posted offer.
func (p *HTTPPublisher) WithdrawSupply(ctx context.Context, id string) error {
	if !p.writeEnabled {
		return nil
	}
	url := fmt.Sprintf("%s%s/%s", p.baseURL, p.path, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	p.setAuth(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("marketplace withdraw returned %d", resp.StatusCode)
	}
	return nil
}
