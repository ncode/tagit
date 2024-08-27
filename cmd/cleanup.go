/*
Copyright © 2024 Juliano Martinez <juliano@martinez.io>

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
	"log/slog"
	"os"

	"github.com/hashicorp/consul/api"
	"github.com/ncode/tagit/pkg/tagit"
	"github.com/spf13/cobra"
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "cleanup removes all services with the tag prefix from a given consul service",
	Run: func(cmd *cobra.Command, args []string) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		config := api.DefaultConfig()
		config.Address = cmd.InheritedFlags().Lookup("consul-addr").Value.String()
		config.Token = cmd.InheritedFlags().Lookup("token").Value.String()

		consulClient, err := api.NewClient(config)
		if err != nil {
			logger.Error("Failed to create Consul client", "error", err)
			os.Exit(1)
		}

		serviceID := cmd.InheritedFlags().Lookup("service-id").Value.String()
		tagPrefix := cmd.InheritedFlags().Lookup("tag-prefix").Value.String()

		t := tagit.New(
			tagit.NewConsulAPIWrapper(consulClient),
			&tagit.CmdExecutor{},
			serviceID,
			"", // script is not needed for cleanup
			0,  // interval is not needed for cleanup
			tagPrefix,
			logger,
		)

		logger.Info("Starting tag cleanup", "serviceID", serviceID, "tagPrefix", tagPrefix)

		err = t.CleanupTags()
		if err != nil {
			logger.Error("Failed to clean up tags", "error", err)
			os.Exit(1)
		}

		logger.Info("Tag cleanup completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}
