package cmd

import (
	"os"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/server"

	familabs "gitlab.com/fami-app-v2/familib/familabs/server"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var grpcCmd = &cobra.Command{
	Use:   "grpc",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
			examples and usage of using your application.`,
	Run: func(_ *cobra.Command, _ []string) {
		startGrpc()
	},
}

// main creating new instances application and serving application server.
func startGrpc() {
	var err error
	if global.Setup.Common, err = familabs.Start(); err != nil {
		logrus.Error(err.Error())
		os.Exit(1)
		return
	}
	server.StartGrpcServer()
}
