package global

import (
	"site-dev-backend/internal/app/client"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type HandlerOptions struct {
	Common opt.Options
	// Clients are the outside services (internal/app/client), built once at
	// start.
	Clients client.Clients
}

var (
	Setup HandlerOptions
)
