package consul

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
)

func TestServiceRegistration_Load(t *testing.T) {
	t.Run("service is found", func(t *testing.T) {
		store := NewServiceRegistration(&registrationClient{
			agent: &registrationAgent{
				serviceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
					return &api.AgentService{ID: serviceID, Service: "api"}, nil, nil
				},
			},
		})

		got, err := store.Load("api-1")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got.ID != "api-1" {
			t.Fatalf("Load() ID = %q, want api-1", got.ID)
		}
	})

	t.Run("service is missing", func(t *testing.T) {
		store := NewServiceRegistration(&registrationClient{
			agent: &registrationAgent{
				serviceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
					return nil, nil, nil
				},
			},
		})

		_, err := store.Load("missing")
		if err == nil {
			t.Fatal("Load() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "missing") {
			t.Fatalf("Load() error = %q, want service ID", err)
		}
	})

	t.Run("lookup error has operation context", func(t *testing.T) {
		store := NewServiceRegistration(&registrationClient{
			agent: &registrationAgent{
				serviceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
					return nil, nil, fmt.Errorf("consul unavailable")
				},
			},
		})

		_, err := store.Load("api-1")
		if err == nil {
			t.Fatal("Load() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "lookup service api-1") {
			t.Fatalf("Load() error = %q, want lookup context", err)
		}
	})
}

func TestServiceRegistration_UpdateTags(t *testing.T) {
	service := &api.AgentService{
		ID:      "api-1",
		Service: "api",
		Tags:    []string{"old"},
		Address: "10.0.0.2",
		Port:    8080,
		Kind:    api.ServiceKindTypical,
		Meta:    map[string]string{"version": "1"},
		Weights: api.AgentWeights{
			Passing: 10,
			Warning: 1,
		},
	}

	var got *api.AgentServiceRegistration
	store := NewServiceRegistration(&registrationClient{
		agent: &registrationAgent{
			registerFunc: func(reg *api.AgentServiceRegistration) error {
				got = reg
				return nil
			},
		},
	})

	if err := store.UpdateTags(service, []string{"new", "static"}, true); err != nil {
		t.Fatalf("UpdateTags() error = %v", err)
	}

	if got == nil {
		t.Fatal("ServiceRegister was not called")
	}
	if got.ID != service.ID || got.Name != service.Service || got.Address != service.Address || got.Port != service.Port || got.Kind != service.Kind {
		t.Fatalf("registration fields were not preserved: %#v", got)
	}
	if got.Meta["version"] != "1" {
		t.Fatalf("Meta = %#v, want version preserved", got.Meta)
	}
	if got.Weights == nil || got.Weights.Passing != 10 || got.Weights.Warning != 1 {
		t.Fatalf("Weights = %#v, want preserved weights", got.Weights)
	}
	if want := []string{"new", "static"}; !equalStrings(got.Tags, want) {
		t.Fatalf("Tags = %v, want %v", got.Tags, want)
	}
}

func TestServiceRegistration_UpdateTagsSkipsUnchanged(t *testing.T) {
	registerCalls := 0
	store := NewServiceRegistration(&registrationClient{
		agent: &registrationAgent{
			registerFunc: func(reg *api.AgentServiceRegistration) error {
				registerCalls++
				return nil
			},
		},
	})

	err := store.UpdateTags(&api.AgentService{ID: "api-1"}, []string{"static"}, false)
	if err != nil {
		t.Fatalf("UpdateTags() error = %v", err)
	}
	if registerCalls != 0 {
		t.Fatalf("ServiceRegister calls = %d, want 0", registerCalls)
	}
}

func TestServiceRegistration_UpdateTagsWriteError(t *testing.T) {
	store := NewServiceRegistration(&registrationClient{
		agent: &registrationAgent{
			registerFunc: func(reg *api.AgentServiceRegistration) error {
				return fmt.Errorf("permission denied")
			},
		},
	})

	err := store.UpdateTags(&api.AgentService{ID: "api-1"}, []string{"new"}, true)
	if err == nil {
		t.Fatal("UpdateTags() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "register service api-1") {
		t.Fatalf("UpdateTags() error = %q, want register context", err)
	}
}

type registrationClient struct {
	agent Agent
}

func (rc *registrationClient) Agent() Agent {
	return rc.agent
}

type registrationAgent struct {
	serviceFunc  func(string, *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error)
	registerFunc func(*api.AgentServiceRegistration) error
}

func (ra *registrationAgent) Service(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
	if ra.serviceFunc == nil {
		return nil, nil, nil
	}
	return ra.serviceFunc(serviceID, q)
}

func (ra *registrationAgent) ServiceRegister(reg *api.AgentServiceRegistration) error {
	if ra.registerFunc == nil {
		return nil
	}
	return ra.registerFunc(reg)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
