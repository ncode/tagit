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
	"os"
	"time"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		consulAddr := cmd.PersistentFlags().Lookup("consul-addr").Value.String()
		serviceID := cmd.PersistentFlags().Lookup("service-id").Value.String()
		script := cmd.PersistentFlags().Lookup("script").Value.String()
		tagPrefix := cmd.PersistentFlags().Lookup("tag-prefix").Value.String()
		interval := cmd.PersistentFlags().Lookup("interval").Value.String()
		token := cmd.PersistentFlags().Lookup("token").Value.String()

		if serviceID == "" {
			fmt.Println("service-id is required")
			os.Exit(1)
		}

		if script == "" {
			fmt.Println("script is required")
			os.Exit(1)
		}

		if tagPrefix == "" {
			fmt.Println("tag-prefix is required")
			os.Exit(1)
		}

		if interval == "" || interval == "0" {
			fmt.Println("interval is required")
			os.Exit(1)
		}

		i, err := time.ParseDuration(interval)
		if err != nil {
			fmt.Printf("invalid interval %s: %s", interval, err.Error())
			os.Exit(1)
		}

		t, err := tagit.New(
			consulAddr,
			serviceID,
			script,
			i,
			token,
			tagPrefix)
		if err != nil {
			fmt.Printf("error creating tagit: %s", err.Error())
			os.Exit(1)
		}
		t.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
	runCmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
	runCmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
	runCmd.PersistentFlags().StringP("tag-prefix", "p", "", "prefix to be added to tags")
	runCmd.PersistentFlags().StringP("interval", "i", "60s", "interval to run the script")
	runCmd.PersistentFlags().StringP("token", "t", "", "consul token")
}
