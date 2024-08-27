package tagit

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/google/shlex"
	"github.com/hashicorp/consul/api"
)

// TagIt is the main struct for the tagit flow.
type TagIt struct {
	ServiceID       string
	Script          string
	Interval        time.Duration
	TagPrefix       string
	client          ConsulClient
	commandExecutor CommandExecutor
	logger          *slog.Logger
}

// ConsulClient is an interface for the Consul client.
type ConsulClient interface {
	Agent() ConsulAgent
}

// ConsulAgent is an interface for the Consul agent.
type ConsulAgent interface {
	Service(string, *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error)
	ServiceRegister(*api.AgentServiceRegistration) error
}

// ConsulAPIWrapper wraps the Consul API client to conform to the ConsulClient interface.
type ConsulAPIWrapper struct {
	client *api.Client
}

// NewConsulAPIWrapper creates a new instance of ConsulAPIWrapper.
func NewConsulAPIWrapper(client *api.Client) *ConsulAPIWrapper {
	return &ConsulAPIWrapper{client: client}
}

// Agent returns an object that conforms to the ConsulAgent interface.
func (w *ConsulAPIWrapper) Agent() ConsulAgent {
	return w.client.Agent()
}

// CommandExecutor is an interface for running commands.
type CommandExecutor interface {
	Execute(command string) ([]byte, error)
}

type CmdExecutor struct{}

func (e *CmdExecutor) Execute(command string) ([]byte, error) {
	args, err := shlex.Split(command)
	if err != nil {
		return nil, fmt.Errorf("failed to split command: %w", err)
	}
	return exec.Command(args[0], args[1:]...).Output()
}

// New creates a new TagIt struct.
func New(consulClient ConsulClient, commandExecutor CommandExecutor, serviceID string, script string, interval time.Duration, tagPrefix string, logger *slog.Logger) *TagIt {
	return &TagIt{
		ServiceID:       serviceID,
		Script:          script,
		Interval:        interval,
		TagPrefix:       tagPrefix,
		client:          consulClient,
		commandExecutor: commandExecutor,
		logger:          logger,
	}
}

// Run will run the tagit flow and tag consul services based on the script output
func (t *TagIt) Run(ctx context.Context) {
	ticker := time.NewTicker(t.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := t.updateServiceTags(); err != nil {
				t.logger.Error("error updating service tags",
					"service", t.ServiceID,
					"error", err)
			}
		}
	}
}

// CleanupTags removes all tags with the given prefix from the service.
func (t *TagIt) CleanupTags() error {
	service, err := t.getService()
	if err != nil {
		return fmt.Errorf("error getting service: %w", err)
	}

	// Filter out tags with the specified prefix
	cleanedTags := make([]string, 0)
	for _, tag := range service.Tags {
		if !strings.HasPrefix(tag, t.TagPrefix+"-") {
			cleanedTags = append(cleanedTags, tag)
		}
	}

	// Update the service with the cleaned tags
	if err := t.updateConsulService(service, cleanedTags); err != nil {
		return fmt.Errorf("error cleaning up tags: %w", err)
	}

	return nil
}

// runScript runs a command and returns the output.
func (t *TagIt) runScript() ([]byte, error) {
	t.logger.Info("running command",
		"service", t.ServiceID,
		"command", t.Script)
	return t.commandExecutor.Execute(t.Script)
}

// updateServiceTags updates the service tags.
func (t *TagIt) updateServiceTags() error {
	service, err := t.getService()
	if err != nil {
		return fmt.Errorf("error getting service: %w", err)
	}

	newTags, err := t.generateNewTags()
	if err != nil {
		return fmt.Errorf("error generating new tags: %w", err)
	}

	if err := t.updateConsulService(service, newTags); err != nil {
		return fmt.Errorf("error updating service in Consul: %w", err)
	}

	return nil
}

// generateNewTags runs the script and generates new tags.
func (t *TagIt) generateNewTags() ([]string, error) {
	out, err := t.runScript()
	if err != nil {
		return nil, fmt.Errorf("error running script: %w", err)
	}
	return t.parseScriptOutput(out), nil
}

// updateConsulService updates the service in Consul with the new tags.
func (t *TagIt) updateConsulService(service *api.AgentService, newTags []string) error {
	registration := t.copyServiceToRegistration(service)
	updatedTags, shouldTag := t.needsTag(registration.Tags, newTags)
	if shouldTag {
		registration.Tags = updatedTags
		if err := t.client.Agent().ServiceRegister(registration); err != nil {
			return fmt.Errorf("error registering service: %w", err)
		}
		t.logger.Info("updated service tags",
			"service", t.ServiceID,
			"tags", updatedTags)
	}
	return nil
}

// parseScriptOutput parses the script output and generates tags.
func (t *TagIt) parseScriptOutput(output []byte) []string {
	var tags []string
	for _, tag := range strings.Fields(string(output)) {
		tags = append(tags, fmt.Sprintf("%s-%s", t.TagPrefix, tag))
	}
	return tags
}

// copyServiceToRegistration copies *api.AgentService to *api.AgentServiceRegistration
func (t *TagIt) copyServiceToRegistration(service *api.AgentService) *api.AgentServiceRegistration {
	registration := &api.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Service,
		Tags:    service.Tags,
		Port:    service.Port,
		Address: service.Address,
		Kind:    service.Kind,
		Meta:    service.Meta,
		Weights: &api.AgentWeights{
			Passing: service.Weights.Passing,
			Warning: service.Weights.Warning,
		},
	}
	return registration
}

// getService returns the registered service.
// getService returns the registered service.
func (t *TagIt) getService() (*api.AgentService, error) {
	agent := t.client.Agent()
	service, _, err := agent.Service(t.ServiceID, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting service %s: %w", t.ServiceID, err)
	}
	if service == nil {
		return nil, fmt.Errorf("service %s not found", t.ServiceID)
	}
	return service, nil
}

// needsTag checks if the service needs to be tagged. Based on the diff of the current and updated tags, filtering out tags that are already tagged.
// but we never override the original tags from the consul service registration
func (t *TagIt) needsTag(current []string, update []string) (updatedTags []string, shouldTag bool) {
	diff := t.diffTags(current, update)
	if len(diff) == 0 {
		return nil, false
	}
	currentFiltered, _ := t.excludeTagged(current)
	updatedTags = append(currentFiltered, update...)
	slices.Sort(updatedTags)
	updatedTags = slices.Compact(updatedTags)
	return updatedTags, true
}

// excludeTagged filters out tags that are already tagged with the prefix.
func (t *TagIt) excludeTagged(tags []string) (filteredTags []string, tagged bool) {
	filteredTags = make([]string, 0) // Initialize with empty slice instead of nil
	for _, tag := range tags {
		if strings.HasPrefix(tag, t.TagPrefix+"-") {
			tagged = true
		} else {
			filteredTags = append(filteredTags, tag)
		}
	}
	return filteredTags, tagged
}

// diffTags compares two slices of strings and returns the difference.
func (t *TagIt) diffTags(current, update []string) []string {
	diff := make([]string, 0)
	currentSet := make(map[string]bool)
	updateSet := make(map[string]bool)

	// Create sets for both current and update
	for _, tag := range current {
		currentSet[tag] = true
	}
	for _, tag := range update {
		updateSet[tag] = true
	}

	// Find tags in update that are not in current
	for tag := range updateSet {
		if !currentSet[tag] {
			diff = append(diff, tag)
		}
	}

	// Find tags in current that are not in update
	for tag := range currentSet {
		if !updateSet[tag] {
			diff = append(diff, tag)
		}
	}

	return diff
}
