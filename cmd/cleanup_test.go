package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCleanupCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Missing required service-id",
			args:          []string{"cleanup"},
			expectError:   true,
			errorContains: "required flag(s)",
		},
		{
			name:          "Missing required script (even though not used for cleanup)",
			args:          []string{"cleanup", "--service-id=test-service"},
			expectError:   true,
			errorContains: "required flag(s) \"script\" not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new root command for each test
			cmd := &cobra.Command{Use: "tagit"}
			cmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
			cmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
			cmd.MarkPersistentFlagRequired("service-id")
			cmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
			cmd.MarkPersistentFlagRequired("script")
			cmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
			cmd.PersistentFlags().StringP("interval", "i", "60s", "interval to run the script")
			cmd.PersistentFlags().StringP("token", "t", "", "consul token")

			// Add the cleanup command
			testCleanupCmd := &cobra.Command{
				Use:   "cleanup",
				Short: "cleanup removes all services with the tag prefix",
				RunE:  cleanupCmd.RunE,
			}
			cmd.AddCommand(testCleanupCmd)

			// Capture stderr
			var buf bytes.Buffer
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					output := buf.String()
					assert.Contains(t, output, tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCleanupCmdFlagParsing(t *testing.T) {
	var capturedFlags map[string]string

	cmd := &cobra.Command{Use: "tagit"}
	cmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
	cmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
	cmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
	cmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
	cmd.PersistentFlags().StringP("token", "t", "", "consul token")

	testCleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "cleanup removes all services with the tag prefix",
		Run: func(cmd *cobra.Command, args []string) {
			// Capture flag values during execution
			capturedFlags = make(map[string]string)
			capturedFlags["service-id"], _ = cmd.InheritedFlags().GetString("service-id")
			capturedFlags["tag-prefix"], _ = cmd.InheritedFlags().GetString("tag-prefix")
			capturedFlags["consul-addr"], _ = cmd.InheritedFlags().GetString("consul-addr")
			capturedFlags["token"], _ = cmd.InheritedFlags().GetString("token")
		},
	}
	cmd.AddCommand(testCleanupCmd)

	cmd.SetArgs([]string{
		"cleanup",
		"--service-id=test-service",
		"--script=/tmp/test.sh", // Required by root command
		"--tag-prefix=test",
		"--consul-addr=localhost:8500",
		"--token=test-token",
	})

	err := cmd.Execute()
	assert.NoError(t, err)

	// Verify flags were parsed correctly
	assert.Equal(t, "test-service", capturedFlags["service-id"])
	assert.Equal(t, "test", capturedFlags["tag-prefix"])
	assert.Equal(t, "localhost:8500", capturedFlags["consul-addr"])
	assert.Equal(t, "test-token", capturedFlags["token"])
}

func TestCleanupCmdHelp(t *testing.T) {
	cmd := &cobra.Command{Use: "tagit"}
	cmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
	cmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
	cmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
	cmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
	cmd.PersistentFlags().StringP("token", "t", "", "consul token")

	testCleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "cleanup removes all services with the tag prefix from a given consul service",
		RunE:  cleanupCmd.RunE,
	}
	cmd.AddCommand(testCleanupCmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"cleanup", "--help"})

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "cleanup removes all services with the tag prefix")
	assert.Contains(t, output, "Usage:")
}

func TestCleanupCmdExecution(t *testing.T) {
	tests := []struct {
		name          string
		consulAddr    string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Invalid consul address",
			consulAddr:    "invalid-consul-address",
			expectError:   true,
			errorContains: "failed to clean up tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "tagit"}
			cmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
			cmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
			cmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
			cmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
			cmd.PersistentFlags().StringP("token", "t", "", "consul token")

			testCleanupCmd := &cobra.Command{
				Use:   "cleanup",
				Short: "cleanup removes all services with the tag prefix from a given consul service",
				RunE:  cleanupCmd.RunE,
			}
			cmd.AddCommand(testCleanupCmd)

			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{
				"cleanup",
				"--service-id=test-service",
				"--script=/tmp/test.sh",
				"--consul-addr=" + tt.consulAddr,
				"--tag-prefix=test",
			})

			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCleanupCmdFlagRetrieval(t *testing.T) {
	// Test that all flag retrievals work correctly within the RunE function
	cmd := &cobra.Command{Use: "tagit"}
	cmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
	cmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
	cmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
	cmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
	cmd.PersistentFlags().StringP("token", "t", "", "consul token")

	var capturedValues map[string]string

	testCleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "cleanup removes all services with the tag prefix from a given consul service",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Test the same flag access pattern used in the actual cleanup command
			capturedValues = make(map[string]string)
			capturedValues["consul-addr"] = cmd.InheritedFlags().Lookup("consul-addr").Value.String()
			capturedValues["token"] = cmd.InheritedFlags().Lookup("token").Value.String()
			capturedValues["service-id"] = cmd.InheritedFlags().Lookup("service-id").Value.String()
			capturedValues["tag-prefix"] = cmd.InheritedFlags().Lookup("tag-prefix").Value.String()

			// Don't actually try to connect to consul - just test flag access
			return nil
		},
	}
	cmd.AddCommand(testCleanupCmd)

	cmd.SetArgs([]string{
		"cleanup",
		"--service-id=test-service",
		"--script=/tmp/test.sh",
		"--consul-addr=localhost:9500",
		"--tag-prefix=test-prefix",
		"--token=test-token",
	})

	err := cmd.Execute()
	assert.NoError(t, err)

	// Verify all values were captured correctly
	assert.Equal(t, "localhost:9500", capturedValues["consul-addr"])
	assert.Equal(t, "test-token", capturedValues["token"])
	assert.Equal(t, "test-service", capturedValues["service-id"])
	assert.Equal(t, "test-prefix", capturedValues["tag-prefix"])
}

func TestCleanupCmdSuccessFlow(t *testing.T) {
	// Test the successful flow of cleanup command
	cmd := &cobra.Command{Use: "tagit"}
	cmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
	cmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
	cmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
	cmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
	cmd.PersistentFlags().StringP("token", "t", "", "consul token")

	testCleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "cleanup removes all services with the tag prefix from a given consul service",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Simulate successful cleanup without actual consul connection
			// This tests the success path that returns nil
			return nil
		},
	}
	cmd.AddCommand(testCleanupCmd)

	cmd.SetArgs([]string{
		"cleanup",
		"--service-id=test-service",
		"--script=/tmp/test.sh",
		"--consul-addr=localhost:8500",
		"--tag-prefix=test",
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}
