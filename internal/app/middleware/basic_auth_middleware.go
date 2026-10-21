package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// BasicAuth protects the machine-to-machine callbacks (/v1/queue/*, called
// by the background-job service) with the [public_api] username/password
// from the config. Answers are plain {"error": "..."} JSON, as in
// site-dev-api's BasicAuthHandler.
func (m *Middleware) BasicAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			username, password, ok := c.Request().BasicAuth()
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Basic auth required"})
			}

			if username != m.Option.Env.GetString("public_api.username") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User does not exist"})
			}

			if password != m.Option.Env.GetString("public_api.password") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Incorrect password"})
			}

			c.Set("Username", username)
			return next(c)
		}
	}
}
