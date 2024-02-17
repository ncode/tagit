package tagit

import (
	"reflect"
	"slices"
	"testing"

	"github.com/hashicorp/consul/api"
)

func TestDiffTags(t *testing.T) {
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
			diff := tagit.diffTags(tt.current, tt.update)
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

func TestNeedsTag(t *testing.T) {
	tests := []struct {
		name           string
		current        []string
		update         []string
		expectedTags   []string
		expectedShould bool
	}{
		{
			name:           "No Update Needed",
			current:        []string{"tag-tag1", "tag-tag2", "tag-tag3"},
			update:         []string{"tag-tag1", "tag-tag2", "tag-tag3"},
			expectedTags:   []string{},
			expectedShould: false,
		},
		{
			name:           "Update Needed",
			current:        []string{"tag-tag1", "tag-tag2"},
			update:         []string{"tag-tag1", "tag-tag2", "tag3"},
			expectedTags:   []string{"tag3"},
			expectedShould: true,
		},
		{
			name:           "All New Tags",
			current:        []string{},
			update:         []string{"tag1", "tag2", "tag3"},
			expectedTags:   []string{"tag1", "tag2", "tag3"},
			expectedShould: true,
		},
		{
			name:           "Current Tags Removed",
			current:        []string{"tag-tag1", "tag2", "tag3"},
			update:         []string{},
			expectedTags:   []string{"tag2", "tag3"},
			expectedShould: true,
		},
		{
			name:           "Mixed Changes",
			current:        []string{"tag-tag1", "tag2", "tag4"},
			update:         []string{"tag2", "tag3", "tag5"},
			expectedTags:   []string{"tag4", "tag3", "tag5"},
			expectedShould: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagit := TagIt{TagPrefix: "tag"} // Assuming TagPrefix is set for needsTag
			filteredTags, shouldTag := tagit.needsTag(tt.current, tt.update)
			//fmt.Println(filteredTags, shouldTag)

			if slices.Compare(filteredTags, tt.expectedTags) != 0 || shouldTag != tt.expectedShould {
				t.Errorf("needsTag() = %v, %v, want %v, %v", filteredTags, shouldTag, tt.expectedTags, tt.expectedShould)
			}
		})
	}
}

func TestCopyServiceToRegistration(t *testing.T) {
	tests := []struct {
		name        string
		service     *api.AgentService
		expectedReg *api.AgentServiceRegistration
	}{
		{
			name: "Copy All Fields",
			service: &api.AgentService{
				ID:      "service-1",
				Service: "test-service",
				Tags:    []string{"tag1", "tag2"},
				Port:    8080,
				Address: "127.0.0.1",
				Kind:    api.ServiceKindTypical,
				Weights: api.AgentWeights{
					Passing: 10,
					Warning: 1,
				},
				Meta: map[string]string{"version": "1.0"},
			},
			expectedReg: &api.AgentServiceRegistration{
				ID:      "service-1",
				Name:    "test-service",
				Tags:    []string{"tag1", "tag2"},
				Port:    8080,
				Address: "127.0.0.1",
				Kind:    api.ServiceKindTypical,
				Weights: &api.AgentWeights{
					Passing: 10,
					Warning: 1,
				},
				Meta: map[string]string{"version": "1.0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagit := TagIt{}
			reg := tagit.copyServiceToRegistration(tt.service)

			if !reflect.DeepEqual(reg, tt.expectedReg) {
				t.Errorf("copyServiceToRegistration() got = %v, want %v", reg, tt.expectedReg)
			}
		})
	}
}
