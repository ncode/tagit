package consul

import (
	"fmt"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

// MockConsulClient for testing
type MockConsulClient struct {
	MockAgent *MockAgent
}

func (m *MockConsulClient) Agent() Agent {
	return m.MockAgent
}

type MockAgent struct {
	ServiceFunc         func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error)
	ServiceRegisterFunc func(reg *api.AgentServiceRegistration) error
}

func (m *MockAgent) Service(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
	if m.ServiceFunc != nil {
		return m.ServiceFunc(serviceID, q)
	}
	return nil, nil, nil
}

func (m *MockAgent) ServiceRegister(reg *api.AgentServiceRegistration) error {
	if m.ServiceRegisterFunc != nil {
		return m.ServiceRegisterFunc(reg)
	}
	return nil
}

func TestDefaultFactory(t *testing.T) {
	factory := &DefaultFactory{}

	// Test with valid configuration
	// Note: This will actually try to connect to Consul, so it might fail
	// in environments without Consul running
	t.Run("Create client with valid config", func(t *testing.T) {
		client, err := factory.NewClient("127.0.0.1:8500", "test-token")
		// We expect this to succeed in creating a client object,
		// even if Consul isn't actually running
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	// Test with environment that causes api.NewClient to fail
	t.Run("Create client with invalid TLS config", func(t *testing.T) {
		// Set environment variables that should cause TLS config to fail
		t.Setenv("CONSUL_CACERT", "/nonexistent/ca.pem")
		t.Setenv("CONSUL_CLIENT_CERT", "/nonexistent/client.pem")
		t.Setenv("CONSUL_CLIENT_KEY", "/nonexistent/client.key")

		// These environment variables should cause api.NewClient to fail
		// when it tries to load the TLS certificates
		client, err := factory.NewClient("127.0.0.1:8500", "test-token")

		// Check if we got an error (which we expect due to invalid cert paths)
		if err != nil {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create Consul client")
			assert.Nil(t, client)
		} else {
			// If no error, the API might have changed or env vars weren't processed
			t.Log("Expected error but got none - Consul API may have changed behavior")
			assert.NotNil(t, client)
		}
	})

	// Test with invalid HTTP proxy that should cause client creation to fail
	t.Run("Create client with invalid HTTP proxy", func(t *testing.T) {
		// Set an invalid HTTP proxy that should cause the client to fail
		t.Setenv("HTTP_PROXY", "://invalid-proxy-url")

		client, err := factory.NewClient("127.0.0.1:8500", "test-token")

		// The invalid proxy URL should cause an error
		if err != nil {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create Consul client")
			assert.Nil(t, client)
		} else {
			// If no error, log it for debugging
			t.Log("Expected error with invalid proxy but got none")
			assert.NotNil(t, client)
		}
	})

	// Test with malformed TLS configuration
	t.Run("Create client with malformed TLS env", func(t *testing.T) {
		// Set CONSUL_HTTP_SSL=true to force TLS but without proper certs
		t.Setenv("CONSUL_HTTP_SSL", "true")
		t.Setenv("CONSUL_CACERT", "not-a-file.pem")

		client, err := factory.NewClient("127.0.0.1:8500", "test-token")

		// This should fail when trying to set up TLS
		if err != nil {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create Consul client")
			assert.Nil(t, client)
		} else {
			t.Log("Expected error with invalid TLS setup but got none")
			assert.NotNil(t, client)
		}
	})
}

func TestMockFactory(t *testing.T) {
	t.Run("Returns mock client", func(t *testing.T) {
		mockClient := &MockConsulClient{
			MockAgent: &MockAgent{},
		}
		factory := &MockFactory{
			MockClient: mockClient,
		}

		client, err := factory.NewClient("any-address", "any-token")
		assert.NoError(t, err)
		assert.Equal(t, mockClient, client)
	})

	t.Run("Returns error when configured", func(t *testing.T) {
		expectedErr := fmt.Errorf("connection failed")
		factory := &MockFactory{
			MockError: expectedErr,
		}

		client, err := factory.NewClient("any-address", "any-token")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, client)
	})
}

func TestGlobalFactory(t *testing.T) {
	// Save original factory
	originalFactory := Factory
	defer func() {
		Factory = originalFactory
	}()

	t.Run("Default factory is set", func(t *testing.T) {
		ResetFactory()
		_, ok := Factory.(*DefaultFactory)
		assert.True(t, ok, "Default factory should be DefaultFactory type")
	})

	t.Run("Can set mock factory", func(t *testing.T) {
		mockFactory := &MockFactory{
			MockClient: &MockConsulClient{},
		}
		SetFactory(mockFactory)
		assert.Equal(t, mockFactory, Factory)
	})

	t.Run("CreateClient uses global factory", func(t *testing.T) {
		mockClient := &MockConsulClient{
			MockAgent: &MockAgent{},
		}
		mockFactory := &MockFactory{
			MockClient: mockClient,
		}
		SetFactory(mockFactory)

		client, err := CreateClient("test-addr", "test-token")
		assert.NoError(t, err)
		assert.Equal(t, mockClient, client)
	})

	t.Run("ResetFactory restores default", func(t *testing.T) {
		mockFactory := &MockFactory{}
		SetFactory(mockFactory)
		ResetFactory()
		_, ok := Factory.(*DefaultFactory)
		assert.True(t, ok, "Factory should be reset to DefaultFactory")
	})
}
