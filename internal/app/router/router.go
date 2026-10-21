package router

import (
	"net/http"
	"os"
	"time"

	"fami-boilerplate-service/global"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"gitlab.com/fami-app-v2/familib/familabs"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

// RouteHandlers interface of handlers
type RouteHandlers interface {
	URLMapping(r *echo.Group)
}

func Router() *echo.Echo {
	e := echo.New()

	// setup otelecho
	if global.Setup.Common.Config.Trace.Enabled {
		e.Use(otelecho.Middleware(global.Setup.Common.Config.App.Name))
	}

	// setup cors
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
		AllowCredentials: true,
		AllowHeaders:     []string{"Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Access-Control-Request-Headers, Access-Control-Request-Method, Connection, Host, Origin, User-Agent, Referer, Cache-Control, X-header"},
	}))

	// setup echo for request id
	e.Use(middleware.RequestID())

	// setup echo for secure
	e.Use(middleware.Secure())

	// setup echo for gzip compres
	e.Use(middleware.Gzip())

	// setup echo for recover
	if !global.Setup.Common.Config.App.Debug {
		e.Use(middleware.Recover())
	}

	// setup echo for real ip
	e.IPExtractor = echo.ExtractIPFromRealIPHeader()

	// setup middleware for edenlabs context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &familabs.Context{
				Context:        c,
				ResponseFormat: familabs.NewResponse(),
				ResponseData:   nil,
			}
			return next(cc)
		}
	})

	// setup echo for logger
	lgr := global.Setup.Common.Logger.Logger()
	if global.Setup.Common.Config.App.Debug {
		lgr.SetFormatter(log.NewFormater(true, global.Setup.Common.Config.App.Name))
	} else {
		lgr.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	}
	lgr.SetReportCaller(true)
	lgr.SetOutput(os.Stdout)

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:           true,
		LogURIPath:       true,
		LogRoutePath:     true,
		LogStatus:        true,
		LogLatency:       true,
		LogProtocol:      true,
		LogMethod:        true,
		LogRequestID:     true,
		LogError:         true,
		LogContentLength: true,
		LogHost:          true,
		LogRemoteIP:      true,
		LogValuesFunc: func(ctx echo.Context, values middleware.RequestLoggerValues) (err error) {
			fields := logrus.Fields{
				"service":    global.Setup.Common.Config.App.Name,
				"request_id": values.RequestID,
				"uri":        values.URI,
				"uri_path":   values.URIPath,
				"route_path": values.RoutePath,
				"status":     values.Status,
				"host":       values.Host,
				"protocol":   values.Protocol,
				"method":     values.Method,
				"remote_ip":  values.RemoteIP,
				"start_time": values.StartTime,
				"end_time":   time.Now(),
				"latency":    values.Latency,
			}

			if values.Error != nil {
				fields["error"] = values.Error
			}

			if values.Status >= 400 || values.Error != nil {
				lgr.WithFields(fields).Error("HTTP " + http.StatusText(values.Status))
			} else {
				lgr.WithFields(fields).Info("HTTP " + http.StatusText(values.Status))
			}

			return
		},
	}))

	// setup binder
	e.Binder = &familabs.Binder{}

	// setup error handler
	e.HTTPErrorHandler = familabs.HTTPErrorHandler

	// handlers register an endpoint with handler here.
	// it will automatic registered into routers
	v1 := e.Group("v1/")

	// setup prometheus
	if global.Setup.Common.Config.Metric.Enabled {
		prom := prometheus.NewPrometheus("echo", middleware.DefaultSkipper)
		prom.Use(e)
	}
	if len(handlers) > 0 {
		for p, h := range handlers {
			h.URLMapping(v1.Group(p))
		}

	}

	return e
}
