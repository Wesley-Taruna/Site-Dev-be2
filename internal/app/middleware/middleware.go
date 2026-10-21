package middleware

import (
	"fami-boilerplate-service/global"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

// Middleware defines object for order api custom middleware
type Middleware struct {
	Option opt.Options
}

func NewMiddleware() *Middleware {
	return &Middleware{
		Option: global.Setup.Common,
	}
}
