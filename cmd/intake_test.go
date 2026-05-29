package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestResolveRunInput_readsCLIValues(t *testing.T) {
	resetViper(t)

	cmd := newIntakeTestCommand(t, "run", withSharedPersistentFlags)
	setFlag(t, cmd.InheritedFlags(), "consul-addr", "consul.example:8500")
	setFlag(t, cmd.InheritedFlags(), "service-id", "api")
	setFlag(t, cmd.InheritedFlags(), "script", "/opt/tagit/tags.sh")
	setFlag(t, cmd.InheritedFlags(), "tag-prefix", "role")
	setFlag(t, cmd.InheritedFlags(), "interval", "15s")
	setFlag(t, cmd.InheritedFlags(), "token", "secret")

	got, err := resolveRunInput(cmd)
	if err != nil {
		t.Fatalf("resolveRunInput() error = %v", err)
	}

	want := commandInput{
		ConsulAddr:  "consul.example:8500",
		ServiceID:   "api",
		Script:      "/opt/tagit/tags.sh",
		TagPrefix:   "role",
		Interval:    15 * time.Second,
		IntervalRaw: "15s",
		Token:       "secret",
	}
	if got != want {
		t.Fatalf("resolveRunInput() = %#v, want %#v", got, want)
	}
}

func TestResolveRunInput_readsConfigAndEnvironment(t *testing.T) {
	resetViper(t)
	configureCommandIntakeEnv()
	t.Setenv("TAGIT_SERVICE_ID", "env-service")
	t.Setenv("TAGIT_TOKEN", "env-token")
	t.Setenv("TAGIT_INTERVAL", "45s")

	writeIntakeConfig(t, `consul-addr: "config-consul:8500"
service-id: "config-service"
script: "/config/script.sh"
tag-prefix: "config-prefix"
interval: "30s"
token: "config-token"
`)

	cmd := newIntakeTestCommand(t, "run", withSharedPersistentFlags)
	setFlag(t, cmd.InheritedFlags(), "consul-addr", "cli-consul:8500")

	got, err := resolveRunInput(cmd)
	if err != nil {
		t.Fatalf("resolveRunInput() error = %v", err)
	}

	if got.ConsulAddr != "cli-consul:8500" {
		t.Fatalf("ConsulAddr = %q, want CLI override", got.ConsulAddr)
	}
	if got.ServiceID != "env-service" {
		t.Fatalf("ServiceID = %q, want environment value", got.ServiceID)
	}
	if got.Script != "/config/script.sh" {
		t.Fatalf("Script = %q, want config value", got.Script)
	}
	if got.TagPrefix != "config-prefix" {
		t.Fatalf("TagPrefix = %q, want config value", got.TagPrefix)
	}
	if got.Interval != 45*time.Second {
		t.Fatalf("Interval = %v, want environment value", got.Interval)
	}
	if got.Token != "env-token" {
		t.Fatalf("Token = %q, want environment value", got.Token)
	}
}

func TestResolveRunInput_validationErrors(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]string
		wantErr string
	}{
		{
			name: "missing service ID",
			values: map[string]string{
				"script":   "/opt/tagit/tags.sh",
				"interval": "15s",
			},
			wantErr: "service-id is required",
		},
		{
			name: "missing script",
			values: map[string]string{
				"service-id": "api",
				"interval":   "15s",
			},
			wantErr: "script is required",
		},
		{
			name: "missing interval",
			values: map[string]string{
				"service-id": "api",
				"script":     "/opt/tagit/tags.sh",
				"interval":   "",
			},
			wantErr: "interval is required",
		},
		{
			name: "zero interval",
			values: map[string]string{
				"service-id": "api",
				"script":     "/opt/tagit/tags.sh",
				"interval":   "0s",
			},
			wantErr: "interval must be greater than zero",
		},
		{
			name: "invalid interval",
			values: map[string]string{
				"service-id": "api",
				"script":     "/opt/tagit/tags.sh",
				"interval":   "soon",
			},
			wantErr: "invalid interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			cmd := newIntakeTestCommand(t, "run", withSharedPersistentFlags)
			for key, value := range tt.values {
				setFlag(t, cmd.InheritedFlags(), key, value)
			}

			_, err := resolveRunInput(cmd)
			if err == nil {
				t.Fatal("resolveRunInput() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("resolveRunInput() error = %q, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestResolveCleanupInput_validationErrors(t *testing.T) {
	resetViper(t)
	cmd := newIntakeTestCommand(t, "cleanup", withSharedPersistentFlags)

	_, err := resolveCleanupInput(cmd)
	if err == nil {
		t.Fatal("resolveCleanupInput() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "service-id is required") {
		t.Fatalf("resolveCleanupInput() error = %q, want service-id validation", err)
	}
}

func resetViper(t *testing.T) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)
}

func writeIntakeConfig(t *testing.T, content string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".tagit.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("read config: %v", err)
	}
}

func newIntakeTestCommand(t *testing.T, use string, flagSetups ...func(*pflag.FlagSet)) *cobra.Command {
	t.Helper()

	parent := &cobra.Command{Use: "tagit"}
	for _, setup := range flagSetups {
		setup(parent.PersistentFlags())
	}
	child := &cobra.Command{Use: use}
	parent.AddCommand(child)
	return child
}

func withSharedPersistentFlags(flags *pflag.FlagSet) {
	flags.StringP("consul-addr", "c", "127.0.0.1:8500", "consul address")
	flags.StringP("service-id", "s", "", "consul service id")
	flags.StringP("script", "x", "", "path to script used to generate tags")
	flags.StringP("tag-prefix", "p", "tagged", "prefix to be added to tags")
	flags.StringP("interval", "i", "60s", "interval to run the script")
	flags.StringP("token", "t", "", "consul token")
}

func setFlag(t *testing.T, flags *pflag.FlagSet, name, value string) {
	t.Helper()

	if err := flags.Set(name, value); err != nil {
		t.Fatalf("set flag %s: %v", name, err)
	}
}
