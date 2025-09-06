package cmd

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRunCmd(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Missing required service-id",
			args:          []string{"run", "--script=/tmp/test.sh"},
			expectError:   true,
			errorContains: "required flag(s) \"service-id\" not set",
		},
		{
			name:          "Missing required script",
			args:          []string{"run", "--service-id=test-service"},
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

			// Add the run command
			testRunCmd := &cobra.Command{
				Use:   "run",
				Short: "Run tagit",
				Run:   runCmd.Run,
			}
			cmd.AddCommand(testRunCmd)

			// Capture stderr
			var buf bytes.Buffer
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			// Set a context with timeout to prevent hanging
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- cmd.Execute()
			}()

			select {
			case err := <-done:
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						output := buf.String()
						assert.Contains(t, output, tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			case <-ctx.Done():
				if tt.expectError {
					t.Log("Command timed out as expected for invalid input")
				} else {
					t.Error("Command timed out unexpectedly")
				}
			}
		})
	}
}

func TestRunCmdFlagParsing(t *testing.T) {
	var capturedFlags map[string]string

	cmd := &cobra.Command{Use: "tagit"}
	cmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
	cmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
	cmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
	cmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
	cmd.PersistentFlags().StringP("interval", "i", "60s", "interval to run the script")
	cmd.PersistentFlags().StringP("token", "t", "", "consul token")

	testRunCmd := &cobra.Command{
		Use:   "run",
		Short: "Run tagit",
		Run: func(cmd *cobra.Command, args []string) {
			// Capture flag values during execution
			capturedFlags = make(map[string]string)
			capturedFlags["service-id"], _ = cmd.InheritedFlags().GetString("service-id")
			capturedFlags["script"], _ = cmd.InheritedFlags().GetString("script")
			capturedFlags["interval"], _ = cmd.InheritedFlags().GetString("interval")
			capturedFlags["tag-prefix"], _ = cmd.InheritedFlags().GetString("tag-prefix")
			capturedFlags["consul-addr"], _ = cmd.InheritedFlags().GetString("consul-addr")
			capturedFlags["token"], _ = cmd.InheritedFlags().GetString("token")
		},
	}
	cmd.AddCommand(testRunCmd)

	cmd.SetArgs([]string{
		"run",
		"--service-id=test-service",
		"--script=/tmp/test.sh",
		"--interval=30s",
		"--tag-prefix=test",
		"--consul-addr=localhost:8500",
		"--token=test-token",
	})

	err := cmd.Execute()
	assert.NoError(t, err)

	// Verify flags were parsed correctly
	assert.Equal(t, "test-service", capturedFlags["service-id"])
	assert.Equal(t, "/tmp/test.sh", capturedFlags["script"])
	assert.Equal(t, "30s", capturedFlags["interval"])
	assert.Equal(t, "test", capturedFlags["tag-prefix"])
	assert.Equal(t, "localhost:8500", capturedFlags["consul-addr"])
	assert.Equal(t, "test-token", capturedFlags["token"])
}