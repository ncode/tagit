package consul

import (
	"fmt"
	"maps"

	"github.com/hashicorp/consul/api"
)

// ServiceRegistration owns lookup and tag writes for one Consul client.
type ServiceRegistration struct {
	agent Agent
}

// NewServiceRegistration returns a registration store backed by client.Agent().
func NewServiceRegistration(client Client) *ServiceRegistration {
	return &ServiceRegistration{agent: client.Agent()}
}

// Load returns the registered Consul service for serviceID.
func (sr *ServiceRegistration) Load(serviceID string) (*api.AgentService, error) {
	service, _, err := sr.agent.Service(serviceID, nil)
	if err != nil {
		return nil, fmt.Errorf("lookup service %s: %w", serviceID, err)
	}
	if service == nil {
		return nil, fmt.Errorf("service %s not found", serviceID)
	}
	return service, nil
}

// UpdateTags writes tags to Consul while preserving all non-tag registration fields.
func (sr *ServiceRegistration) UpdateTags(service *api.AgentService, tags []string, changed bool) error {
	if !changed {
		return nil
	}
	if service == nil {
		return fmt.Errorf("register service: nil service")
	}

	registration := copyServiceToRegistration(service)
	registration.Tags = tags
	if err := sr.agent.ServiceRegister(registration); err != nil {
		return fmt.Errorf("register service %s: %w", service.ID, err)
	}
	return nil
}

func copyServiceToRegistration(service *api.AgentService) *api.AgentServiceRegistration {
	return &api.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Service,
		Tags:    service.Tags,
		Port:    service.Port,
		Address: service.Address,
		Kind:    service.Kind,
		Meta:    maps.Clone(service.Meta),
		Weights: &api.AgentWeights{
			Passing: service.Weights.Passing,
			Warning: service.Weights.Warning,
		},
	}
}
