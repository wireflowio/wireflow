// Copyright 2025 The Wireflow Authors, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"wireflow/wrrp"

	"github.com/spf13/cobra"
)

type wrrpOptions struct {
	Listen   string
	LogLevel string
	TLS      bool
}

func NewWrrpCmd() *cobra.Command {
	var opts wrrpOptions
	var cmd = &cobra.Command{
		Use:          "wrrp [command]",
		SilenceUsage: true,
		Short:        "wrrp using as relay server for wireflow",
		Long:         `wrrp using as relay server for wireflow`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			return runWrrp(&opts)
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&opts.Listen, "", "l", "", "http port for drp over http")
	fs.StringVarP(&opts.LogLevel, "level", "", "info", "log level (debug|info|warn|error)")
	fs.BoolVarP(&opts.TLS, "", "", false, "using tls")
	return cmd
}

// run signaling server
func runWrrp(opts *wrrpOptions) error {
	server := wrrp.NewServer()
	return server.Start()
}
