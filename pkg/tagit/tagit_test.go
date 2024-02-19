package tagit

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

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
			tagit := TagIt{} // Assuming TagIt doesn't require initialization for compareTags
			diff := tagit.diffTags(tt.current, tt.update)
			if (len(diff) == 0) && (len(tt.expected) == 0) {
				return
			}
			slices.Sort(diff)
			slices.Sort(tt.expected)
			if !reflect.DeepEqual(diff, tt.expected) {
				t.Errorf("compareTags(%v, %v) = %v, want %v", tt.current, tt.update, diff, tt.expected)
			}
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

			if slices.Compare(filteredTags, tt.expected) != 0 || tagged != tt.shouldTag {
				t.Errorf("excludeTagged() = %v, %v, want %v, %v", filteredTags, tagged, tt.expected, tt.shouldTag)
			}
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
			expectedTags:   []string{},
			expectedShould: false,
		},
		{
			name:           "Update Needed",
			current:        []string{"tag-tag1", "tag-tag2"},
			update:         []string{"tag-tag1", "tag-tag2", "tag3"},
			expectedTags:   []string{"tag3"},
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
			expectedTags:   []string{"tag4", "tag3", "tag5"},
			expectedShould: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagit := TagIt{TagPrefix: "tag"} // Assuming TagPrefix is set for needsTag
			filteredTags, shouldTag := tagit.needsTag(tt.current, tt.update)
			//fmt.Println(filteredTags, shouldTag)

			if slices.Compare(filteredTags, tt.expectedTags) != 0 || shouldTag != tt.expectedShould {
				t.Errorf("needsTag() = %v, %v, want %v, %v", filteredTags, shouldTag, tt.expectedTags, tt.expectedShould)
			}
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

			if !reflect.DeepEqual(reg, tt.expectedReg) {
				t.Errorf("copyServiceToRegistration() got = %v, want %v", reg, tt.expectedReg)
			}
		})
	}
}

type MockCommandExecutor struct {
	MockOutput []byte
	MockError  error
}

func (m *MockCommandExecutor) Execute(command string) ([]byte, error) {
	return m.MockOutput, m.MockError
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
			tagit := TagIt{Script: tt.script, commandExecutor: mockExecutor}

			output, err := tagit.runScript()

			if (err != nil) != tt.wantErr {
				t.Errorf("runScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(output) != tt.wantOutput {
				t.Errorf("runScript() got = %v, want %v", string(output), tt.wantOutput)
			}
		})
	}
}

// MockConsulClient implements the ConsulClient interface for testing.
type MockConsulClient struct {
	MockAgent *MockAgent
}

func (m *MockConsulClient) Agent() ConsulAgent {
	return m.MockAgent
}

// MockAgent simulates the Agent part of the Consul client.
type MockAgent struct {
	ServicesFunc        func() (map[string]*api.AgentService, error)
	ServiceRegisterFunc func(reg *api.AgentServiceRegistration) error
}

func (m *MockAgent) Services() (map[string]*api.AgentService, error) {
	return m.ServicesFunc()
}

func (m *MockAgent) ServiceRegister(reg *api.AgentServiceRegistration) error {
	return m.ServiceRegisterFunc(reg)
}

func TestNew(t *testing.T) {
	// Create mock dependencies
	mockConsulClient := &MockConsulClient{}
	mockCommandExecutor := &MockCommandExecutor{}

	// Call New with mock dependencies
	tagit := New(mockConsulClient, mockCommandExecutor, "test-service", "echo test", 30*time.Second, "test-prefix")

	// Validate the returned TagIt instance
	if tagit == nil {
		t.Fatalf("New() returned nil")
	}
	if tagit.client == nil {
		t.Errorf("TagIt client is nil")
	}
	if tagit.commandExecutor == nil {
		t.Errorf("TagIt commandExecutor is nil")
	}
	if tagit.ServiceID != "test-service" {
		t.Errorf("Expected ServiceID to be 'test-service', got '%s'", tagit.ServiceID)
	}
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
					ServicesFunc: func() (map[string]*api.AgentService, error) {
						return tt.mockServicesData, tt.mockServicesErr
					},
				},
			}
			tagit := New(mockConsulClient, nil, tt.serviceID, "", 0, "")

			service, err := tagit.getService()

			if tt.expectErr && err == nil {
				t.Errorf("Expected an error but got none")
			} else if !tt.expectErr && err != nil {
				t.Errorf("Did not expect an error but got: %v", err)
			}

			if !reflect.DeepEqual(service, tt.expectService) {
				t.Errorf("Expected service: %v, got: %v", tt.expectService, service)
			}
		})
	}
}

