package tagit

import (
	"reflect"
	"slices"
	"testing"
)

func TestCompareTags(t *testing.T) {
	tests := []struct {
		name     string
		current  []string
		update   []string
		expected []string
	}{
		{
			name:     "No Difference",
			current:  []string{"tag1", "tag2", "tag3"},
			update:   []string{"tag1", "tag2", "tag3"},
			expected: []string{},
		},
		{
			name:     "Difference In Current",
			current:  []string{"tag1", "tag2", "tag4"},
			update:   []string{"tag1", "tag2", "tag3"},
			expected: []string{"tag3", "tag4"},
		},
		{
			name:     "Difference In Update",
			current:  []string{"tag1", "tag2"},
			update:   []string{"tag1", "tag2", "tag3"},
			expected: []string{"tag3"},
		},
		{
			name:     "Empty Current",
			current:  []string{},
			update:   []string{"tag1", "tag2"},
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "Empty Update",
			current:  []string{"tag1", "tag2"},
			update:   []string{},
			expected: []string{"tag1", "tag2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagit := TagIt{} // Assuming TagIt doesn't require initialization for compareTags
			diff := tagit.compareTags(tt.current, tt.update)
			if (len(diff) == 0) && (len(tt.expected) == 0) {
				return
			}
			slices.Sort(diff)
			slices.Sort(tt.expected)
			if !reflect.DeepEqual(diff, tt.expected) {
				t.Errorf("compareTags(%v, %v) = %v, want %v", tt.current, tt.update, diff, tt.expected)
			}
		})
	}
}
