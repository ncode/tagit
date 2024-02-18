package tagit

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/shlex"
	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

// TagIt is the main struct for the tagit flow.
type TagIt struct {
	ConsulAddr      string
	ServiceID       string
	Script          string
	Interval        time.Duration
	Token           string
	TagPrefix       string
	client          ConsulClient
	commandExecutor CommandExecutor
}

// ConsulClient is an interface for the Consul client.
type ConsulClient interface {
	Agent() ConsulAgent
}

// ConsulClientAgent is an interface for the Consul agent.
type ConsulAgent interface {
	Services() (map[string]*api.AgentService, error)
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
		return nil, err
	}
	return exec.Command(args[0], args[1:]...).Output()
}

// New creates a new TagIt struct.
func New(consulClient ConsulClient, commandExecutor CommandExecutor, serviceID string, script string, interval time.Duration, tagPrefix string) *TagIt {
	return &TagIt{
		ServiceID:       serviceID,
		Script:          script,
		Interval:        interval,
		TagPrefix:       tagPrefix,
		client:          consulClient,
		commandExecutor: commandExecutor,
	}
}

// Run will run the tagit flow and tag consul services based on the script output
func (t *TagIt) Run() {
	for {
		err := t.updateServiceTags()
		if err != nil {
			log.WithFields(log.Fields{
				"service": t.ServiceID,
				"error":   err,
			}).Error("error updating service tags")
		}
		time.Sleep(t.Interval)
	}
}

// runScript runs a command and returns the output.
func (t *TagIt) runScript() ([]byte, error) {
	log.WithFields(log.Fields{
		"service": t.ServiceID,
		"command": t.Script,
	}).Info("running command")
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
		return nil, err
	}
	return t.parseScriptOutput(out), nil
}

// updateConsulService updates the service in Consul with the new tags.
func (t *TagIt) updateConsulService(service *api.AgentService, newTags []string) error {
	registration := t.copyServiceToRegistration(service)
	updatedTags, shouldTag := t.needsTag(registration.Tags, newTags)
	if shouldTag {
		registration.Tags = updatedTags
		return t.client.Agent().ServiceRegister(registration)
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

// CleanupServiceTags cleans up the service tags added by tagit.
func (t *TagIt) CleanupServiceTags() error {
	service, err := t.getService()
	if err != nil {
		return err
	}
	registration := t.copyServiceToRegistration(service)
	log.WithFields(log.Fields{
		"service": t.ServiceID,
		"tags":    registration.Tags,
	}).Info("current service tags")

	filteredTags, tagged := t.excludeTagged(registration.Tags)
	if tagged {
		log.WithFields(log.Fields{
			"service": t.ServiceID,
			"tags":    filteredTags,
		}).Info("updating service tags")
		registration.Tags = filteredTags
		err = t.client.Agent().ServiceRegister(registration)
		if err != nil {
			return err
		}
	}

	return err
}

// Copy *api.AgentService to *api.AgentServiceRegistration
func (t *TagIt) copyServiceToRegistration(service *api.AgentService) *api.AgentServiceRegistration {
	return &api.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Service,
		Tags:    service.Tags,
		Port:    service.Port,
		Address: service.Address,
		Kind:    service.Kind,
		Weights: &service.Weights,
		Meta:    service.Meta,
	}
}

// getService returns the registered service.
func (t *TagIt) getService() (service *api.AgentService, err error) {
	agent := t.client.Agent()
	services, err := agent.Services()
	if err != nil {
		return service, err
	}

	service, ok := services[t.ServiceID]
	if !ok {
		return service, fmt.Errorf("service %s not found", t.ServiceID)
	}

	return service, err
}

// needsTag checks if the service needs to be tagged. Based of the diff of the current and updated tags, filtering out tags that are already tagged.
// but we never override the original tags from the consul service registration
func (t *TagIt) needsTag(current []string, update []string) (updatedTags []string, shouldTag bool) {
	diff := t.diffTags(current, update)
	if len(diff) == 0 {
		return nil, false
	}

	updatedTags, _ = t.excludeTagged(diff)
	return updatedTags, true
}

// excludeTagged filters out tags that are already tagged with the prefix.
func (t *TagIt) excludeTagged(tags []string) (filteredTags []string, tagged bool) {
	for _, tag := range tags {
		// Using HasPrefix for a more accurate prefix check
		if strings.HasPrefix(tag, t.TagPrefix+"-") {
			tagged = true
		} else {
			filteredTags = append(filteredTags, tag)
		}
	}
	return filteredTags, tagged
}

// diffTags compares two slices of strings and returns the difference.
func (t *TagIt) diffTags(current []string, update []string) []string {
	tagMap := make(map[string]bool)
	var diff []string

	// Map each tag in the update slice to true
	for _, tag := range update {
		tagMap[tag] = true
	}

	// Add tags from current that are not in update
	for _, tag := range current {
		if _, found := tagMap[tag]; !found {
			diff = append(diff, tag)
		}
	}

	// Reset the map for reuse
	for k := range tagMap {
		delete(tagMap, k)
	}

	// Map each tag in the current slice to true
	for _, tag := range current {
		tagMap[tag] = true
	}

	// Add tags from update that are not in current
	for _, tag := range update {
		if _, found := tagMap[tag]; !found {
			diff = append(diff, tag)
		}
	}

	return diff
}
