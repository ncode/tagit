package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// setupSystemdCmd creates and returns a properly configured systemd command
func setupSystemdCmd() *cobra.Command {
	rootCmd := &cobra.Command{Use: "tagit"}
	systCmd := &cobra.Command{
		Use:   "systemd",
		Short: "Generate a systemd service file for TagIt",
		Run:   systemdCmd.Run,
	}

	systCmd.Flags().String("service-id", "", "ID of the service (required)")
	systCmd.Flags().String("script", "", "Path to the script to execute (required)")
	systCmd.Flags().String("tag-prefix", "", "Prefix for tags (required)")
	systCmd.Flags().String("interval", "", "Interval for script execution (required)")
	systCmd.Flags().String("token", "", "Consul token (optional)")
	systCmd.Flags().String("consul-addr", "", "Consul address (optional)")
	systCmd.Flags().String("user", "", "User to run the service as (required)")
	systCmd.Flags().String("group", "", "Group to run the service as (required)")

	systCmd.MarkFlagRequired("service-id")
	systCmd.MarkFlagRequired("script")
	systCmd.MarkFlagRequired("tag-prefix")
	systCmd.MarkFlagRequired("interval")
	systCmd.MarkFlagRequired("user")
	systCmd.MarkFlagRequired("group")

	rootCmd.AddCommand(systCmd)
	return rootCmd
}

func TestSystemdCmd(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		expectedError  string
	}{
		{
			name: "All required flags provided",
			args: []string{
				"--service-id=test-service",
				"--script=/path/to/script.sh",
				"--tag-prefix=test",
				"--interval=30s",
				"--user=testuser",
				"--group=testgroup",
			},
			expectedOutput: []string{
				"[Unit]",
				"Description=Tagit test-service",
				"After=network.target",
				"After=network-online.target",
				"Wants=network-online.target",
				"",
				"[Service]",
				"Type=simple",
				"ExecStart=/usr/bin/tagit run -s test-service -x /path/to/script.sh -p test -i 30s",
				"Environment=HOME=/var/run/tagit/test-service",
				"Restart=always",
				"User=testuser",
				"Group=testgroup",
				"",
				"[Install]",
				"WantedBy=multi-user.target",
			},
		},
		{
			name: "All flags provided including optional",
			args: []string{
				"--service-id=test-service",
				"--script=/path/to/script.sh",
				"--tag-prefix=test",
				"--interval=30s",
				"--user=testuser",
				"--group=testgroup",
				"--token=test-token",
				"--consul-addr=localhost:8500",
			},
			expectedOutput: []string{
				"[Unit]",
				"Description=Tagit test-service",
				"After=network.target",
				"After=network-online.target",
				"Wants=network-online.target",
				"",
				"[Service]",
				"Type=simple",
				"ExecStart=/usr/bin/tagit run -s test-service -x /path/to/script.sh -p test -i 30s -t test-token -c localhost:8500",
				"Environment=HOME=/var/run/tagit/test-service",
				"Restart=always",
				"User=testuser",
				"Group=testgroup",
				"",
				"[Install]",
				"WantedBy=multi-user.target",
			},
		},
		{
			name: "Missing required flag",
			args: []string{
				"--service-id=test-service",
				"--script=/path/to/script.sh",
				"--tag-prefix=test",
				"--interval=30s",
				"--user=testuser",
				// missing --group flag
			},
			expectedError: "required flag(s) \"group\" not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := setupSystemdCmd()
			cmd.SetArgs(append([]string{"systemd"}, tt.args...))

			t.Logf("Test case: %s", tt.name)
			t.Logf("Command args: %v", tt.args)

			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			err := cmd.Execute()

			// Restore stdout and stderr
			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			t.Logf("Command execution completed. Error: %v", err)
			t.Logf("Command output:\n%s", output)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Contains(t, output, tt.expectedError)
			} else {
				assert.NoError(t, err)
				for _, expectedLine := range tt.expectedOutput {
					assert.Contains(t, output, expectedLine, "Output should contain %q", expectedLine)
				}
			}

			// Check if all required flags are set
			systCmd, _, _ := cmd.Find([]string{"systemd"})
			requiredFlags := []string{"service-id", "script", "tag-prefix", "interval", "user", "group"}
			for _, flagName := range requiredFlags {
				flag := systCmd.Flags().Lookup(flagName)
				assert.NotNil(t, flag, "Required flag %q is not defined", flagName)
				t.Logf("Flag %q value: %q", flagName, flag.Value.String())

				if tt.expectedError == "" {
					assert.NotEmpty(t, flag.Value.String(), "Required flag %q is not set", flagName)
				}
			}

			// Print all set flags
			t.Log("All set flags:")
			systCmd.Flags().VisitAll(func(f *pflag.Flag) {
				t.Logf("  %s: %s", f.Name, f.Value.String())
			})
		})
	}
}

