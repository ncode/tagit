package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/ncode/tagit/pkg/consul"
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
			errorContains: "service-id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new root command for each test
			cmd := &cobra.Command{Use: "tagit"}
			cmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
			cmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
			cmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
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
					assert.Contains(t, err.Error(), tt.errorContains)
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
	// Since the actual cleanupCmd creates a real Consul client internally,
	// we test with a mock command that simulates the successful path
	cmd := &cobra.Command{Use: "tagit"}
	cmd.PersistentFlags().StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
	cmd.PersistentFlags().StringP("service-id", "s", "", "consul service id")
	cmd.PersistentFlags().StringP("script", "x", "", "path to script used to generate tags")
	cmd.PersistentFlags().StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
	cmd.PersistentFlags().StringP("token", "t", "", "consul token")

	var logOutput bytes.Buffer
	testCleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "cleanup removes all services with the tag prefix from a given consul service",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Verify all required flags are accessible
			serviceID := cmd.InheritedFlags().Lookup("service-id").Value.String()
			tagPrefix := cmd.InheritedFlags().Lookup("tag-prefix").Value.String()
			consulAddr := cmd.InheritedFlags().Lookup("consul-addr").Value.String()
			token := cmd.InheritedFlags().Lookup("token").Value.String()

			// Simulate the logging that would happen
			fmt.Fprintf(&logOutput, "Starting tag cleanup, serviceID=%s, tagPrefix=%s, consulAddr=%s\n",
				serviceID, tagPrefix, consulAddr)

			if token != "" {
				fmt.Fprintf(&logOutput, "Using token authentication\n")
			}

			// Simulate successful cleanup
			fmt.Fprintf(&logOutput, "Tag cleanup completed successfully\n")
			return nil
		},
	}
	cmd.AddCommand(testCleanupCmd)

	cmd.SetArgs([]string{
		"cleanup",
		"--service-id=test-service",
		"--consul-addr=localhost:8500",
		"--tag-prefix=test",
		"--token=secret-token",
	})

	err := cmd.Execute()
	assert.NoError(t, err)

	// Verify the command would have executed with the right parameters
	output := logOutput.String()
	assert.Contains(t, output, "serviceID=test-service")
	assert.Contains(t, output, "tagPrefix=test")
	assert.Contains(t, output, "consulAddr=localhost:8500")
	assert.Contains(t, output, "Using token authentication")
	assert.Contains(t, output, "Tag cleanup completed successfully")
}

// MockConsulClient for testing
type MockConsulClient struct {
	MockAgent consul.Agent
}

func (m *MockConsulClient) Agent() consul.Agent {
	return m.MockAgent
}

// MockAgent implements the Agent interface
type MockAgent struct {
	ServiceFunc         func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error)
	ServiceRegisterFunc func(reg *api.AgentServiceRegistration) error
}

func (m *MockAgent) Service(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
	if m.ServiceFunc != nil {
		return m.ServiceFunc(serviceID, q)
	}
	return &api.AgentService{
		ID:      "test-service",
		Service: "test",
		Tags:    []string{"tagged:old", "other-tag"},
	}, nil, nil
}

func (m *MockAgent) ServiceRegister(reg *api.AgentServiceRegistration) error {
	if m.ServiceRegisterFunc != nil {
		return m.ServiceRegisterFunc(reg)
	}
	return nil
}

