package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/ncode/tagit/pkg/consul"
	"github.com/ncode/tagit/pkg/tagit"
)

func TestCleanupCommand_validatesInputBeforeCreatingClient(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "cleanup", withSharedPersistentFlags)

	clientCalls := 0
	deps := commandDeps{
		Logger: discardLogger(),
		NewClient: func(address, token string) (consul.Client, error) {
			clientCalls++
			return commandClient{}, nil
		},
		NewTagger: func(consul.Client, tagit.CommandExecutor, commandInput, *slog.Logger) tagger {
			t.Fatal("NewTagger called before validation")
			return nil
		},
	}

	err := cleanupCommand(cmd, deps)
	if err == nil {
		t.Fatal("cleanupCommand() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "service-id is required") {
		t.Fatalf("cleanupCommand() error = %q, want service-id validation", err)
	}
	if clientCalls != 0 {
		t.Fatalf("NewClient calls = %d, want 0", clientCalls)
	}
}

func TestCleanupCommand_usesResolvedInput(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "cleanup", withSharedPersistentFlags)
	setFlag(t, cmd.InheritedFlags(), "consul-addr", "consul.example:8500")
	setFlag(t, cmd.InheritedFlags(), "service-id", "api")
	setFlag(t, cmd.InheritedFlags(), "tag-prefix", "role")
	setFlag(t, cmd.InheritedFlags(), "token", "secret")

	var gotAddress string
	var gotToken string
	var gotInput commandInput
	fakeTagger := &commandTagger{}
	deps := commandDeps{
		Logger: discardLogger(),
		NewClient: func(address, token string) (consul.Client, error) {
			gotAddress = address
			gotToken = token
			return commandClient{}, nil
		},
		NewExecutor: func() tagit.CommandExecutor {
			return &tagit.CmdExecutor{}
		},
		NewTagger: func(client consul.Client, executor tagit.CommandExecutor, input commandInput, logger *slog.Logger) tagger {
			gotInput = input
			return fakeTagger
		},
	}

	if err := cleanupCommand(cmd, deps); err != nil {
		t.Fatalf("cleanupCommand() error = %v", err)
	}

	if gotAddress != "consul.example:8500" {
		t.Fatalf("address = %q, want consul.example:8500", gotAddress)
	}
	if gotToken != "secret" {
		t.Fatalf("token = %q, want secret", gotToken)
	}
	wantInput := commandInput{
		ConsulAddr:  "consul.example:8500",
		ServiceID:   "api",
		TagPrefix:   "role",
		IntervalRaw: "60s",
		Token:       "secret",
	}
	if gotInput != wantInput {
		t.Fatalf("input = %#v, want %#v", gotInput, wantInput)
	}
	if fakeTagger.cleanupCalls != 1 {
		t.Fatalf("CleanupTags calls = %d, want 1", fakeTagger.cleanupCalls)
	}
}

func TestCleanupCommand_returnsClientErrors(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "cleanup", withSharedPersistentFlags)
	setFlag(t, cmd.InheritedFlags(), "service-id", "api")

	deps := commandDeps{
		Logger: discardLogger(),
		NewClient: func(address, token string) (consul.Client, error) {
			return nil, fmt.Errorf("connect consul")
		},
		NewTagger: func(consul.Client, tagit.CommandExecutor, commandInput, *slog.Logger) tagger {
			t.Fatal("NewTagger called after client error")
			return nil
		},
	}

	err := cleanupCommand(cmd, deps)
	if err == nil {
		t.Fatal("cleanupCommand() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "connect consul") {
		t.Fatalf("cleanupCommand() error = %q, want client error", err)
	}
}

func TestCleanupCommand_returnsCleanupErrors(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "cleanup", withSharedPersistentFlags)
	setFlag(t, cmd.InheritedFlags(), "service-id", "api")

	deps := commandDeps{
		Logger: discardLogger(),
		NewClient: func(address, token string) (consul.Client, error) {
			return commandClient{}, nil
		},
		NewTagger: func(consul.Client, tagit.CommandExecutor, commandInput, *slog.Logger) tagger {
			return cleanupErrorTagger{err: fmt.Errorf("consul write failed")}
		},
	}

	err := cleanupCommand(cmd, deps)
	if err == nil {
		t.Fatal("cleanupCommand() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "failed to clean up tags: consul write failed") {
		t.Fatalf("cleanupCommand() error = %q, want cleanup context", err)
	}
}

func TestCleanupCmd_RunEUsesSharedHandler(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "cleanup", withSharedPersistentFlags)

	err := cleanupCmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("cleanupCmd.RunE() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "service-id is required") {
		t.Fatalf("cleanupCmd.RunE() error = %q, want service-id validation", err)
	}
}

type cleanupErrorTagger struct {
	err error
}

func (cet cleanupErrorTagger) Run(context.Context) {}

func (cet cleanupErrorTagger) CleanupTags() error {
	return cet.err
}
