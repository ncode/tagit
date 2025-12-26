//go:build integration

package tagit

import (
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/ncode/tagit/pkg/consul"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getConsulAddr() string {
	addr := os.Getenv("CONSUL_ADDR")
	if addr == "" {
		return "127.0.0.1:8500"
	}
	return addr
}

func setupTestService(t *testing.T, client *api.Client, serviceID string, initialTags []string) {
	t.Helper()
	reg := &api.AgentServiceRegistration{
		ID:   serviceID,
		Name: serviceID,
		Port: 8080,
		Tags: initialTags,
	}
	err := client.Agent().ServiceRegister(reg)
	require.NoError(t, err, "failed to register test service")

	t.Cleanup(func() {
		_ = client.Agent().ServiceDeregister(serviceID)
	})
}

func getServiceTags(t *testing.T, client *api.Client, serviceID string) []string {
	t.Helper()
	svc, _, err := client.Agent().Service(serviceID, nil)
	require.NoError(t, err, "failed to get service")
	require.NotNil(t, svc, "service not found")
	return svc.Tags
}

func TestIntegration_TagItRun(t *testing.T) {
	consulClient, err := api.NewClient(&api.Config{
		Address: getConsulAddr(),
	})
	require.NoError(t, err, "failed to create consul client")

	// Verify Consul is reachable
	_, err = consulClient.Agent().Self()
	require.NoError(t, err, "Consul not reachable at %s", getConsulAddr())

	serviceID := "integration-test-service"
	initialTags := []string{"existing-tag", "another-tag"}

	setupTestService(t, consulClient, serviceID, initialTags)

	// Create a mock executor that returns predictable output
	mockExecutor := &MockCommandExecutor{
		MockOutput: []byte("tag1 tag2 tag3"),
		MockError:  nil,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	wrappedClient, err := consul.CreateClient(getConsulAddr(), "")
	require.NoError(t, err)

	tagit := New(
		wrappedClient,
		mockExecutor,
		serviceID,
		"echo tag1 tag2 tag3", // not actually used since we mock executor
		1*time.Second,
		"test",
		logger,
	)

	// Run one update cycle
	err = tagit.updateServiceTags()
	require.NoError(t, err)

	// Verify tags were applied
	tags := getServiceTags(t, consulClient, serviceID)
	slices.Sort(tags)

	expected := []string{"another-tag", "existing-tag", "test-tag1", "test-tag2", "test-tag3"}
	slices.Sort(expected)

	assert.Equal(t, expected, tags, "tags should include original tags plus new prefixed tags")
}

func TestIntegration_TagItCleanup(t *testing.T) {
	consulClient, err := api.NewClient(&api.Config{
		Address: getConsulAddr(),
	})
	require.NoError(t, err, "failed to create consul client")

	// Verify Consul is reachable
	_, err = consulClient.Agent().Self()
	require.NoError(t, err, "Consul not reachable at %s", getConsulAddr())

	serviceID := "integration-cleanup-service"
	initialTags := []string{"existing-tag", "test-tag1", "test-tag2", "other-tag"}

	setupTestService(t, consulClient, serviceID, initialTags)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	wrappedClient, err := consul.CreateClient(getConsulAddr(), "")
	require.NoError(t, err)

	tagit := New(
		wrappedClient,
		&CmdExecutor{},
		serviceID,
		"",
		0,
		"test",
		logger,
	)

	// Run cleanup
	err = tagit.CleanupTags()
	require.NoError(t, err)

	// Verify prefixed tags were removed
	tags := getServiceTags(t, consulClient, serviceID)
	slices.Sort(tags)

	expected := []string{"existing-tag", "other-tag"}
	slices.Sort(expected)

	assert.Equal(t, expected, tags, "only non-prefixed tags should remain")
}

func TestIntegration_TagItRunLoop(t *testing.T) {
	consulClient, err := api.NewClient(&api.Config{
		Address: getConsulAddr(),
	})
	require.NoError(t, err, "failed to create consul client")

	// Verify Consul is reachable
	_, err = consulClient.Agent().Self()
	require.NoError(t, err, "Consul not reachable at %s", getConsulAddr())

	serviceID := "integration-loop-service"
	initialTags := []string{"existing-tag"}

	setupTestService(t, consulClient, serviceID, initialTags)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	wrappedClient, err := consul.CreateClient(getConsulAddr(), "")
	require.NoError(t, err)

	// Create executor that changes output over time
	callCount := 0
	dynamicExecutor := &DynamicMockExecutor{
		ExecuteFunc: func(command string) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return []byte("v1"), nil
			}
			return []byte("v2"), nil
		},
	}

	tagit := New(
		wrappedClient,
		dynamicExecutor,
		serviceID,
		"echo dynamic",
		500*time.Millisecond,
		"dyn",
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	// Run in background
	done := make(chan struct{})
	go func() {
		tagit.Run(ctx)
		close(done)
	}()

	// Wait for completion
	<-done

	// Verify final state has updated tags
	tags := getServiceTags(t, consulClient, serviceID)

	assert.Contains(t, tags, "existing-tag", "original tag should be preserved")
	assert.Contains(t, tags, "dyn-v2", "final dynamic tag should be present")
}

