package tagit

import (
	"slices"
	"testing"
)

func TestReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name        string
		current     []string
		output      []byte
		wantTags    []string
		wantChanged bool
	}{
		{
			name:        "script output becomes managed tags",
			output:      []byte("primary replica"),
			wantTags:    []string{"role-primary", "role-replica"},
			wantChanged: true,
		},
		{
			name:        "unmanaged tags are preserved",
			current:     []string{"static"},
			output:      []byte("primary"),
			wantTags:    []string{"role-primary", "static"},
			wantChanged: true,
		},
		{
			name:        "stale managed tags are removed",
			current:     []string{"role-old", "static"},
			output:      []byte("new"),
			wantTags:    []string{"role-new", "static"},
			wantChanged: true,
		},
		{
			name:        "identical managed tags do not need a write",
			current:     []string{"static", "role-primary"},
			output:      []byte("primary"),
			wantTags:    []string{"role-primary", "static"},
			wantChanged: false,
		},
		{
			name:        "duplicate desired tags are compacted in stable order",
			output:      []byte("replica primary primary"),
			wantTags:    []string{"role-primary", "role-replica"},
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewReconciler("role").Reconcile(tt.current, tt.output)

			if !slices.Equal(got.Tags, tt.wantTags) {
				t.Fatalf("Tags = %v, want %v", got.Tags, tt.wantTags)
			}
			if got.Changed != tt.wantChanged {
				t.Fatalf("Changed = %v, want %v", got.Changed, tt.wantChanged)
			}
		})
	}
}

func TestReconciler_Cleanup(t *testing.T) {
	got := NewReconciler("role").Cleanup([]string{"role-primary", "static", "role-replica"})

	if want := []string{"static"}; !slices.Equal(got.Tags, want) {
		t.Fatalf("Tags = %v, want %v", got.Tags, want)
	}
	if !got.Changed {
		t.Fatal("Changed = false, want true")
	}
}

func TestReconciler_CleanupNoop(t *testing.T) {
	got := NewReconciler("role").Cleanup([]string{"static"})

	if want := []string{"static"}; !slices.Equal(got.Tags, want) {
		t.Fatalf("Tags = %v, want %v", got.Tags, want)
	}
	if got.Changed {
		t.Fatal("Changed = true, want false")
	}
}
