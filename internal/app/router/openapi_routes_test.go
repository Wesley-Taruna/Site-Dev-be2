package router

// The API docs (docs/openapi.yaml) must list exactly the routes the router
// has, so they cannot go stale: a new route without docs, or docs for a
// route that was removed, fails here. Also checks the docs page switch.

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	"site-dev-backend/docs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// specOperation is one "METHOD /path" of openapi.yaml with the keys found
// directly under it.
type specOperation struct {
	key  string
	keys map[string]bool
}

var (
	specPathLine   = regexp.MustCompile(`^  (/[^:\s]*):\s*$`)
	specMethodLine = regexp.MustCompile(`^    (get|put|post|delete|patch|head|options):\s*$`)
	specFieldLine  = regexp.MustCompile(`^      ([a-zA-Z-]+):`)
	routeParam     = regexp.MustCompile(`:([A-Za-z0-9_]+)`)
)

// specOperations reads the paths block of openapi.yaml line by line (the
// file keeps paths at 2 spaces, methods at 4, fields at 6), so the test
// needs no YAML library.
func specOperations(t *testing.T) []specOperation {
	t.Helper()
	var ops []specOperation
	inPaths := false
	path := ""
	sc := bufio.NewScanner(bytes.NewReader(docs.OpenAPI))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "paths:") {
			inPaths = true
			continue
		}
		if !inPaths {
			continue
		}
		if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "#") {
			break // next top-level key
		}
		if m := specPathLine.FindStringSubmatch(line); m != nil {
			path = m[1]
			continue
		}
		if m := specMethodLine.FindStringSubmatch(line); m != nil {
			require.NotEmpty(t, path, "method line before any path: %q", line)
			ops = append(ops, specOperation{key: strings.ToUpper(m[1]) + " " + path, keys: map[string]bool{}})
			continue
		}
		if m := specFieldLine.FindStringSubmatch(line); m != nil && len(ops) > 0 {
			ops[len(ops)-1].keys[m[1]] = true
		}
	}
	require.NoError(t, sc.Err())
	return ops
}

// routerOperations lists "METHOD /path" of every /v1/ route, with echo's
// ":id" written as OpenAPI's "{id}".
func routerOperations(t *testing.T) []string {
	t.Helper()
	testEnv.Set("docs.enabled", true)
	t.Cleanup(func() { testEnv.Set("docs.enabled", false) })
	e, _, _ := stack(t)

	seen := map[string]bool{}
	var out []string
	for _, r := range e.Routes() {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		default:
			continue
		}
		if !strings.HasPrefix(r.Path, "/v1/") {
			continue
		}
		k := r.Method + " " + routeParam.ReplaceAllString(r.Path, "{$1}")
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func TestOpenAPIListsEveryRoute(t *testing.T) {
	routes := routerOperations(t)
	ops := specOperations(t)

	inSpec := map[string]bool{}
	for _, op := range ops {
		assert.False(t, inSpec[op.key], "openapi.yaml lists %s twice", op.key)
		inSpec[op.key] = true
	}
	inRouter := map[string]bool{}
	for _, k := range routes {
		inRouter[k] = true
	}

	for _, k := range routes {
		assert.True(t, inSpec[k], "route %s is not in docs/openapi.yaml: add it there", k)
	}
	for _, op := range ops {
		assert.True(t, inRouter[op.key], "docs/openapi.yaml lists %s, but the router has no such route", op.key)
	}
	// every operation says what it is, who may call it and what it answers
	for _, op := range ops {
		for _, field := range []string{"tags", "summary", "description", "x-roles", "operationId", "responses"} {
			assert.True(t, op.keys[field], "%s in docs/openapi.yaml has no %s", op.key, field)
		}
	}
	// guard against the test reading nothing
	assert.Greater(t, len(routes), 60)
	assert.Equal(t, len(routes), len(ops))
}

func TestDocsPageOnlyWhenEnabled(t *testing.T) {
	get := func(h http.Handler, path string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		return rr
	}

	// default (key missing / false): closed
	off, _, _ := stack(t)
	assert.Equal(t, http.StatusNotFound, get(off, "/v1/docs").Code)
	assert.Equal(t, http.StatusNotFound, get(off, "/v1/docs/openapi.yaml").Code)

	testEnv.Set("docs.enabled", true)
	t.Cleanup(func() { testEnv.Set("docs.enabled", false) })
	on, _, _ := stack(t)

	page := get(on, "/v1/docs")
	require.Equal(t, http.StatusOK, page.Code)
	assert.Contains(t, page.Header().Get(echoContentType), "text/html")
	assert.Contains(t, page.Body.String(), "swagger-ui-dist@5.17.14")

	spec := get(on, "/v1/docs/openapi.yaml")
	require.Equal(t, http.StatusOK, spec.Code)
	assert.Contains(t, spec.Header().Get(echoContentType), "application/yaml")
	assert.True(t, bytes.HasPrefix(spec.Body.Bytes(), []byte("openapi: 3.")))

	// the API itself still needs a token while the docs are on
	assert.Equal(t, http.StatusForbidden, get(on, "/v1/task").Code)
}

const echoContentType = "Content-Type"
