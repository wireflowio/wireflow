package cmd

import (
	"github.com/spf13/cobra"
	"linkany/pkg/drp"
	"linkany/pkg/drp/drphttp"
)

func drpCmd() *cobra.Command {
	var opts drp.Options
	var cmd = &cobra.Command{
		Use:          "drp [command]",
		SilenceUsage: true,
		Short:        "drp is a relay server",
		Long:         `drp used for relay and choose the best network for you`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			return runDrp(opts)
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&opts.Listen, "", "l", "", "http port for drp over http")
	//fs.BoolVarP(&opts.RunDrp, "", "b", true, "run drp")
	return cmd
}

// run drp
func runDrp(opts drp.Options) error {
	return drphttp.Start(opts)
}
