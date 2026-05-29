package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ncode/tagit/pkg/consul"
	"github.com/ncode/tagit/pkg/tagit"
	"github.com/spf13/cobra"
)

type tagger interface {
	Run(context.Context)
	CleanupTags() error
}

type commandDeps struct {
	Logger      *slog.Logger
	NewClient   func(address, token string) (consul.Client, error)
	NewExecutor func() tagit.CommandExecutor
	NewTagger   func(consul.Client, tagit.CommandExecutor, commandInput, *slog.Logger) tagger
}

func runCommandWithContext(ctx context.Context, cmd *cobra.Command, deps commandDeps) error {
	deps = deps.withDefaults()

	input, err := resolveRunInput(cmd)
	if err != nil {
		deps.Logger.Error("Invalid command input", "error", err)
		return err
	}

	consulClient, err := deps.NewClient(input.ConsulAddr, input.Token)
	if err != nil {
		deps.Logger.Error("Failed to create Consul client", "error", err)
		return err
	}

	t := deps.NewTagger(consulClient, deps.NewExecutor(), input, deps.Logger)
	deps.Logger.Info("Starting tagit",
		"serviceID", input.ServiceID,
		"script", input.Script,
		"interval", input.Interval,
		"tagPrefix", input.TagPrefix)

	t.Run(ctx)

	deps.Logger.Info("Tagit has stopped")
	return nil
}

func cleanupCommand(cmd *cobra.Command, deps commandDeps) error {
	deps = deps.withDefaults()

	input, err := resolveCleanupInput(cmd)
	if err != nil {
		deps.Logger.Error("Invalid command input", "error", err)
		return err
	}

	consulClient, err := deps.NewClient(input.ConsulAddr, input.Token)
	if err != nil {
		deps.Logger.Error("Failed to create Consul client", "error", err)
		return err
	}

	t := deps.NewTagger(consulClient, deps.NewExecutor(), input, deps.Logger)
	deps.Logger.Info("Starting tag cleanup", "serviceID", input.ServiceID, "tagPrefix", input.TagPrefix)

	if err := t.CleanupTags(); err != nil {
		deps.Logger.Error("Failed to clean up tags", "error", err)
		return fmt.Errorf("failed to clean up tags: %w", err)
	}

	deps.Logger.Info("Tag cleanup completed successfully")
	return nil
}

func (deps commandDeps) withDefaults() commandDeps {
	if deps.Logger == nil {
		deps.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}
	if deps.NewClient == nil {
		deps.NewClient = consul.CreateClient
	}
	if deps.NewExecutor == nil {
		deps.NewExecutor = func() tagit.CommandExecutor {
			return &tagit.CmdExecutor{}
		}
	}
	if deps.NewTagger == nil {
		deps.NewTagger = func(client consul.Client, executor tagit.CommandExecutor, input commandInput, logger *slog.Logger) tagger {
			return tagit.New(
				client,
				executor,
				input.ServiceID,
				input.Script,
				input.Interval,
				input.TagPrefix,
				logger,
			)
		}
	}
	return deps
}
