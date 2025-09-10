package tagit

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/ncode/tagit/pkg/consul"
	"github.com/stretchr/testify/assert"
)

// MockConsulClient implements the Client interface for testing.
type MockConsulClient struct {
	MockAgent *MockAgent
}

func (m *MockConsulClient) Agent() consul.Agent {
	return m.MockAgent
}

// MockAgent simulates the Agent part of the Consul client.
type MockAgent struct {
	ServiceFunc         func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error)
	ServiceRegisterFunc func(reg *api.AgentServiceRegistration) error
}

func (m *MockAgent) Service(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
	return m.ServiceFunc(serviceID, q)
}

func (m *MockAgent) ServiceRegister(reg *api.AgentServiceRegistration) error {
	return m.ServiceRegisterFunc(reg)
}

type MockCommandExecutor struct {
	MockOutput []byte
	MockError  error
}

func (m *MockCommandExecutor) Execute(command string) ([]byte, error) {
	return m.MockOutput, m.MockError
}

func TestDiffTags(t *testing.T) {
	tests := []struct {
		name     string
		current  []string
		update   []string
		expected []string
	}{
		{
			name:     "No Difference",
			current:  []string{"tag1", "tag2", "tag3"},
			update:   []string{"tag1", "tag2", "tag3"},
			expected: []string{},
		},
		{
			name:     "Difference In Current",
			current:  []string{"tag1", "tag2", "tag4"},
			update:   []string{"tag1", "tag2", "tag3"},
			expected: []string{"tag3", "tag4"},
		},
		{
			name:     "Difference In Update",
			current:  []string{"tag1", "tag2"},
			update:   []string{"tag1", "tag2", "tag3"},
			expected: []string{"tag3"},
		},
		{
			name:     "Empty Current",
			current:  []string{},
			update:   []string{"tag1", "tag2"},
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "Empty Update",
			current:  []string{"tag1", "tag2"},
			update:   []string{},
			expected: []string{"tag1", "tag2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagit := TagIt{}
			diff := tagit.diffTags(tt.current, tt.update)
			sort.Strings(diff)
			sort.Strings(tt.expected)
			assert.Equal(t, tt.expected, diff, "diffTags() returned unexpected result")
		})
	}
}

func TestExcludeTagged(t *testing.T) {
	tests := []struct {
		name      string
		tags      []string
		tagPrefix string
		expected  []string
		shouldTag bool
	}{
		{
			name:      "No Tags With Prefix",
			tags:      []string{"alpha", "beta", "gamma"},
			tagPrefix: "tag",
			expected:  []string{"alpha", "beta", "gamma"},
			shouldTag: false,
		},
		{
			name:      "All Tags With Prefix",
			tags:      []string{"tag-alpha", "tag-beta", "tag-gamma"},
			tagPrefix: "tag",
			expected:  []string{},
			shouldTag: true,
		},
		{
			name:      "Some Tags With Prefix",
			tags:      []string{"alpha", "tag-beta", "gamma"},
			tagPrefix: "tag",
			expected:  []string{"alpha", "gamma"},
			shouldTag: true,
		},
		{
			name:      "Empty Tags",
			tags:      []string{},
			tagPrefix: "tag",
			expected:  []string{},
			shouldTag: false,
		},
		{
			name:      "Prefix in Middle",
			tags:      []string{"alpha-tag", "beta", "gamma"},
			tagPrefix: "tag",
			expected:  []string{"alpha-tag", "beta", "gamma"},
			shouldTag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagit := TagIt{TagPrefix: tt.tagPrefix}
			filteredTags, tagged := tagit.excludeTagged(tt.tags)
			assert.Equal(t, tt.expected, filteredTags, "excludeTagged() returned unexpected filtered tags")
			assert.Equal(t, tt.shouldTag, tagged, "excludeTagged() returned unexpected shouldTag value")
		})
	}
}

