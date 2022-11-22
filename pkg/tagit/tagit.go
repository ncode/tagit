package tagit

import (
	"fmt"
	"github.com/google/shlex"
	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
	"time"
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
				"caller": "Run",
				"error":  err,
			}).Error("error updating service tags")
		}
		time.Sleep(t.Interval)
	}
}

// runCommand runs a command and returns the output.
func runCommand(command string) ([]byte, error) {
	log.WithFields(log.Fields{
		"caller": "runCommand",
	}).Info("Running command: ", command)
	args, err := shlex.Split(command)
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
		"caller":  "updateServiceTags",
		"service": t.ServiceID,
		"tags":    registration.Tags,
	}).Debug("current service tags")
	out, err := runCommand(t.Script)
	if err != nil {
		return err
	}
	var tags []string
	for _, tag := range strings.Fields(string(out)) {
		tags = append(tags, fmt.Sprintf("%s-%s", t.TagPrefix, tag))
	}
	diff, shouldTag := t.needsTag(registration.Tags, tags)
	if shouldTag {
		registration.Tags = append(diff, tags...)
		log.WithFields(log.Fields{
			"caller":  "updateServiceTags",
			"service": t.ServiceID,
			"tags":    registration.Tags,
		}).Info("updating service tags")
		err = t.client.Agent().ServiceRegister(registration)
		if err != nil {
			return err
		}
	} else {
		log.WithFields(log.Fields{
			"caller":  "updateServiceTags",
			"service": t.ServiceID,
			"tags":    registration.Tags,
		}).Debug("no changes to service tags")
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

func (t *TagIt) needsTag(current []string, update []string) (filteredTags []string, shouldTag bool) {
	diff := t.compareTags(current, update)
	filteredTags, tagged := t.filterTagged(current)
	if !tagged {
		return filteredTags, true
	}

	if len(append(update, diff...)) != len(current) {
		return filteredTags, true
	}

	_, shouldTag = t.filterTagged(diff)

	return filteredTags, shouldTag
}

// filterTagged filters out tags that are already tagged with the prefix.
func (t *TagIt) filterTagged(tags []string) (filteredTags []string, tagged bool) {
	for _, tag := range tags {
		if !strings.Contains(tag, t.TagPrefix) {
			tagged = true
			filteredTags = append(filteredTags, tag)
		}
	}
	return filteredTags, tagged
}

// compareTags compares two slices of strings and returns the difference.
func (t *TagIt) compareTags(current []string, update []string) []string {
	var diff []string
	for i := 0; i < 2; i++ {
		for _, s1 := range current {
			found := false
			for _, s2 := range update {
				if s1 == s2 {
					found = true
					break
				}
			}
			if !found {
				diff = append(diff, s1)
			}
		}
		if i == 0 {
			current, update = update, current
		}
	}
	return diff
}
