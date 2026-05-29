package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/ncode/tagit/pkg/systemd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type commandInput struct {
	ConsulAddr  string
	ServiceID   string
	Script      string
	TagPrefix   string
	Interval    time.Duration
	IntervalRaw string
	Token       string
}

type systemdInput struct {
	Invocation systemd.Invocation
	User       string
	Group      string
}

func configureCommandIntakeEnv() {
	viper.SetEnvPrefix("TAGIT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func resolveRunInput(cmd *cobra.Command) (commandInput, error) {
	input := resolveSharedInput(cmd)
	if strings.TrimSpace(input.ServiceID) == "" {
		return commandInput{}, fmt.Errorf("service-id is required")
	}
	if strings.TrimSpace(input.Script) == "" {
		return commandInput{}, fmt.Errorf("script is required")
	}
	interval, err := parseRequiredInterval(input.IntervalRaw)
	if err != nil {
		return commandInput{}, err
	}
	input.Interval = interval
	return input, nil
}

func resolveCleanupInput(cmd *cobra.Command) (commandInput, error) {
	input := resolveSharedInput(cmd)
	if strings.TrimSpace(input.ServiceID) == "" {
		return commandInput{}, fmt.Errorf("service-id is required")
	}
	return input, nil
}

func resolveSystemdInput(cmd *cobra.Command) (systemdInput, error) {
	input, err := resolveRunInput(cmd)
	if err != nil {
		return systemdInput{}, err
	}

	user := resolveString(cmd, "user")
	if strings.TrimSpace(user) == "" {
		return systemdInput{}, fmt.Errorf("user is required")
	}
	group := resolveString(cmd, "group")
	if strings.TrimSpace(group) == "" {
		return systemdInput{}, fmt.Errorf("group is required")
	}

	return systemdInput{
		Invocation: systemd.Invocation{
			ServiceID:  input.ServiceID,
			Script:     input.Script,
			TagPrefix:  input.TagPrefix,
			Interval:   input.IntervalRaw,
			Token:      input.Token,
			ConsulAddr: input.ConsulAddr,
		},
		User:  user,
		Group: group,
	}, nil
}

func resolveSharedInput(cmd *cobra.Command) commandInput {
	return commandInput{
		ConsulAddr:  resolveString(cmd, "consul-addr"),
		ServiceID:   resolveString(cmd, "service-id"),
		Script:      resolveString(cmd, "script"),
		TagPrefix:   resolveString(cmd, "tag-prefix"),
		IntervalRaw: resolveString(cmd, "interval"),
		Token:       resolveString(cmd, "token"),
	}
}

func parseRequiredInterval(raw string) (time.Duration, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, fmt.Errorf("interval is required and cannot be empty or zero")
	}
	interval, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid interval %q: %w", raw, err)
	}
	if interval <= 0 {
		return 0, fmt.Errorf("interval is required and cannot be empty or zero: interval must be greater than zero")
	}
	return interval, nil
}

func resolveString(cmd *cobra.Command, key string) string {
	flag := lookupFlag(cmd, key)
	if flag != nil && flag.Changed {
		return flag.Value.String()
	}
	if viper.IsSet(key) {
		return viper.GetString(key)
	}
	if flag != nil {
		return flag.Value.String()
	}
	return ""
}

func lookupFlag(cmd *cobra.Command, key string) *pflag.Flag {
	if flag := cmd.Flags().Lookup(key); flag != nil {
		return flag
	}
	return cmd.InheritedFlags().Lookup(key)
}
