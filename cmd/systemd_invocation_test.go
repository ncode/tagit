package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveSystemdInput_usesRunInputAndSystemdFields(t *testing.T) {
	resetViper(t)
	cmd := newSystemdIntakeTestCommand()
	setFlag(t, cmd.Flags(), "consul-addr", "consul.example:8500")
	setFlag(t, cmd.Flags(), "service-id", "api")
	setFlag(t, cmd.Flags(), "script", "/opt/tagit/tags.sh")
	setFlag(t, cmd.Flags(), "tag-prefix", "role")
	setFlag(t, cmd.Flags(), "interval", "15s")
	setFlag(t, cmd.Flags(), "token", "secret")
	setFlag(t, cmd.Flags(), "user", "tagit")
	setFlag(t, cmd.Flags(), "group", "tagit")

	got, err := resolveSystemdInput(cmd)
	if err != nil {
		t.Fatalf("resolveSystemdInput() error = %v", err)
	}

	if got.Invocation.ServiceID != "api" || got.Invocation.Script != "/opt/tagit/tags.sh" ||
		got.Invocation.TagPrefix != "role" || got.Invocation.Interval != "15s" ||
		got.Invocation.Token != "secret" || got.Invocation.ConsulAddr != "consul.example:8500" {
		t.Fatalf("Invocation = %#v, want resolved run values", got.Invocation)
	}
	if got.User != "tagit" || got.Group != "tagit" {
		t.Fatalf("User/Group = %q/%q, want tagit/tagit", got.User, got.Group)
	}
}

func TestResolveSystemdInput_validatesUserAndGroup(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		group   string
		wantErr string
	}{
		{name: "missing user", group: "tagit", wantErr: "user is required"},
		{name: "missing group", user: "tagit", wantErr: "group is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			cmd := newSystemdIntakeTestCommand()
			setFlag(t, cmd.Flags(), "service-id", "api")
			setFlag(t, cmd.Flags(), "script", "/opt/tagit/tags.sh")
			setFlag(t, cmd.Flags(), "tag-prefix", "role")
			setFlag(t, cmd.Flags(), "interval", "15s")
			setFlag(t, cmd.Flags(), "user", tt.user)
			setFlag(t, cmd.Flags(), "group", tt.group)

			_, err := resolveSystemdInput(cmd)
			if err == nil {
				t.Fatal("resolveSystemdInput() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("resolveSystemdInput() error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}

func newSystemdIntakeTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "systemd"}
	addSystemdFlags(cmd.Flags())
	return cmd
}
