package middleware

import (
	"gitlab.com/fami-app-v2/familib/familabs/middlewarex"

	"github.com/labstack/echo/v4"
)

func (m *Middleware) Authorized(permissions ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) (err error) {
			err = middlewarex.Authorization(ctx, m.Option.Logger, m.Option.Config.Jwt.Key, permissions)
			if err != nil {
				return err
			}
			return next(ctx)
		}
	}
}