func TestGenerateNewTags(t *testing.T) {
	tests := []struct {
		name       string
		script     string
		mockOutput string
		mockError  error
		want       []string
		wantErr    bool
	}{
		{
			name:       "Valid Script Output",
			script:     "echo tag1 tag2",
			mockOutput: "tag1 tag2",
			want:       []string{"tag-tag1", "tag-tag2"},
			wantErr:    false,
		},
		{
			name:      "Script Execution Error",
			script:    "someinvalidcommand",
			mockError: fmt.Errorf("command failed"),
			want:      nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockCommandExecutor{
				MockOutput: []byte(tt.mockOutput),
				MockError:  tt.mockError,
			}
			tagit := TagIt{Script: tt.script, commandExecutor: mockExecutor, TagPrefix: "tag"}

			got, err := tagit.generateNewTags()
			if (err != nil) != tt.wantErr {
				t.Fatalf("generateNewTags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateNewTags() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateConsulService(t *testing.T) {
	tests := []struct {
		name            string
		existingTags    []string
		newTags         []string
		mockRegisterErr error
		expectUpdate    bool
		expectErr       bool
	}{
		{
			name:         "Update Needed",
			existingTags: []string{"tag1", "tag2"},
			newTags:      []string{"tag1", "tag3"},
			expectUpdate: true,
			expectErr:    false,
		},
		{
			name:         "No Update Needed",
			existingTags: []string{"tag1", "tag2"},
			newTags:      []string{"tag1", "tag2"},
			expectUpdate: false,
			expectErr:    false,
		},
		{
			name:            "Consul Register Error",
			existingTags:    []string{"tag1", "tag2"},
			newTags:         []string{"tag1", "tag3"},
			mockRegisterErr: fmt.Errorf("consul error"),
			expectUpdate:    true,
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &api.AgentService{Tags: tt.existingTags}
			mockConsulClient := &MockConsulClient{
				MockAgent: &MockAgent{
					ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
						return tt.mockRegisterErr
					},
				},
			}
			tagit := TagIt{client: mockConsulClient}

			err := tagit.updateConsulService(service, tt.newTags)
			if (err != nil) != tt.expectErr {
				t.Fatalf("updateConsulService() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestNewConsulAPIWrapper(t *testing.T) {
	// Mock Consul API client
	consulClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create Consul client: %v", err)
	}

	// Test NewConsulAPIWrapper
	wrapper := NewConsulAPIWrapper(consulClient)

	// Assert that wrapper is not nil
	assert.NotNil(t, wrapper, "NewConsulAPIWrapper returned nil")

	// Assert that wrapper implements ConsulClient interface
	_, isConsulClient := interface{}(wrapper).(ConsulClient)
	assert.True(t, isConsulClient, "NewConsulAPIWrapper does not implement ConsulClient interface")

	// Optionally, assert that wrapper's Agent method returns a ConsulAgent
	_, isConsulAgent := wrapper.Agent().(ConsulAgent)
	assert.True(t, isConsulAgent, "Wrapper's Agent method does not return a ConsulAgent")
}

func TestCmdExecutor_Execute(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantOutput  string
		expectError bool
	}{
		{
			name:        "Echo Command",
			command:     "echo test",
			wantOutput:  "test\n",
			expectError: false,
		},
		{
			name:        "Invalid Command",
			command:     "invalidcommand",
			expectError: true,
		},
	}

	executor := &CmdExecutor{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(tt.command)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error for command: %s, but got none", tt.command)
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error for command: %s, but got: %v", tt.command, err)
				}
				if strings.TrimSpace(string(output)) != strings.TrimSpace(tt.wantOutput) {
					t.Errorf("Unexpected output for command: %s. Expected: %s, got: %s", tt.command, tt.wantOutput, string(output))
				}
			}
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
				MockOutput: []byte(tt.mockScriptOutput),
				MockError:  tt.mockScriptError,
			}
			mockConsulClient := &MockConsulClient{
				MockAgent: &MockAgent{
					ServicesFunc: func() (map[string]*api.AgentService, error) {
						return map[string]*api.AgentService{
							"test-service": {
								ID:   "test-service",
								Tags: tt.existingTags,
							},
						}, nil
					},
					ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
						return tt.mockRegisterErr
					},
				},
			}
			tagit := New(mockConsulClient, mockExecutor, "test-service", "echo test", 30*time.Second, "tag")

			err := tagit.updateServiceTags()
			if (err != nil) != tt.expectError {
				t.Errorf("updateServiceTags() error = %v, wantErr %v", err, tt.expectError)
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
			expectTags:  []string{"prefix1", "prefix2", "other-tag"},
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
					ServicesFunc: func() (map[string]*api.AgentService, error) {
						return tt.mockServices, nil
					},
					ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
						// Ensure the service exists in the mock data
						if service, exists := tt.mockServices[reg.ID]; exists && tt.mockRegisterErr == nil {
							// Update the tags of the service
							service.Tags = reg.Tags
							tt.mockServices[reg.ID] = service // Update the map with the modified service
						}
						return tt.mockRegisterErr
					},
				},
			}
			tagit := New(mockConsulClient, nil, tt.serviceID, "", 0, tt.tagPrefix)

			err := tagit.CleanupTags()
			if (err != nil) != tt.expectError {
				t.Errorf("CleanupTags() error = %v, wantErr %v", err, tt.expectError)
			}

			if !tt.expectError {
				updatedService := tt.mockServices[tt.serviceID]
				if updatedService != nil && !reflect.DeepEqual(updatedService.Tags, tt.expectTags) {
					t.Errorf("Expected tags after cleanup: %v, got: %v", tt.expectTags, updatedService.Tags)
				}
			}
		})
	}
}

