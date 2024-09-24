/*
Copyright © 2022 Juliano Martinez <juliano@martinez.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/ncode/tagit/pkg/tagit"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run tagit to add tags to a given consul service based on a script output",
	Long: `Run tagit to add tags to a given consul service based on a script output.

example: tagit run -s my-super-service -x '/tmp/tag-role.sh'
`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		interval, err := cmd.InheritedFlags().GetString("interval")
		if err != nil {
			logger.Error("Failed to get interval flag", "error", err)
			os.Exit(1)
		}

		if interval == "" || interval == "0" {
			logger.Error("Interval is required")
			os.Exit(1)
		}

		validInterval, err := time.ParseDuration(interval)
		if err != nil {
			logger.Error("Invalid interval", "interval", interval, "error", err)
			os.Exit(1)
		}

		config := api.DefaultConfig()
		config.Address, err = cmd.InheritedFlags().GetString("consul-addr")
		if err != nil {
			logger.Error("Failed to get consul-addr flag", "error", err)
			os.Exit(1)
		}
		config.Token, err = cmd.InheritedFlags().GetString("token")
		if err != nil {
			logger.Error("Failed to get token flag", "error", err)
			os.Exit(1)
		}

		consulClient, err := api.NewClient(config)
		if err != nil {
			logger.Error("Failed to create Consul client", "error", err)
			os.Exit(1)
		}

		serviceID, err := cmd.InheritedFlags().GetString("service-id")
		if err != nil {
			logger.Error("Failed to get service-id flag", "error", err)
			os.Exit(1)
		}
		script, err := cmd.InheritedFlags().GetString("script")
		if err != nil {
			logger.Error("Failed to get script flag", "error", err)
			os.Exit(1)
		}
		tagPrefix, err := cmd.InheritedFlags().GetString("tag-prefix")
		if err != nil {
			logger.Error("Failed to get tag-prefix flag", "error", err)
			os.Exit(1)
		}

		t := tagit.New(
			tagit.NewConsulAPIWrapper(consulClient),
			&tagit.CmdExecutor{},
			serviceID,
			script,
			validInterval,
			tagPrefix,
			logger,
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Setup signal handling for graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			sig := <-sigCh
			logger.Info("Received signal, shutting down", "signal", sig)
			cancel()
		}()

		logger.Info("Starting tagit",
			"serviceID", serviceID,
			"script", script,
			"interval", validInterval,
			"tagPrefix", tagPrefix)

		t.Run(ctx)

		logger.Info("Tagit has stopped")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
