// Package docs holds the API description (openapi.yaml) and the page that
// shows it (swagger-ui.html), built into the binary with go:embed so the
// server needs no extra files. They are served by handler.DocsHandler at
// /v1/docs only when config [docs] enabled = true (default off).
package docs

import _ "embed"

// OpenAPI is docs/openapi.yaml (OpenAPI 3). A test in internal/app/router
// fails when a route of the router is missing here, or when this file
// lists a route the router does not have.
//
//go:embed openapi.yaml
var OpenAPI []byte

// SwaggerUI is the interactive page. Swagger UI itself is loaded from a
// pinned CDN version (no Go library added); the page reads openapi.yaml
// next to its own address.
//
//go:embed swagger-ui.html
var SwaggerUI []byte