func TestNeedsTag(t *testing.T) {
	tests := []struct {
		name           string
		current        []string
		update         []string
		expectedTags   []string
		expectedShould bool
	}{
		{
			name:           "No Update Needed",
			current:        []string{"tag-tag1", "tag-tag2", "tag-tag3"},
			update:         []string{"tag-tag1", "tag-tag2", "tag-tag3"},
			expectedTags:   nil,
			expectedShould: false,
		},
		{
			name:           "Update Needed",
			current:        []string{"tag-tag1", "tag-tag2"},
			update:         []string{"tag-tag1", "tag-tag2", "tag3"},
			expectedTags:   []string{"tag-tag1", "tag-tag2", "tag3"},
			expectedShould: true,
		},
		{
			name:           "All New Tags",
			current:        []string{},
			update:         []string{"tag1", "tag2", "tag3"},
			expectedTags:   []string{"tag1", "tag2", "tag3"},
			expectedShould: true,
		},
		{
			name:           "Current Tags Removed",
			current:        []string{"tag-tag1", "tag2", "tag3"},
			update:         []string{},
			expectedTags:   []string{"tag2", "tag3"},
			expectedShould: true,
		},
		{
			name:           "Mixed Changes",
			current:        []string{"tag-tag1", "tag2", "tag4"},
			update:         []string{"tag2", "tag3", "tag5"},
			expectedTags:   []string{"tag2", "tag3", "tag4", "tag5"},
			expectedShould: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagit := TagIt{TagPrefix: "tag"}
			filteredTags, shouldTag := tagit.needsTag(tt.current, tt.update)
			assert.Equal(t, tt.expectedTags, filteredTags, "needsTag() returned unexpected filtered tags")
			assert.Equal(t, tt.expectedShould, shouldTag, "needsTag() returned unexpected shouldTag value")
		})
	}
}

func TestCopyServiceToRegistration(t *testing.T) {
	tests := []struct {
		name        string
		service     *api.AgentService
		expectedReg *api.AgentServiceRegistration
	}{
		{
			name: "Copy All Fields",
			service: &api.AgentService{
				ID:      "service-1",
				Service: "test-service",
				Tags:    []string{"tag1", "tag2"},
				Port:    8080,
				Address: "127.0.0.1",
				Kind:    api.ServiceKindTypical,
				Weights: api.AgentWeights{
					Passing: 10,
					Warning: 1,
				},
				Meta: map[string]string{"version": "1.0"},
			},
			expectedReg: &api.AgentServiceRegistration{
				ID:      "service-1",
				Name:    "test-service",
				Tags:    []string{"tag1", "tag2"},
				Port:    8080,
				Address: "127.0.0.1",
				Kind:    api.ServiceKindTypical,
				Weights: &api.AgentWeights{
					Passing: 10,
					Warning: 1,
				},
				Meta: map[string]string{"version": "1.0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagit := TagIt{}
			reg := tagit.copyServiceToRegistration(tt.service)
			assert.Equal(t, tt.expectedReg, reg, "copyServiceToRegistration() returned unexpected result")
		})
	}
}

func TestRunScript(t *testing.T) {
	tests := []struct {
		name       string
		script     string
		mockOutput string
		mockError  error
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "Valid Command",
			script:     "echo test",
			mockOutput: "test\n",
			wantOutput: "test\n",
			wantErr:    false,
		},
		{
			name:      "Invalid Command",
			script:    "someinvalidcommand",
			mockError: fmt.Errorf("command failed"),
			wantErr:   true,
		},
		{
			name:       "Empty Command",
			script:     "",
			mockOutput: "",
			wantOutput: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockCommandExecutor{
				MockOutput: []byte(tt.mockOutput),
				MockError:  tt.mockError,
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			tagit := TagIt{Script: tt.script, commandExecutor: mockExecutor, logger: logger}

			output, err := tagit.runScript()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOutput, string(output))
			}
		})
	}
}

