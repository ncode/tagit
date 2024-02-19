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
	"fmt"
	"os"
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
		interval := cmd.InheritedFlags().Lookup("interval").Value.String()
		ctx := context.Background()
		if interval == "" || interval == "0" {
			fmt.Println("interval is required")
			os.Exit(1)
		}

		validInterval, err := time.ParseDuration(interval)
		if err != nil {
			fmt.Printf("invalid interval %s: %s", interval, err.Error())
			os.Exit(1)
		}

		config := api.DefaultConfig()
		config.Address = cmd.InheritedFlags().Lookup("consul-addr").Value.String()
		config.Token = cmd.InheritedFlags().Lookup("token").Value.String()
		consulClient, err := api.NewClient(config)
		if err != nil {
			fmt.Printf("error creating consul client: %s", err.Error())
			os.Exit(1)
		}

		t := tagit.New(
			tagit.NewConsulAPIWrapper(consulClient),
			&tagit.CmdExecutor{},
			cmd.InheritedFlags().Lookup("service-id").Value.String(),
			cmd.InheritedFlags().Lookup("script").Value.String(),
			validInterval,
			cmd.InheritedFlags().Lookup("tag-prefix").Value.String(),
		)
		t.Run(ctx)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