func TestCleanupCmdWithMockFactory(t *testing.T) {
	// Save and restore the original factory
	originalFactory := consul.Factory
	defer func() {
		consul.Factory = originalFactory
	}()

	t.Run("Successful cleanup with mock", func(t *testing.T) {
		// Create a mock agent that simulates a service with tags
		mockAgent := &MockAgent{
			ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
				return &api.AgentService{
					ID:      serviceID,
					Service: "test",
					Tags:    []string{"tagged-value1", "tagged-value2", "other-tag"},
				}, nil, nil
			},
			ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
				// Verify that the tags were cleaned up
				assert.Equal(t, "test-service", reg.ID)
				assert.NotContains(t, reg.Tags, "tagged-value1")
				assert.NotContains(t, reg.Tags, "tagged-value2")
				assert.Contains(t, reg.Tags, "other-tag")
				return nil
			},
		}

		// Create mock client with the mock agent
		mockClient := &MockConsulClient{
			MockAgent: mockAgent,
		}

		// Set up the mock factory
		mockFactory := &consul.MockFactory{
			MockClient: mockClient,
		}
		consul.SetFactory(mockFactory)

		// Create a new command instance for this test
		cmd := &cobra.Command{
			Use:  "cleanup",
			RunE: cleanupCmd.RunE,
		}
		// Set up parent command for flags inheritance
		parent := &cobra.Command{}
		parent.PersistentFlags().String("consul-addr", "127.0.0.1:8500", "")
		parent.PersistentFlags().String("token", "", "")
		parent.PersistentFlags().String("service-id", "test-service", "")
		parent.PersistentFlags().String("tag-prefix", "tagged", "")
		parent.AddCommand(cmd)

		// Run the actual cleanup command
		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("Cleanup with connection error", func(t *testing.T) {
		// Set up a factory that returns an error
		mockFactory := &consul.MockFactory{
			MockError: fmt.Errorf("connection failed"),
		}
		consul.SetFactory(mockFactory)

		// Create a new command instance for this test
		cmd := &cobra.Command{
			Use:  "cleanup",
			RunE: cleanupCmd.RunE,
		}
		// Set up parent command for flags inheritance
		parent := &cobra.Command{}
		parent.PersistentFlags().String("consul-addr", "127.0.0.1:8500", "")
		parent.PersistentFlags().String("token", "", "")
		parent.PersistentFlags().String("service-id", "test-service", "")
		parent.PersistentFlags().String("tag-prefix", "tagged", "")
		parent.AddCommand(cmd)

		// Run the cleanup command - should fail
		err := cmd.RunE(cmd, []string{})
		assert.Error(t, err)
	})

	t.Run("Cleanup with service not found", func(t *testing.T) {
		// Create a mock agent that returns nil service
		mockAgent := &MockAgent{
			ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
				return nil, nil, nil
			},
		}

		// Create mock client with the mock agent
		mockClient := &MockConsulClient{
			MockAgent: mockAgent,
		}

		// Set up the mock factory
		mockFactory := &consul.MockFactory{
			MockClient: mockClient,
		}
		consul.SetFactory(mockFactory)

		// Create a new command instance for this test
		cmd := &cobra.Command{
			Use:  "cleanup",
			RunE: cleanupCmd.RunE,
		}
		// Set up parent command for flags inheritance
		parent := &cobra.Command{}
		parent.PersistentFlags().String("consul-addr", "127.0.0.1:8500", "")
		parent.PersistentFlags().String("token", "", "")
		parent.PersistentFlags().String("service-id", "test-service", "")
		parent.PersistentFlags().String("tag-prefix", "tagged", "")
		parent.AddCommand(cmd)

		// Run the cleanup command - should fail
		err := cmd.RunE(cmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service test-service not found")
	})

	t.Run("Cleanup with service register error", func(t *testing.T) {
		// Create a mock agent that simulates a service with tags but fails on register
		mockAgent := &MockAgent{
			ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
				return &api.AgentService{
					ID:      serviceID,
					Service: "test",
					Tags:    []string{"tagged-value1", "other-tag"},
				}, nil, nil
			},
			ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
				return fmt.Errorf("failed to register service")
			},
		}

		// Create mock client with the mock agent
		mockClient := &MockConsulClient{
			MockAgent: mockAgent,
		}

		// Set up the mock factory
		mockFactory := &consul.MockFactory{
			MockClient: mockClient,
		}
		consul.SetFactory(mockFactory)

		// Create a new command instance for this test
		cmd := &cobra.Command{
			Use:  "cleanup",
			RunE: cleanupCmd.RunE,
		}
		// Set up parent command for flags inheritance
		parent := &cobra.Command{}
		parent.PersistentFlags().String("consul-addr", "127.0.0.1:8500", "")
		parent.PersistentFlags().String("token", "", "")
		parent.PersistentFlags().String("service-id", "test-service", "")
		parent.PersistentFlags().String("tag-prefix", "tagged", "")
		parent.AddCommand(cmd)

		// Run the cleanup command - should fail
		err := cmd.RunE(cmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clean up tags")
	})
}
