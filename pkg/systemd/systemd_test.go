package systemd

import (
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	// Define a test case struct
	type testCase struct {
		name      string
		fields    Fields
		wantErr   bool
		checkStr  string // A substring we expect in the output
		expectStr bool   // A flag to indicate the absence of a substring
	}

	// Define your test cases
	testCases := []testCase{
		{
			name: "Basic test",
			fields: Fields{
				ServiceID:  "testservice",
				Script:     "testscript",
				TagPrefix:  "testprefix",
				Interval:   "testinterval",
				Token:      "testtoken",
				ConsulAddr: "testaddr",
				User:       "testuser",
				Group:      "testgroup",
			},
			wantErr:   false,
			checkStr:  "ExecStart=/usr/bin/tagit run -s testservice -x testscript",
			expectStr: true,
		},
		{
			name: "With Token",
			fields: Fields{
				ServiceID: "testservice",
				Token:     "sometoken",
			},
			wantErr:   false,
			checkStr:  "-t sometoken",
			expectStr: true,
		},
		{
			name: "Without Token",
			fields: Fields{
				ServiceID: "testservice",
			},
			wantErr:   false,
			checkStr:  "-t",
			expectStr: false,
		},
		{
			name: "With Consul Address",
			fields: Fields{
				ServiceID:  "testservice",
				ConsulAddr: "someaddress",
			},
			wantErr:   false,
			checkStr:  "-c someaddress",
			expectStr: true,
		},
		{
			name: "Without Consul Address",
			fields: Fields{
				ServiceID: "testservice",
			},
			wantErr:   false,
			checkStr:  "-c someaddress",
			expectStr: false,
		},
		{
			name:      "Empty Fields",
			fields:    Fields{},
			wantErr:   false,
			checkStr:  "ExecStart=/usr/bin/tagit run -s",
			expectStr: true,
		},
		{
			name: "Consul Address Only",
			fields: Fields{
				ConsulAddr: "127.0.0.1",
			},
			wantErr:   false,
			checkStr:  "-c 127.0.0.1",
			expectStr: true,
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderTemplate(&tc.fields)
			if (err != nil) != tc.wantErr {
				t.Errorf("RenderTemplate() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			// Check for the presence or absence of the token
			if tc.expectStr && !strings.Contains(got, tc.checkStr) {
				t.Errorf("RenderTemplate() = %v, want %v", got, tc.checkStr)
			} else if !tc.expectStr && strings.Contains(got, tc.checkStr) {
				t.Errorf("RenderTemplate() = %v, should not contain %v", got, tc.checkStr)
			}
		})
	}
}

func TestRenderTemplateFailure(t *testing.T) {
	_, err := RenderTemplate(nil)
	if err == nil {
		t.Errorf("RenderTemplate() with nil input did not fail as expected")
	}
}
