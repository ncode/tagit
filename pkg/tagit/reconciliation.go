package tagit

import (
	"fmt"
	"slices"
	"strings"
)

// Reconciler applies TagIt's managed tag policy for one tag prefix.
type Reconciler struct {
	prefix string
}

// Reconciliation describes the stable tag set and whether Consul needs a write.
type Reconciliation struct {
	Tags    []string
	Changed bool
}

// NewReconciler returns a managed tag reconciler for prefix-value tags.
func NewReconciler(prefix string) Reconciler {
	return Reconciler{prefix: prefix}
}

// Reconcile converts script output into managed tags and merges them with unmanaged tags.
func (r Reconciler) Reconcile(current []string, output []byte) Reconciliation {
	return r.ReconcileManaged(current, r.managedTags(strings.Fields(string(output))))
}

// ReconcileManaged merges desired managed tags with current unmanaged tags.
func (r Reconciler) ReconcileManaged(current, desiredManaged []string) Reconciliation {
	unmanaged, currentManaged := r.split(current)
	desiredManaged = stableTags(desiredManaged)
	tags := stableTags(append(unmanaged, desiredManaged...))

	return Reconciliation{
		Tags:    tags,
		Changed: !sameTags(stableTags(currentManaged), desiredManaged),
	}
}

// Cleanup removes only tags owned by this managed tag prefix.
func (r Reconciler) Cleanup(current []string) Reconciliation {
	unmanaged, currentManaged := r.split(current)
	return Reconciliation{
		Tags:    stableTags(unmanaged),
		Changed: len(currentManaged) > 0,
	}
}

func (r Reconciler) managedTags(tokens []string) []string {
	tags := make([]string, 0, len(tokens))
	for _, tag := range tokens {
		tags = append(tags, fmt.Sprintf("%s-%s", r.prefix, tag))
	}
	return stableTags(tags)
}

func (r Reconciler) split(tags []string) (unmanaged, managed []string) {
	managedPrefix := r.prefix + "-"
	for _, tag := range tags {
		if strings.HasPrefix(tag, managedPrefix) {
			managed = append(managed, tag)
		} else {
			unmanaged = append(unmanaged, tag)
		}
	}
	return unmanaged, managed
}

func stableTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	stable := slices.Clone(tags)
	slices.Sort(stable)
	return slices.Compact(stable)
}

func sameTags(a, b []string) bool {
	return slices.Equal(a, b)
}