func TestIntegration_RealScriptExecution(t *testing.T) {
	consulClient, err := api.NewClient(&api.Config{
		Address: getConsulAddr(),
	})
	require.NoError(t, err, "failed to create consul client")

	// Verify Consul is reachable
	_, err = consulClient.Agent().Self()
	require.NoError(t, err, "Consul not reachable at %s", getConsulAddr())

	serviceID := "integration-script-service"
	initialTags := []string{"existing"}

	setupTestService(t, consulClient, serviceID, initialTags)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	wrappedClient, err := consul.CreateClient(getConsulAddr(), "")
	require.NoError(t, err)

	// Use real executor with actual shell command
	tagit := New(
		wrappedClient,
		&CmdExecutor{Timeout: 5 * time.Second},
		serviceID,
		"echo alpha beta gamma",
		1*time.Second,
		"real",
		logger,
	)

	// Run one update cycle
	err = tagit.updateServiceTags()
	require.NoError(t, err)

	// Verify tags were applied
	tags := getServiceTags(t, consulClient, serviceID)
	slices.Sort(tags)

	expected := []string{"existing", "real-alpha", "real-beta", "real-gamma"}
	slices.Sort(expected)

	assert.Equal(t, expected, tags, "tags should include original plus script output tags")
}

// DynamicMockExecutor allows changing behavior between calls
type DynamicMockExecutor struct {
	ExecuteFunc func(command string) ([]byte, error)
}

func (d *DynamicMockExecutor) Execute(command string) ([]byte, error) {
	return d.ExecuteFunc(command)
}

func TestIntegration_ServiceNotFound(t *testing.T) {
	wrappedClient, err := consul.CreateClient(getConsulAddr(), "")
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tagit := New(
		wrappedClient,
		&MockCommandExecutor{MockOutput: []byte("tag1")},
		"non-existent-service-12345",
		"echo tag1",
		1*time.Second,
		"test",
		logger,
	)

	// Should return error for non-existent service
	err = tagit.updateServiceTags()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent-service-12345")
}

func TestIntegration_EmptyScriptOutput(t *testing.T) {
	consulClient, err := api.NewClient(&api.Config{
		Address: getConsulAddr(),
	})
	require.NoError(t, err)

	_, err = consulClient.Agent().Self()
	require.NoError(t, err, "Consul not reachable")

	serviceID := "integration-empty-output-service"
	// Start with some prefixed tags that should be removed
	initialTags := []string{"keep-me", "test-remove1", "test-remove2", "also-keep"}

	setupTestService(t, consulClient, serviceID, initialTags)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	wrappedClient, err := consul.CreateClient(getConsulAddr(), "")
	require.NoError(t, err)

	// Empty output should remove all prefixed tags
	tagit := New(
		wrappedClient,
		&MockCommandExecutor{MockOutput: []byte("")},
		serviceID,
		"echo",
		1*time.Second,
		"test",
		logger,
	)

	err = tagit.updateServiceTags()
	require.NoError(t, err)

	tags := getServiceTags(t, consulClient, serviceID)
	slices.Sort(tags)

	// Only non-prefixed tags should remain
	expected := []string{"also-keep", "keep-me"}
	assert.Equal(t, expected, tags, "prefixed tags should be removed when script outputs nothing")
}