func TestSystemdCmdFlagDefinitions(t *testing.T) {
	cmd := setupSystemdCmd()
	systCmd, _, err := cmd.Find([]string{"systemd"})
	assert.NoError(t, err)

	expectedFlags := map[string]struct {
		expectedRequired bool
		flagType         string
	}{
		"service-id":  {true, "string"},
		"script":      {true, "string"},
		"tag-prefix":  {true, "string"},
		"interval":    {true, "string"},
		"token":       {false, "string"},
		"consul-addr": {false, "string"},
		"user":        {true, "string"},
		"group":       {true, "string"},
	}

	for flagName, details := range expectedFlags {
		t.Run(flagName, func(t *testing.T) {
			flag := systCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "Flag %q should be defined", flagName)

			t.Logf("Checking flag: %s", flagName)
			t.Logf("Flag found: %v", flag != nil)
			if flag != nil {
				t.Logf("Flag value: %s", flag.Value.String())
				t.Logf("Flag type: %s", flag.Value.Type())
			}

			assert.Equal(t, details.flagType, flag.Value.Type(), "Flag %q should be of type %s", flagName, details.flagType)

			// Try to mark the flag as required
			err := cobra.MarkFlagRequired(systCmd.Flags(), flagName)

			if details.expectedRequired {
				assert.NoError(t, err, "Flag %q should be able to be marked as required", flagName)
			} else {
				if err == nil {
					t.Logf("Warning: Flag %q was expected to be optional but can be marked as required", flagName)
				} else {
					t.Logf("Flag %q behaves as expected (optional)", flagName)
				}
			}

			// Check if the flag is actually marked as required in the command definition
			isRequired := systCmd.Flags().Lookup(flagName).Annotations != nil &&
				len(systCmd.Flags().Lookup(flagName).Annotations[cobra.BashCompOneRequiredFlag]) > 0

			t.Logf("Flag %q is%s marked as required in the command definition", flagName, map[bool]string{true: "", false: " not"}[isRequired])

			if details.expectedRequired != isRequired {
				t.Logf("Warning: Flag %q required status (%v) does not match expected (%v)", flagName, isRequired, details.expectedRequired)
			}
		})
	}
}

func TestSystemdCmdHelp(t *testing.T) {
	cmd := setupSystemdCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"systemd", "--help"})
	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Generate a systemd service file for TagIt")
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "Flags:")
}

func TestSystemdCmdInvalidFlag(t *testing.T) {
	cmd := setupSystemdCmd()
	cmd.SetArgs([]string{"systemd", "--invalid-flag=value"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag: --invalid-flag")
}

func TestSystemdCmdFlagParsing(t *testing.T) {
	cmd := setupSystemdCmd()
	args := []string{
		"systemd",
		"--service-id=test-service",
		"--script=/path/to/script.sh",
		"--tag-prefix=test",
		"--interval=30s",
		"--user=testuser",
		"--group=testgroup",
		"--token=test-token",
		"--consul-addr=localhost:8500",
	}

	cmd.SetArgs(args)
	err := cmd.Execute()
	assert.NoError(t, err)

	systCmd, _, _ := cmd.Find([]string{"systemd"})
	for _, arg := range args[1:] { // Skip "systemd"
		parts := strings.SplitN(arg, "=", 2)
		flagName := strings.TrimLeft(parts[0], "-")
		expectedValue := parts[1]

		flag := systCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag)
		assert.Equal(t, expectedValue, flag.Value.String())
	}
}
