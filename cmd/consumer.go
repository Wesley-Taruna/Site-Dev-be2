package cmd

import (
	"os"

	"fami-boilerplate-service/global"
	"fami-boilerplate-service/internal/app/server"

	familabs "gitlab.com/fami-app-v2/familib/familabs/server"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	server.StartConsumer()
}
