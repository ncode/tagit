package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Help command",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "Unknown subcommand",
			args:        []string{"invalid-command"},
			expectError: false, // Unknown subcommands just show help, don't error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test version of the root command to avoid affecting global state
			testRootCmd := &cobra.Command{
				Use:   "tagit",
				Short: "Update consul services with dynamic tags coming from a script",
			}
			testRootCmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
			testRootCmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
			testRootCmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
			testRootCmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
			testRootCmd.PersistentFlags().StringP("interval", "i", "60s", "interval to run the script")
			testRootCmd.PersistentFlags().StringP("token", "t", "", "consul token")

			var buf bytes.Buffer
			testRootCmd.SetOut(&buf)
			testRootCmd.SetErr(&buf)
			testRootCmd.SetArgs(tt.args)

			err := testRootCmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	// Save original environment
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	tests := []struct {
		name         string
		setupConfig  func() (string, func()) // Returns config file path and cleanup function
		expectError  bool
		expectedVals map[string]string
	}{
		{
			name: "No config file",
			setupConfig: func() (string, func()) {
				// Create a temporary home directory
				tempDir, err := os.MkdirTemp("", "tagit-test-home")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				os.Setenv("HOME", tempDir)
				return "", func() { os.RemoveAll(tempDir) }
			},
			expectError: false,
		},
		{
			name: "Valid config file",
			setupConfig: func() (string, func()) {
				// Create a temporary home directory
				tempDir, err := os.MkdirTemp("", "tagit-test-home")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				os.Setenv("HOME", tempDir)

				// Create a config file
				configPath := filepath.Join(tempDir, ".tagit.yaml")
				configContent := `consul-addr: "localhost:8500"
service-id: "test-service"
script: "/tmp/test.sh"
tag-prefix: "test"
interval: "30s"
token: "test-token"
`
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}

				return configPath, func() { os.RemoveAll(tempDir) }
			},
			expectError: false,
			expectedVals: map[string]string{
				"consul-addr": "localhost:8500",
				"service-id":  "test-service",
				"script":      "/tmp/test.sh",
				"tag-prefix":  "test",
				"interval":    "30s",
				"token":       "test-token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()

			configPath, cleanup := tt.setupConfig()
			defer cleanup()

			// Set config file if provided
			if configPath != "" {
				viper.SetConfigFile(configPath)
			}

			// Call initConfig
			initConfig()

			// Verify expected values
			for key, expectedVal := range tt.expectedVals {
				actualVal := viper.GetString(key)
				assert.Equal(t, expectedVal, actualVal, "Config value for %q should be %q but got %q", key, expectedVal, actualVal)
			}
		})
	}
}

func TestRootCmdFlags(t *testing.T) {
	// Test that all expected flags are defined
	expectedFlags := []struct {
		name         string
		shorthand    string
		defaultValue string
		required     bool
	}{
		{"consul-addr", "c", "127.0.0.1:8500", false},
		{"service-id", "s", "", true},
		{"script", "x", "", true},
		{"tag-prefix", "p", "tagged", false},
		{"interval", "i", "60s", false},
		{"token", "t", "", false},
	}

	for _, flag := range expectedFlags {
		t.Run(flag.name, func(t *testing.T) {
			f := rootCmd.PersistentFlags().Lookup(flag.name)
			assert.NotNil(t, f, "Flag %q should be defined", flag.name)

			if f != nil {
				assert.Equal(t, flag.shorthand, f.Shorthand, "Flag %q shorthand should be %q", flag.name, flag.shorthand)
				assert.Equal(t, flag.defaultValue, f.DefValue, "Flag %q default value should be %q", flag.name, flag.defaultValue)
			}
		})
	}
}

func TestRootCmdHelp(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})
	
	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Update consul services with dynamic tags")
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "Available Commands:")
	assert.Contains(t, output, "Flags:")

	// Check that our subcommands are listed
	assert.Contains(t, output, "cleanup")
	assert.Contains(t, output, "run")
	assert.Contains(t, output, "systemd")
}