func TestIntegration_Idempotency(t *testing.T) {
	consulClient, err := api.NewClient(&api.Config{
		Address: getConsulAddr(),
	})
	require.NoError(t, err)

	_, err = consulClient.Agent().Self()
	require.NoError(t, err, "Consul not reachable")

	serviceID := "integration-idempotent-service"
	initialTags := []string{"existing"}

	setupTestService(t, consulClient, serviceID, initialTags)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	wrappedClient, err := consul.CreateClient(getConsulAddr(), "")
	require.NoError(t, err)

	mockExecutor := &MockCommandExecutor{
		MockOutput: []byte("stable-tag"),
	}

	tagit := New(
		wrappedClient,
		mockExecutor,
		serviceID,
		"echo stable-tag",
		1*time.Second,
		"idem",
		logger,
	)

	// First update should register
	err = tagit.updateServiceTags()
	require.NoError(t, err)

	// Get the current modify index to detect changes
	svc1, _, err := consulClient.Agent().Service(serviceID, nil)
	require.NoError(t, err)
	modifyIndex1 := svc1.ModifyIndex

	// Second update with same output should NOT register
	err = tagit.updateServiceTags()
	require.NoError(t, err)

	svc2, _, err := consulClient.Agent().Service(serviceID, nil)
	require.NoError(t, err)
	modifyIndex2 := svc2.ModifyIndex

	// Third update with same output should NOT register
	err = tagit.updateServiceTags()
	require.NoError(t, err)

	svc3, _, err := consulClient.Agent().Service(serviceID, nil)
	require.NoError(t, err)
	modifyIndex3 := svc3.ModifyIndex

	// ModifyIndex should not change after first update since tags are identical
	assert.Equal(t, modifyIndex1, modifyIndex2, "second update should not modify service")
	assert.Equal(t, modifyIndex2, modifyIndex3, "third update should not modify service")

	// Verify correct tags are present
	tags := getServiceTags(t, consulClient, serviceID)
	slices.Sort(tags)
	expected := []string{"existing", "idem-stable-tag"}
	assert.Equal(t, expected, tags)
}

func TestIntegration_ServiceMetadataPreservation(t *testing.T) {
	consulClient, err := api.NewClient(&api.Config{
		Address: getConsulAddr(),
	})
	require.NoError(t, err)

	_, err = consulClient.Agent().Self()
	require.NoError(t, err, "Consul not reachable")

	serviceID := "integration-metadata-service"

	// Register service with metadata, weights, and other fields
	reg := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    "metadata-test",
		Port:    9090,
		Address: "10.0.0.1",
		Tags:    []string{"original-tag"},
		Meta: map[string]string{
			"version":     "1.2.3",
			"environment": "test",
			"owner":       "integration-test",
		},
		Weights: &api.AgentWeights{
			Passing: 10,
			Warning: 5,
		},
	}
	err = consulClient.Agent().ServiceRegister(reg)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = consulClient.Agent().ServiceDeregister(serviceID)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	wrappedClient, err := consul.CreateClient(getConsulAddr(), "")
	require.NoError(t, err)

	tagit := New(
		wrappedClient,
		&MockCommandExecutor{MockOutput: []byte("new-tag")},
		serviceID,
		"echo new-tag",
		1*time.Second,
		"meta",
		logger,
	)

	// Update tags
	err = tagit.updateServiceTags()
	require.NoError(t, err)

	// Verify all original properties are preserved
	svc, _, err := consulClient.Agent().Service(serviceID, nil)
	require.NoError(t, err)

	assert.Equal(t, "metadata-test", svc.Service, "service name should be preserved")
	assert.Equal(t, 9090, svc.Port, "port should be preserved")
	assert.Equal(t, "10.0.0.1", svc.Address, "address should be preserved")

	// Verify metadata is preserved
	assert.Equal(t, "1.2.3", svc.Meta["version"], "metadata version should be preserved")
	assert.Equal(t, "test", svc.Meta["environment"], "metadata environment should be preserved")
	assert.Equal(t, "integration-test", svc.Meta["owner"], "metadata owner should be preserved")

	// Verify weights are preserved
	assert.Equal(t, 10, svc.Weights.Passing, "passing weight should be preserved")
	assert.Equal(t, 5, svc.Weights.Warning, "warning weight should be preserved")

	// Verify tags were updated correctly
	tags := svc.Tags
	slices.Sort(tags)
	expected := []string{"meta-new-tag", "original-tag"}
	assert.Equal(t, expected, tags, "tags should include original plus new prefixed tag")
}
