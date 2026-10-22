package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func overpassClient(fn roundTripFunc, urls ...string) *OverpassClient {
	return &OverpassClient{
		urls:      urls,
		userAgent: "site-dev-backend test",
		perServer: 2 * time.Second,
		deadline:  10 * time.Second,
		http:      &http.Client{Transport: fn},
		gate:      &overpassGate{gap: 5 * time.Millisecond},
	}
}

func TestOverpassFetchArea_NextServerWhenBusy(t *testing.T) {
	var mu sync.Mutex
	var hosts []string
	c := overpassClient(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		hosts = append(hosts, r.URL.Host)
		mu.Unlock()
		if r.URL.Host == "busy.example.test" {
			return answer(504, "<html>Gateway Timeout</html>")
		}
		assert.Equal(t, "site-dev-backend test", r.Header.Get("User-Agent"))
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		raw, _ := io.ReadAll(r.Body)
		form, err := url.ParseQuery(string(raw))
		require.NoError(t, err)
		assert.Equal(t, OverpassAreaQuery(dagoLat, dagoLng, 620), form.Get("data"), "only the query (coordinates) is sent")
		return answer(200, fixture(t, "overpass_dago.json"))
	}, "https://busy.example.test/api/interpreter", "https://ok.example.test/api/interpreter")

	places, err := c.FetchArea(context.Background(), dagoLat, dagoLng, 620)

	require.NoError(t, err)
	assert.Equal(t, []string{"busy.example.test", "ok.example.test"}, hosts)
	assert.Equal(t, map[string]int{"competitor": 4, "office": 2, "school": 3, "university": 1, "apartment": 1}, countBy(places))
}

func TestOverpassFetchArea_AllFail(t *testing.T) {
	c := overpassClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "slow.example.test" {
			<-r.Context().Done() // per-server timeout
			return nil, r.Context().Err()
		}
		if r.URL.Host == "junk.example.test" {
			return answer(200, `{"remark":"runtime error: Query timed out"}`)
		}
		return nil, errors.New("connection refused")
	}, "https://slow.example.test/i", "https://junk.example.test/i", "https://down.example.test/i")
	c.perServer = 50 * time.Millisecond

	_, err := c.FetchArea(context.Background(), dagoLat, dagoLng, 620)

	assert.ErrorIs(t, err, ErrOverpassUnavailable)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestOverpassGate_OneRequestPerGap(t *testing.T) {
	g := &overpassGate{gap: 40 * time.Millisecond}
	start := time.Now()
	for i := 0; i < 3; i++ {
		require.NoError(t, g.wait(context.Background()))
	}
	assert.GreaterOrEqual(t, time.Since(start), 80*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.Error(t, g.wait(ctx))
}

func TestNewOverpassClientDefaults(t *testing.T) {
	c := NewOverpassClient(nil).(*OverpassClient)
	assert.Len(t, c.urls, 3)
	assert.True(t, strings.HasPrefix(c.urls[0], "https://maps.mail.ru/"))
	assert.Equal(t, 25*time.Second, c.perServer)
	assert.Equal(t, 60*time.Second, c.deadline)
}
