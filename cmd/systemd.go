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
	"fmt"
	"github.com/ncode/tagit/pkg/systemd"
	"github.com/spf13/cobra"
)

// systemdCmd represents the systemd command
var systemdCmd = &cobra.Command{
	Use:   "systemd",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fields := &systemd.Fields{
			User:       cmd.PersistentFlags().Lookup("user").Value.String(),
			Group:      cmd.PersistentFlags().Lookup("group").Value.String(),
			ConsulAddr: cmd.InheritedFlags().Lookup("consul-addr").Value.String(),
			ServiceID:  cmd.InheritedFlags().Lookup("service-id").Value.String(),
			Script:     cmd.InheritedFlags().Lookup("script").Value.String(),
			TagPrefix:  cmd.InheritedFlags().Lookup("tag-prefix").Value.String(),
			Interval:   cmd.InheritedFlags().Lookup("interval").Value.String(),
			Token:      cmd.InheritedFlags().Lookup("token").Value.String(),
		}
		fmt.Println(systemd.RenderTemplate(fields))
	},
}

func init() {
	rootCmd.AddCommand(systemdCmd)
	systemdCmd.PersistentFlags().StringP("user", "u", "nobody", "user to run the service")
	systemdCmd.PersistentFlags().StringP("group", "g", "nobody", "group to run the service")
}
