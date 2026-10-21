package cmd

import (
	"os"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/server"

	familabs "gitlab.com/fami-app-v2/familib/familabs/server"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
)

var consumerCmd = &cobra.Command{
	Use:   "consumer",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
			examples and usage of using your application.`,
	Run: func(_ *cobra.Command, _ []string) {
		startConsumer()
	},
}

// main creating new instances application and serving application server.
func startConsumer() {
	var err error
	if global.Setup.Common, err = familabs.Start(); err != nil {
		logrus.Error(err.Error())
		os.Exit(1)
		return
	}
	// familabs.Start() leaves Options.Trace nil (its tracer setup is
	// commented out), but every service/repository calls opt.Trace.Start.
	// Use the global OpenTelemetry tracer (no-op unless a provider is set).
	if global.Setup.Common.Trace == nil {
		global.Setup.Common.Trace = otel.Tracer(global.Setup.Common.Config.App.Name)
	}
	server.StartConsumer()
}
