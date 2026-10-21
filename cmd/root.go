package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCommand returns a new instance of an command
func NewRootCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "fami-boilerplate-service",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
			examples and usage of using your application.`,
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(apiCmd, grpcCmd, consumerCmd)

	return command
}
