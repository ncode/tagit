/*
Copyright © 2025 Juliano Martinez <juliano@martinez.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package consul

import (
	"fmt"

	"github.com/hashicorp/consul/api"
)

// Client is an interface for the Consul client.
type Client interface {
	Agent() Agent
}

// Agent is an interface for the Consul agent.
type Agent interface {
	Service(string, *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error)
	ServiceRegister(*api.AgentServiceRegistration) error
}

// ApiWrapper wraps the Consul API client to conform to the Client interface.
type ApiWrapper struct {
	client *api.Client
}

// NewConsulAPIWrapper creates a new instance of ApiWrapper.
func NewConsulAPIWrapper(client *api.Client) *ApiWrapper {
	return &ApiWrapper{client: client}
}

// Agent returns an object that conforms to the Agent interface.
func (w *ApiWrapper) Agent() Agent {
	return w.client.Agent()
}

// ClientFactory is an interface for creating Consul clients
type ClientFactory interface {
	NewClient(address, token string) (Client, error)
}

// DefaultFactory creates real Consul clients
type DefaultFactory struct{}

// NewClient creates a new Consul client with the given configuration
func (f *DefaultFactory) NewClient(address, token string) (Client, error) {
	config := api.DefaultConfig()
	config.Address = address
	config.Token = token

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	return NewConsulAPIWrapper(client), nil
}

// MockFactory creates mock Consul clients for testing
type MockFactory struct {
	MockClient Client
	MockError  error
}

// NewClient returns the mock client or error
func (f *MockFactory) NewClient(address, token string) (Client, error) {
	if f.MockError != nil {
		return nil, f.MockError
	}
	return f.MockClient, nil
}

// Factory is the global factory instance (can be overridden for testing)
var Factory ClientFactory = &DefaultFactory{}

// SetFactory allows tests to inject a mock factory
func SetFactory(f ClientFactory) {
	Factory = f
}

// ResetFactory resets to the default factory
func ResetFactory() {
	Factory = &DefaultFactory{}
}

// CreateClient is a convenience function that uses the global factory
func CreateClient(address, token string) (Client, error) {
	return Factory.NewClient(address, token)
}
