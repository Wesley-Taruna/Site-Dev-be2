package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	dto "site-dev-backend/internal/app/dto"

	"gitlab.com/fami-app-v2/familib/familabs/env"
	"go.opentelemetry.io/otel/attribute"
)

// ErrOverpassUnavailable: no OpenStreetMap server answered in time.
var ErrOverpassUnavailable = errors.New("overpass: no OpenStreetMap server answered")

// IOverpassClient is OpenStreetMap through the Overpass API (free, no key):
// the fallback source of nearby places.
type IOverpassClient interface {
	// FetchArea returns every counted OSM place within radiusM, nearest
	// first (OverpassAreaQuery + ParseOverpassArea).
	FetchArea(ctx context.Context, lat float64, lng float64, radiusM int) (places []dto.NearbyPlaceItem, err error)
}

var (
	defaultOverpassURLs = []string{
		"https://maps.mail.ru/osm/tools/overpass/api/interpreter",
		"https://overpass-api.de/api/interpreter",
		"https://overpass.kumi.systems/api/interpreter",
	}
	defaultOverpassUserAgent = "site-dev-backend (FamilyMart Indonesia site survey; contact: Site Development)"
)

const (
	// overpassMaxBody: answers for a 1 km box are well below this
	overpassMaxBody = 20 << 20
	// overpassMinLeft: do not start another server with less time left
	overpassMinLeft = 3 * time.Second
)

// overpassGate keeps at most one request per gap per process (fair use of
// the free public servers).
type overpassGate struct {
	mu   sync.Mutex
	last time.Time
	gap  time.Duration
}

var sharedOverpassGate = &overpassGate{gap: 1100 * time.Millisecond}

// wait reserves the next free slot and sleeps until it.
func (g *overpassGate) wait(ctx context.Context) error {
	g.mu.Lock()
	slot := time.Now()
	if next := g.last.Add(g.gap); next.After(slot) {
		slot = next
	}
	g.last = slot
	g.mu.Unlock()

	d := time.Until(slot)
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// OverpassClient posts OverpassAreaQuery to each server in turn until one
// answers 200 with readable data (standard library only).
type OverpassClient struct {
	urls      []string
	userAgent string
	perServer time.Duration
	deadline  time.Duration
	http      *http.Client
	gate      *overpassGate
}

// NewOverpassClient reads places.overpass_urls (default: maps.mail.ru,
// overpass-api.de, overpass.kumi.systems), places.overpass_user_agent,
// places.overpass_timeout_seconds (per server, 25) and
// places.overpass_deadline_seconds (all servers, 60). Only coordinates
// are sent.
func NewOverpassClient(env env.Provider) IOverpassClient {
	c := &OverpassClient{
		urls:      defaultOverpassURLs,
		userAgent: defaultOverpassUserAgent,
		perServer: 25 * time.Second,
		deadline:  60 * time.Second,
		http:      &http.Client{},
		gate:      sharedOverpassGate,
	}
	if env == nil {
		return c
	}
	var urls []string
	for _, u := range env.GetStringSlice("places.overpass_urls") {
		if u = strings.TrimSpace(u); u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) > 0 {
		c.urls = urls
	}
	if ua := strings.TrimSpace(env.GetString("places.overpass_user_agent")); ua != "" {
		c.userAgent = ua
	}
	if n := env.GetInt("places.overpass_timeout_seconds"); n > 0 {
		c.perServer = time.Duration(n) * time.Second
	}
	if n := env.GetInt("places.overpass_deadline_seconds"); n > 0 {
		c.deadline = time.Duration(n) * time.Second
	}
	return c
}

func (c *OverpassClient) FetchArea(ctx context.Context, lat float64, lng float64, radiusM int) (places []dto.NearbyPlaceItem, err error) {
	ctx, span := placesTracer.Start(ctx, "OverpassClient.FetchArea")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, c.deadline)
	defer cancel()

	form := url.Values{}
	form.Set("data", OverpassAreaQuery(lat, lng, radiusM))
	body := form.Encode()

	lastErr := "no server tried"
	for i, server := range c.urls {
		dl, _ := ctx.Deadline()
		left := time.Until(dl)
		if left < overpassMinLeft {
			break
		}
		if err = c.gate.wait(ctx); err != nil {
			lastErr = err.Error()
			break
		}
		per := c.perServer
		if left < per {
			per = left
		}
		places, err = c.try(ctx, server, body, per, lat, lng, radiusM)
		if err == nil {
			span.SetAttributes(attribute.Int("overpass.server", i), attribute.Int("overpass.count", len(places)))
			return places, nil
		}
		lastErr = err.Error()
	}
	err = fmt.Errorf("%w (%s)", ErrOverpassUnavailable, lastErr)
	span.RecordError(err)
	return nil, err
}

// try asks one server; any error = try the next one.
func (c *OverpassClient) try(ctx context.Context, server string, body string, timeout time.Duration, lat float64, lng float64, radiusM int) ([]dto.NearbyPlaceItem, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	res, err := c.http.Do(req)
	if err != nil {
		var uErr *url.Error
		if errors.As(err, &uErr) {
			err = uErr.Err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.New("timeout")
		}
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
		return nil, fmt.Errorf("HTTP %d", res.StatusCode) // 429 / 504 = busy: next server
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, overpassMaxBody))
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return ParseOverpassArea(raw, lat, lng, radiusM)
}
