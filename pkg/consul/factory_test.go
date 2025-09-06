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

func (m *MockConsulClient) Agent() ConsulAgent {
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