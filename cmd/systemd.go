/*
Copyright © 2024 Juliano Martinez

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
	"os"

	"github.com/ncode/tagit/pkg/systemd"
	"github.com/spf13/cobra"
)

// systemdCmd represents the systemd command
var systemdCmd = &cobra.Command{
	Use:   "systemd",
	Short: "Generate a systemd service file for TagIt",
	Long: `The systemd command generates a systemd service file for TagIt.
This allows you to easily set up TagIt as a system service that starts
automatically on boot and can be managed using systemctl.

Example usage:
  tagit systemd --service-id=my-service --script=/path/to/script.sh --tag-prefix=tagit --interval=5s --user=tagit --group=tagit
`,
	Run: func(cmd *cobra.Command, args []string) {
		flags := make(map[string]string)
		for _, flag := range append(systemd.GetRequiredFlags(), systemd.GetOptionalFlags()...) {
			flags[flag], _ = cmd.Flags().GetString(flag)
		}

		fields, err := systemd.NewFieldsFromFlags(flags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		serviceFile, err := systemd.RenderTemplate(fields)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating systemd service file: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(serviceFile)
	},
}

func init() {
	rootCmd.AddCommand(systemdCmd)

	// Define flags for all required and optional fields
	systemdCmd.Flags().String("service-id", "", "ID of the service (required)")
	systemdCmd.Flags().String("script", "", "Path to the script to execute (required)")
	systemdCmd.Flags().String("tag-prefix", "", "Prefix for tags (required)")
	systemdCmd.Flags().String("interval", "", "Interval for script execution (required)")
	systemdCmd.Flags().String("token", "", "Consul token (optional)")
	systemdCmd.Flags().String("consul-addr", "", "Consul address (optional)")
	systemdCmd.Flags().String("user", "", "User to run the service as (required)")
	systemdCmd.Flags().String("group", "", "Group to run the service as (required)")

	// Mark required flags
	systemdCmd.MarkFlagRequired("service-id")
	systemdCmd.MarkFlagRequired("script")
	systemdCmd.MarkFlagRequired("tag-prefix")
	systemdCmd.MarkFlagRequired("interval")
	systemdCmd.MarkFlagRequired("user")
	systemdCmd.MarkFlagRequired("group")
}
