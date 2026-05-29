package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/ncode/tagit/pkg/consul"
	"github.com/ncode/tagit/pkg/tagit"
)

func TestRunCommand_validatesInputBeforeCreatingClient(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "run", withSharedPersistentFlags)
	setFlag(t, cmd.InheritedFlags(), "service-id", "api")
	setFlag(t, cmd.InheritedFlags(), "interval", "15s")

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

	err := runCommandWithContext(t.Context(), cmd, deps)
	if err == nil {
		t.Fatal("runCommandWithContext() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "script is required") {
		t.Fatalf("runCommandWithContext() error = %q, want script validation", err)
	}
	if clientCalls != 0 {
		t.Fatalf("NewClient calls = %d, want 0", clientCalls)
	}
}

func TestRunCommand_usesResolvedInput(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "run", withSharedPersistentFlags)
	setFlag(t, cmd.InheritedFlags(), "consul-addr", "consul.example:8500")
	setFlag(t, cmd.InheritedFlags(), "service-id", "api")
	setFlag(t, cmd.InheritedFlags(), "script", "echo primary")
	setFlag(t, cmd.InheritedFlags(), "tag-prefix", "role")
	setFlag(t, cmd.InheritedFlags(), "interval", "15s")
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
			if client == nil {
				t.Fatal("client = nil")
			}
			if executor == nil {
				t.Fatal("executor = nil")
			}
			if logger == nil {
				t.Fatal("logger = nil")
			}
			gotInput = input
			return fakeTagger
		},
	}

	if err := runCommandWithContext(t.Context(), cmd, deps); err != nil {
		t.Fatalf("runCommandWithContext() error = %v", err)
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
		Script:      "echo primary",
		TagPrefix:   "role",
		Interval:    15 * time.Second,
		IntervalRaw: "15s",
		Token:       "secret",
	}
	if gotInput != wantInput {
		t.Fatalf("input = %#v, want %#v", gotInput, wantInput)
	}
	if fakeTagger.runCalls != 1 {
		t.Fatalf("Run calls = %d, want 1", fakeTagger.runCalls)
	}
}

func TestRunCommand_returnsClientErrors(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "run", withSharedPersistentFlags)
	setFlag(t, cmd.InheritedFlags(), "service-id", "api")
	setFlag(t, cmd.InheritedFlags(), "script", "echo primary")
	setFlag(t, cmd.InheritedFlags(), "interval", "15s")

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

	err := runCommandWithContext(t.Context(), cmd, deps)
	if err == nil {
		t.Fatal("runCommandWithContext() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "connect consul") {
		t.Fatalf("runCommandWithContext() error = %q, want client error", err)
	}
}

func TestRunCommand_passesCancellationToTagger(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "run", withSharedPersistentFlags)
	setFlag(t, cmd.InheritedFlags(), "service-id", "api")
	setFlag(t, cmd.InheritedFlags(), "script", "echo primary")
	setFlag(t, cmd.InheritedFlags(), "interval", "15s")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	observedCancellation := false
	deps := commandDeps{
		Logger: discardLogger(),
		NewClient: func(address, token string) (consul.Client, error) {
			return commandClient{}, nil
		},
		NewTagger: func(consul.Client, tagit.CommandExecutor, commandInput, *slog.Logger) tagger {
			return runFuncTagger(func(ctx context.Context) {
				select {
				case <-ctx.Done():
					observedCancellation = true
				default:
				}
			})
		},
	}

	if err := runCommandWithContext(ctx, cmd, deps); err != nil {
		t.Fatalf("runCommandWithContext() error = %v", err)
	}
	if !observedCancellation {
		t.Fatal("tagger did not observe cancelled context")
	}
}

type commandClient struct{}

func (commandClient) Agent() consul.Agent {
	return commandAgent{}
}

type commandAgent struct{}

func (commandAgent) Service(string, *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
	return nil, nil, nil
}

func (commandAgent) ServiceRegister(*api.AgentServiceRegistration) error {
	return nil
}

type commandTagger struct {
	runCalls     int
	cleanupCalls int
}

func (ct *commandTagger) Run(context.Context) {
	ct.runCalls++
}

func (ct *commandTagger) CleanupTags() error {
	ct.cleanupCalls++
	return nil
}

type runFuncTagger func(context.Context)

func (rft runFuncTagger) Run(ctx context.Context) {
	rft(ctx)
}

func (rft runFuncTagger) CleanupTags() error {
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
