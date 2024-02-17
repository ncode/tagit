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

func TestExcludeTagged(t *testing.T) {
	tests := []struct {
		name      string
		tags      []string
		tagPrefix string
		expected  []string
		shouldTag bool
	}{
		{
			name:      "No Tags With Prefix",
			tags:      []string{"alpha", "beta", "gamma"},
			tagPrefix: "tag",
			expected:  []string{"alpha", "beta", "gamma"},
			shouldTag: false,
		},
		{
			name:      "All Tags With Prefix",
			tags:      []string{"tag-alpha", "tag-beta", "tag-gamma"},
			tagPrefix: "tag",
			expected:  []string{},
			shouldTag: true,
		},
		{
			name:      "Some Tags With Prefix",
			tags:      []string{"alpha", "tag-beta", "gamma"},
			tagPrefix: "tag",
			expected:  []string{"alpha", "gamma"},
			shouldTag: true,
		},
		{
			name:      "Empty Tags",
			tags:      []string{},
			tagPrefix: "tag",
			expected:  []string{},
			shouldTag: false,
		},
		{
			name:      "Prefix in Middle",
			tags:      []string{"alpha-tag", "beta", "gamma"},
			tagPrefix: "tag",
			expected:  []string{"alpha-tag", "beta", "gamma"},
			shouldTag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagit := TagIt{TagPrefix: tt.tagPrefix}
			filteredTags, tagged := tagit.excludeTagged(tt.tags)

			if slices.Compare(filteredTags, tt.expected) != 0 || tagged != tt.shouldTag {
				t.Errorf("excludeTagged() = %v, %v, want %v, %v", filteredTags, tagged, tt.expected, tt.shouldTag)
			}
		})
	}
}
