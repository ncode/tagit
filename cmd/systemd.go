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

	"github.com/ncode/tagit/pkg/systemd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		return systemdCommand(cmd)
	},
}

func init() {
	rootCmd.AddCommand(systemdCmd)

	addSystemdFlags(systemdCmd.Flags())

	// Mark required flags
	systemdCmd.MarkFlagRequired("service-id")
	systemdCmd.MarkFlagRequired("script")
	systemdCmd.MarkFlagRequired("tag-prefix")
	systemdCmd.MarkFlagRequired("interval")
	systemdCmd.MarkFlagRequired("user")
	systemdCmd.MarkFlagRequired("group")
}

func systemdCommand(cmd *cobra.Command) error {
	input, err := resolveSystemdInput(cmd)
	if err != nil {
		return err
	}

	fields, err := systemd.NewFieldsFromInvocation(input.Invocation, input.User, input.Group)
	if err != nil {
		return err
	}

	serviceFile, err := systemd.RenderTemplate(fields)
	if err != nil {
		return fmt.Errorf("generate systemd service file: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), serviceFile)
	return nil
}

func addSystemdFlags(flags *pflag.FlagSet) {
	flags.String("service-id", "", "ID of the service (required)")
	flags.String("script", "", "Path to the script to execute (required)")
	flags.String("tag-prefix", "", "Prefix for tags (required)")
	flags.String("interval", "", "Interval for script execution (required)")
	flags.String("token", "", "Consul token (optional)")
	flags.String("consul-addr", "", "Consul address (optional)")
	flags.String("user", "", "User to run the service as (required)")
	flags.String("group", "", "Group to run the service as (required)")
}
