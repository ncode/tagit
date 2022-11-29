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
	"fmt"
	"github.com/ncode/tagit/pkg/tagit"
	"github.com/spf13/cobra"
	"os"
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "cleanup all tags added by tagit to a given consul service",
	Run: func(cmd *cobra.Command, args []string) {
		consulAddr := cmd.PersistentFlags().Lookup("consul-addr").Value.String()
		serviceID := cmd.PersistentFlags().Lookup("service-id").Value.String()
		tagPrefix := cmd.PersistentFlags().Lookup("tag-prefix").Value.String()
		token := cmd.PersistentFlags().Lookup("token").Value.String()

		if serviceID == "" {
			fmt.Println("service-id is required")
			os.Exit(1)
		}

		if tagPrefix == "" {
			fmt.Println("tag-prefix is required")
			os.Exit(1)
		}

		t, err := tagit.New(
			consulAddr,
			serviceID,
			"",
			0,
			token,
			tagPrefix)
		if err != nil {
			fmt.Printf("error creating tagit: %s", err.Error())
			os.Exit(1)
		}
		err = t.CleanupServiceTags()
		if err != nil {
			fmt.Printf("error cleaning up tags: %s", err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
	cleanupCmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
	cleanupCmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
	cleanupCmd.PersistentFlags().StringP("token", "t", "", "consul token")
}
