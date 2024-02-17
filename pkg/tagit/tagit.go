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
	ConsulAddr string
	ServiceID  string
	Script     string
	Interval   time.Duration
	Token      string
	TagPrefix  string
	client     *api.Client
}

// New creates a new TagIt struct.
func New(consulAddr string, serviceID string, script string, interval time.Duration, token string, tagPrefix string) (t *TagIt, err error) {
	t = &TagIt{
		ConsulAddr: consulAddr,
		ServiceID:  serviceID,
		Script:     script,
		Interval:   interval,
		Token:      token,
		TagPrefix:  tagPrefix,
	}
	config := api.DefaultConfig()
	config.Address = t.ConsulAddr
	config.Token = t.Token
	t.client, err = api.NewClient(config)
	if err != nil {
		return t, err
	}
	return t, err
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
	args, err := shlex.Split(t.Script)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Output()
}

// updateServiceTags updates the service tags.
func (t *TagIt) updateServiceTags() error {
	service, err := t.getService()
	if err != nil {
		return err
	}
	registration := t.copyServiceToRegistration(service)
	log.WithFields(log.Fields{
		"service": t.ServiceID,
		"tags":    registration.Tags,
	}).Debug("current service tags")
	out, err := t.runScript()
	if err != nil {
		return err
	}

	var tags []string
	for _, tag := range strings.Fields(string(out)) {
		tags = append(tags, fmt.Sprintf("%s-%s", t.TagPrefix, tag))
	}

	updatedTags, shouldTag := t.needsTag(registration.Tags, tags)
	if shouldTag {
		registration.Tags = updatedTags
		log.WithFields(log.Fields{
			"service": t.ServiceID,
			"tags":    registration.Tags,
		}).Info("updating service tags")
		err = t.client.Agent().ServiceRegister(registration)
		if err != nil {
			return err
		}
	} else {
		log.WithFields(log.Fields{
			"service": t.ServiceID,
			"tags":    registration.Tags,
		}).Debug("no changes to service tags")
	}

	return err
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