func TestNew(t *testing.T) {
	mockConsulClient := &MockConsulClient{}
	mockCommandExecutor := &MockCommandExecutor{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tagit := New(mockConsulClient, mockCommandExecutor, "test-service", "echo test", 30*time.Second, "test-prefix", logger)

	assert.NotNil(t, tagit, "New() returned nil")
	assert.NotNil(t, tagit.client, "TagIt client is nil")
	assert.NotNil(t, tagit.commandExecutor, "TagIt commandExecutor is nil")
	assert.Equal(t, "test-service", tagit.ServiceID, "Unexpected ServiceID")
	assert.Equal(t, "echo test", tagit.Script, "Unexpected Script")
	assert.Equal(t, 30*time.Second, tagit.Interval, "Unexpected Interval")
	assert.Equal(t, "test-prefix", tagit.TagPrefix, "Unexpected TagPrefix")
}

func TestGetService(t *testing.T) {
	tests := []struct {
		name             string
		serviceID        string
		mockServicesData map[string]*api.AgentService
		mockServicesErr  error
		expectErr        bool
		expectService    *api.AgentService
	}{
		{
			name:      "Service Found",
			serviceID: "test-service",
			mockServicesData: map[string]*api.AgentService{
				"test-service": {
					ID:      "test-service",
					Service: "test",
					Tags:    []string{"tag1", "tag2"},
				},
			},
			expectErr:     false,
			expectService: &api.AgentService{ID: "test-service", Service: "test", Tags: []string{"tag1", "tag2"}},
		},
		{
			name:             "Service Not Found",
			serviceID:        "nonexistent-service",
			mockServicesData: map[string]*api.AgentService{},
			expectErr:        true,
			expectService:    nil,
		},
		{
			name:            "Consul Client Error",
			serviceID:       "test-service",
			mockServicesErr: fmt.Errorf("consul client error"),
			expectErr:       true,
			expectService:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConsulClient := &MockConsulClient{
				MockAgent: &MockAgent{
					ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
						if tt.mockServicesErr != nil {
							return nil, nil, tt.mockServicesErr
						}
						service, ok := tt.mockServicesData[serviceID]
						if !ok {
							return nil, nil, nil
						}
						return service, nil, nil
					},
				},
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			tagit := New(mockConsulClient, nil, tt.serviceID, "", time.Duration(0), "", logger)

			service, err := tagit.getService()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectService, service)
		})
	}
}

func TestUpdateServiceTags(t *testing.T) {
	tests := []struct {
		name             string
		mockScriptOutput string
		mockScriptError  error
		existingTags     []string
		newTags          []string
		mockRegisterErr  error
		expectError      bool
	}{
		{
			name:             "Successful Update",
			mockScriptOutput: "new-tag1 new-tag2",
			existingTags:     []string{"old-tag"},
			newTags:          []string{"tag-new-tag1", "tag-new-tag2"},
			expectError:      false,
		},
		{
			name:            "Script Error",
			mockScriptError: fmt.Errorf("script error"),
			expectError:     true,
		},
		{
			name:             "Consul Register Error",
			mockScriptOutput: "new-tag1 new-tag2",
			mockRegisterErr:  fmt.Errorf("consul error"),
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockCommandExecutor{
				MockOutput: []byte(tt.mockScriptOutput), MockError: tt.mockScriptError,
			}
			mockConsulClient := &MockConsulClient{
				MockAgent: &MockAgent{
					ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
						return &api.AgentService{
							ID:   "test-service",
							Tags: tt.existingTags,
						}, nil, nil
					},
					ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
						return tt.mockRegisterErr
					},
				},
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			tagit := New(mockConsulClient, mockExecutor, "test-service", "echo test", 30*time.Second, "tag", logger)

			err := tagit.updateServiceTags()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCleanupTags(t *testing.T) {
	tests := []struct {
		name            string
		serviceID       string
		mockServices    map[string]*api.AgentService
		tagPrefix       string
		mockRegisterErr error
		expectError     bool
		expectTags      []string
	}{
		{
			name:      "Successful Tag Cleanup",
			serviceID: "test-service",
			mockServices: map[string]*api.AgentService{
				"test-service": {
					ID:   "test-service",
					Tags: []string{"tag-prefix1", "tag-prefix2", "other-tag"},
				},
			},
			tagPrefix:   "tag",
			expectError: false,
			expectTags:  []string{"other-tag"},
		},
		{
			name:      "No Tag Cleanup needed",
			serviceID: "test-service",
			mockServices: map[string]*api.AgentService{
				"test-service": {
					ID:   "test-service",
					Tags: []string{"prefix1", "prefix2", "other-tag"},
				},
			},
			tagPrefix:   "tag",
			expectError: false,
			expectTags:  []string{"other-tag", "prefix1", "prefix2"},
		},
		{
			name:      "Service Not Found",
			serviceID: "non-existent-service",
			mockServices: map[string]*api.AgentService{
				"other-service": {
					ID:   "other-service",
					Tags: []string{"some-tag", "another-tag"},
				},
			},
			tagPrefix:   "tag-prefix",
			expectError: true,
		},
		{
			name:      "Consul Register Error",
			serviceID: "test-service",
			mockServices: map[string]*api.AgentService{
				"test-service": {
					ID:   "test-service",
					Tags: []string{"tag-prefix1", "other-tag"},
				},
			},
			tagPrefix:       "tag",
			mockRegisterErr: fmt.Errorf("consul register error"),
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConsulClient := &MockConsulClient{
				MockAgent: &MockAgent{
					ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
						service, exists := tt.mockServices[serviceID]
						if !exists {
							return nil, nil, fmt.Errorf("service not found")
						}
						return service, nil, nil
					},
					ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
						if tt.mockRegisterErr != nil {
							return tt.mockRegisterErr
						}
						// Update the mock service with the new tags
						tt.mockServices[reg.ID].Tags = reg.Tags
						return nil
					},
				},
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			tagit := New(mockConsulClient, nil, tt.serviceID, "", time.Duration(0), tt.tagPrefix, logger)

			err := tagit.CleanupTags()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				service, _ := tagit.getService()
				if service != nil {
					actualTags := service.Tags
					sort.Strings(actualTags)
					sort.Strings(tt.expectTags)
					assert.Equal(t, tt.expectTags, actualTags, "Unexpected tags after cleanup")
				}
			}
		})
	}
}

func TestRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updateServiceTagsCalled := atomic.Int32{}
	mockExecutor := &MockCommandExecutor{
		MockOutput: []byte("new-tag1 new-tag2"),
		MockError:  nil,
	}
	mockConsulClient := &MockConsulClient{
		MockAgent: &MockAgent{
			ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
				updateServiceTagsCalled.Add(1)
				if updateServiceTagsCalled.Load() == 2 {
					return nil, nil, fmt.Errorf("simulated error")
				}
				return &api.AgentService{
					ID:   "test-service",
					Tags: []string{"old-tag"},
				}, nil, nil
			},
			ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
				return nil
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tagit := New(mockConsulClient, mockExecutor, "test-service", "echo test", 100*time.Millisecond, "tag", logger)

	go tagit.Run(ctx)

	time.Sleep(350 * time.Millisecond)
	cancel()

	time.Sleep(50 * time.Millisecond)

	assert.GreaterOrEqual(t, updateServiceTagsCalled.Load(), int32(2), "Expected updateServiceTags to be called at least 2 times")
	assert.LessOrEqual(t, updateServiceTagsCalled.Load(), int32(4), "Expected updateServiceTags to be called at most 4 times")
}

func TestConsulInterfaceCompatibility(t *testing.T) {
	// Test that our mocks implement the consul package interfaces correctly
	mockAgent := &MockAgent{
		ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
			return &api.AgentService{ID: serviceID}, nil, nil
		},
		ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
			return nil
		},
	}

	mockClient := &MockConsulClient{
		MockAgent: mockAgent,
	}

	// Verify that MockConsulClient implements consul.Client
	var _ consul.Client = mockClient

	// Verify that MockAgent implements consul.Agent
	var _ consul.Agent = mockAgent

	// Test that the mock client works correctly
	agent := mockClient.Agent()
	assert.NotNil(t, agent, "Agent() should return non-nil")

	service, _, err := agent.Service("test-service", nil)
	assert.NoError(t, err)
	assert.Equal(t, "test-service", service.ID)
}

func TestCmdExecutor_Execute(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantOutput  string
		wantErr     string
		expectError bool
	}{
		{
			name:        "Valid command",
			command:     "echo test",
			wantOutput:  "test\n",
			expectError: false,
		},
		{
			name:        "Empty command",
			command:     "",
			wantErr:     "failed to execute: empty command",
			expectError: true,
		},
		{
			name:        "Command with unclosed quote",
			command:     "echo \"unclosed quote",
			wantErr:     "failed to split command:",
			expectError: true,
		},
		{
			name:        "Invalid command",
			command:     "invalidcommand",
			wantErr:     "exec: \"invalidcommand\": executable file not found in $PATH",
			expectError: true,
		},
	}

	executor := &CmdExecutor{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(tt.command)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOutput, string(output))
			}
		})
	}
}