func TestRun(t *testing.T) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updateServiceTagsCalled := atomic.Int32{}
	mockExecutor := &MockCommandExecutor{
		MockOutput: []byte("new-tag1 new-tag2"),
		MockError:  nil,
	}
	mockConsulClient := &MockConsulClient{
		MockAgent: &MockAgent{
			ServicesFunc: func() (map[string]*api.AgentService, error) {
				updateServiceTagsCalled.Add(1)
				if updateServiceTagsCalled.Load() == 2 {
					return nil, fmt.Errorf("enter error")
				}
				return map[string]*api.AgentService{
					"test-service": {
						ID:   "test-service",
						Tags: []string{"old-tag"},
					},
				}, nil
			},
			ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {

				return nil
			},
		},
	}

	tagit := New(mockConsulClient, mockExecutor, "test-service", "echo test", 100*time.Millisecond, "tag")

	// Start Run in a goroutine
	go tagit.Run(ctx)

	// Allow some time to pass and then cancel the context
	time.Sleep(350 * time.Millisecond) // Adjust this duration as needed
	cancel()

	// Allow some time for the goroutine to react to the context cancellation
	time.Sleep(50 * time.Millisecond)

	// Check if updateServiceTags was called as expected
	if updateServiceTagsCalled.Load() < 2 || updateServiceTagsCalled.Load() > 3 {
		t.Errorf("Expected updateServiceTags to be called 2 or 3 times, got %d", updateServiceTagsCalled)
	}
}
