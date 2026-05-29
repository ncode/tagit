package tagit

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/google/shlex"
	"github.com/hashicorp/consul/api"
	"github.com/ncode/tagit/pkg/consul"
)

// TagIt is the main struct for the tagit flow.
type TagIt struct {
	ServiceID       string
	Script          string
	Interval        time.Duration
	TagPrefix       string
	client          consul.Client
	registration    *consul.ServiceRegistration
	commandExecutor CommandExecutor
	logger          *slog.Logger
}

// CommandExecutor is an interface for running commands.
type CommandExecutor interface {
	Execute(command string) ([]byte, error)
}

// DefaultScriptTimeout is the default timeout for script execution.
const DefaultScriptTimeout = 30 * time.Second

type CmdExecutor struct {
	Timeout time.Duration
}

func (e *CmdExecutor) Execute(command string) ([]byte, error) {
	if command == "" {
		return nil, fmt.Errorf("failed to execute: empty command")
	}
	args, err := shlex.Split(command)
	if err != nil {
		return nil, fmt.Errorf("failed to split command: %w", err)
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("failed to execute: no command after splitting")
	}

	timeout := e.Timeout
	if timeout == 0 {
		timeout = DefaultScriptTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, args[0], args[1:]...).Output()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("script execution timed out after %v", timeout)
	}
	return out, err
}

// New creates a new TagIt struct.
func New(consulClient consul.Client, commandExecutor CommandExecutor, serviceID string, script string, interval time.Duration, tagPrefix string, logger *slog.Logger) *TagIt {
	return &TagIt{
		ServiceID:       serviceID,
		Script:          script,
		Interval:        interval,
		TagPrefix:       tagPrefix,
		client:          consulClient,
		registration:    consul.NewServiceRegistration(consulClient),
		commandExecutor: commandExecutor,
		logger:          logger,
	}
}

// Run will run the tagit flow and tag consul services based on the script output
func (t *TagIt) Run(ctx context.Context) {
	Scheduler{
		Interval: t.Interval,
		RunOnce:  t.ReconcileOnce,
		Logger:   t.logger,
	}.Run(ctx)
}

// ReconcileOnce runs one deterministic service tag reconciliation pass.
func (t *TagIt) ReconcileOnce() error {
	return t.updateServiceTags()
}

// CleanupTags removes all tags with the given prefix from the service.
func (t *TagIt) CleanupTags() error {
	service, err := t.getService()
	if err != nil {
		return fmt.Errorf("error getting service: %w", err)
	}

	reconciliation := NewReconciler(t.TagPrefix).Cleanup(service.Tags)
	if err := t.registrationStore().UpdateTags(service, reconciliation.Tags, reconciliation.Changed); err != nil {
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

	out, err := t.runScript()
	if err != nil {
		return fmt.Errorf("error running script: %w", err)
	}

	reconciliation := NewReconciler(t.TagPrefix).Reconcile(service.Tags, out)
	if err := t.registrationStore().UpdateTags(service, reconciliation.Tags, reconciliation.Changed); err != nil {
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
	if err := t.registrationStore().UpdateTags(service, newTags, true); err != nil {
		return err
	}
	t.logger.Info("updated service tags",
		"service", t.ServiceID,
		"tags", newTags)
	return nil
}

// parseScriptOutput parses the script output and generates tags.
func (t *TagIt) parseScriptOutput(output []byte) []string {
	return NewReconciler(t.TagPrefix).Reconcile(nil, output).Tags
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
func (t *TagIt) getService() (*api.AgentService, error) {
	return t.registrationStore().Load(t.ServiceID)
}

func (t *TagIt) registrationStore() *consul.ServiceRegistration {
	if t.registration != nil {
		return t.registration
	}
	return consul.NewServiceRegistration(t.client)
}

// needsTag checks if the service needs to be tagged. Based on the diff of the current and updated tags, filtering out tags that are already tagged.
// but we never override the original tags from the consul service registration
func (t *TagIt) needsTag(current []string, update []string) (updatedTags []string, shouldTag bool) {
	reconciliation := NewReconciler(t.TagPrefix).ReconcileManaged(current, update)
	if !reconciliation.Changed {
		return nil, false
	}
	return reconciliation.Tags, true
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
