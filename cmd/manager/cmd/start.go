package cmd

import (
	"wireflow/cmd/manager/controller"
	"wireflow/cmd/manager/management"
	"wireflow/cmd/manager/turn"
	"wireflow/cmd/manager/wrrp"

	"github.com/spf13/cobra"
)

// start cmd
func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a Wireflow component (controller, client, wrrp, turn).",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmd.AddCommand(controller.NewControllerCmd())
	cmd.AddCommand(wrrp.NewWrrpCmd())
	cmd.AddCommand(turn.NewTurnCmd())
	cmd.AddCommand(management.NewManagementCmd())

	return cmd
}
