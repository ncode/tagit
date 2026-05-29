package systemd

import (
	"strings"
	"testing"
)

func TestRenderInvocation(t *testing.T) {
	tests := []struct {
		name       string
		invocation Invocation
		want       string
	}{
		{
			name: "required invocation values",
			invocation: Invocation{
				ServiceID: "api",
				Script:    "/opt/tagit/tags.sh",
				TagPrefix: "role",
				Interval:  "15s",
			},
			want: "/usr/bin/tagit run -s api -x /opt/tagit/tags.sh -p role -i 15s",
		},
		{
			name: "optional invocation values",
			invocation: Invocation{
				ServiceID:  "api",
				Script:     "/opt/tagit/tags.sh",
				TagPrefix:  "role",
				Interval:   "15s",
				Token:      "secret",
				ConsulAddr: "consul.example:8500",
			},
			want: "/usr/bin/tagit run -s api -x /opt/tagit/tags.sh -p role -i 15s -t secret -c consul.example:8500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderInvocation(tt.invocation)
			if got != tt.want {
				t.Fatalf("RenderInvocation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewFieldsFromInvocation_validatesSystemdFields(t *testing.T) {
	invocation := Invocation{
		ServiceID: "api",
		Script:    "/opt/tagit/tags.sh",
		TagPrefix: "role",
		Interval:  "15s",
	}

	_, err := NewFieldsFromInvocation(invocation, "", "tagit")
	if err == nil {
		t.Fatal("NewFieldsFromInvocation() error = nil, want missing user error")
	}
	if !strings.Contains(err.Error(), "User") {
		t.Fatalf("NewFieldsFromInvocation() error = %q, want User", err)
	}

	_, err = NewFieldsFromInvocation(invocation, "tagit", "")
	if err == nil {
		t.Fatal("NewFieldsFromInvocation() error = nil, want missing group error")
	}
	if !strings.Contains(err.Error(), "Group") {
		t.Fatalf("NewFieldsFromInvocation() error = %q, want Group", err)
	}
}

func TestNewFieldsFromInvocation_setsExecStart(t *testing.T) {
	fields, err := NewFieldsFromInvocation(Invocation{
		ServiceID:  "api",
		Script:     "/opt/tagit/tags.sh",
		TagPrefix:  "role",
		Interval:   "15s",
		Token:      "secret",
		ConsulAddr: "consul.example:8500",
	}, "tagit", "tagit")
	if err != nil {
		t.Fatalf("NewFieldsFromInvocation() error = %v", err)
	}

	want := "/usr/bin/tagit run -s api -x /opt/tagit/tags.sh -p role -i 15s -t secret -c consul.example:8500"
	if fields.ExecStart != want {
		t.Fatalf("ExecStart = %q, want %q", fields.ExecStart, want)
	}
}
