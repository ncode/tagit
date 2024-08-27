package systemd

import (
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	type testCase struct {
		name      string
		fields    Fields
		wantErr   bool
		checkStr  string
		expectStr bool
	}

	testCases := []testCase{
		{
			name: "All fields provided",
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
			checkStr:  "ExecStart=/usr/bin/tagit run -s testservice -x testscript -p testprefix -i testinterval -t testtoken -c testaddr",
			expectStr: true,
		},
		{
			name: "Only required fields",
			fields: Fields{
				ServiceID: "testservice",
				Script:    "testscript",
				TagPrefix: "testprefix",
				Interval:  "testinterval",
				User:      "testuser",
				Group:     "testgroup",
			},
			wantErr:   false,
			checkStr:  "ExecStart=/usr/bin/tagit run -s testservice -x testscript -p testprefix -i testinterval",
			expectStr: true,
		},
		{
			name: "With Token, without ConsulAddr",
			fields: Fields{
				ServiceID: "testservice",
				Script:    "testscript",
				TagPrefix: "testprefix",
				Interval:  "testinterval",
				Token:     "sometoken",
				User:      "testuser",
				Group:     "testgroup",
			},
			wantErr:   false,
			checkStr:  "-t sometoken",
			expectStr: true,
		},
		{
			name: "With ConsulAddr, without Token",
			fields: Fields{
				ServiceID:  "testservice",
				Script:     "testscript",
				TagPrefix:  "testprefix",
				Interval:   "testinterval",
				ConsulAddr: "someaddress",
				User:       "testuser",
				Group:      "testgroup",
			},
			wantErr:   false,
			checkStr:  "-c someaddress",
			expectStr: true,
		},
		{
			name: "Missing required field",
			fields: Fields{
				ServiceID: "testservice",
				Script:    "testscript",
				TagPrefix: "testprefix",
				Interval:  "testinterval",
				User:      "testuser",
				// Group is missing
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderTemplate(&tc.fields)
			if (err != nil) != tc.wantErr {
				t.Errorf("RenderTemplate() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}
			if tc.expectStr && !strings.Contains(got, tc.checkStr) {
				t.Errorf("RenderTemplate() = %v, want %v", got, tc.checkStr)
			} else if !tc.expectStr && strings.Contains(got, tc.checkStr) {
				t.Errorf("RenderTemplate() = %v, should not contain %v", got, tc.checkStr)
			}
		})
	}
}

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name    string
		fields  Fields
		wantErr bool
	}{
		{
			name: "All fields provided",
			fields: Fields{
				ServiceID: "test", Script: "test", TagPrefix: "test",
				Interval: "test", User: "test", Group: "test",
			},
			wantErr: false,
		},
		{
			name: "Missing ServiceID",
			fields: Fields{
				Script: "test", TagPrefix: "test", Interval: "test", User: "test", Group: "test",
			},
			wantErr: true,
		},
		{
			name: "Missing Script",
			fields: Fields{
				ServiceID: "test", TagPrefix: "test", Interval: "test", User: "test", Group: "test",
			},
			wantErr: true,
		},
		{
			name: "Empty Script",
			fields: Fields{
				ServiceID: "test", Script: "", TagPrefix: "test", Interval: "test", User: "test", Group: "test",
			},
			wantErr: true,
		},
		{
			name: "Missing multiple fields including Script",
			fields: Fields{
				ServiceID: "test", TagPrefix: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFields(&tt.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFields() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateFields() expected error, got nil")
				} else {
					if tt.name == "Missing Script" || tt.name == "Empty Script" {
						if !strings.Contains(err.Error(), "Script") {
							t.Errorf("validateFields() error does not mention 'Script': %v", err)
						}
					}
				}
			}
		})
	}
}

func TestNewFieldsFromFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		wantErr bool
	}{
		{
			name: "All required flags provided",
			flags: map[string]string{
				"service-id": "test", "script": "test", "tag-prefix": "test",
				"interval": "test", "user": "test", "group": "test",
			},
			wantErr: false,
		},
		{
			name: "Missing required flag",
			flags: map[string]string{
				"service-id": "test", "script": "test",
			},
			wantErr: true,
		},
		{
			name: "All flags provided including optional",
			flags: map[string]string{
				"service-id": "test", "script": "test", "tag-prefix": "test",
				"interval": "test", "user": "test", "group": "test",
				"token": "test", "consul-addr": "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFieldsFromFlags(tt.flags)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFieldsFromFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetRequiredFlags(t *testing.T) {
	required := GetRequiredFlags()
	expected := []string{"service-id", "script", "tag-prefix", "interval", "user", "group"}
	if !stringSlicesEqual(required, expected) {
		t.Errorf("GetRequiredFlags() = %v, want %v", required, expected)
	}
}

func TestGetOptionalFlags(t *testing.T) {
	optional := GetOptionalFlags()
	expected := []string{"token", "consul-addr"}
	if !stringSlicesEqual(optional, expected) {
		t.Errorf("GetOptionalFlags() = %v, want %v", optional, expected)
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
