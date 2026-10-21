package router

import (
	"site-dev-backend/internal/app/handler"
)

var handlers = map[string]RouteHandlers{}

func init() {
	handlers["health-check"] = &handler.HealthCheckHandler{}
	handlers["person"] = &handler.PersonHandler{}
}
