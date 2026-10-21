package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/router"

	"github.com/sirupsen/logrus"
)

func StartRestServer() {
	var srv http.Server

	done := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		logrus.Infoln("[API] Server is shutting down")

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			logrus.Errorf("[API] Fail to shutting down: %v", err)
		}

		close(done)
	}()

	srv.Addr = fmt.Sprintf("%s:%d", global.Setup.Common.Config.App.Host, global.Setup.Common.Config.App.Port)

	srv.Handler = router.Router()

	logrus.Infof("[API] HTTP serve at %s\n", srv.Addr)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener
		logrus.Errorf("[API] Fail to start listen and server: %v", err)
	}

	<-done
	logrus.Info("[API] Bye")
}
