package systemd

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

const (
	templateName     = "serviceTemplate"
	templateContents = `
[Unit]
Description=Tagit {{ .ServiceID }}
After=network.target
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/tagit run -s {{ .ServiceID }} -x {{ .Script }} -p {{ .TagPrefix }} -i {{ .Interval }}{{ if .Token }} -t {{ .Token }}{{ end }}{{ if .ConsulAddr }} -c {{ .ConsulAddr }}{{ end }}
Environment=HOME=/var/run/tagit/{{ .ServiceID }}
Restart=always
User={{ .User }}
Group={{ .Group }}

[Install]
WantedBy=multi-user.target
`
)

// Fields is the struct that holds the fields for the systemd service.
type Fields struct {
	ServiceID  string
	Script     string
	TagPrefix  string
	Interval   string
	Token      string
	ConsulAddr string
	User       string
	Group      string
}

var parsedTemplate *template.Template

func init() {
	var err error
	parsedTemplate, err = template.New(templateName).Parse(templateContents)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse template: %v", err))
	}
}

// RenderTemplate renders the template for the systemd service.
func RenderTemplate(fields *Fields) (string, error) {
	if err := validateFields(fields); err != nil {
		return "", fmt.Errorf("field validation failed: %w", err)
	}

	var tmplBuffer bytes.Buffer
	err := parsedTemplate.Execute(&tmplBuffer, fields)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return tmplBuffer.String(), nil
}

func validateFields(fields *Fields) error {
	var missingFields []string

	if fields.ServiceID == "" {
		missingFields = append(missingFields, "ServiceID")
	}
	if fields.Script == "" {
		missingFields = append(missingFields, "Script")
	}
	if fields.TagPrefix == "" {
		missingFields = append(missingFields, "TagPrefix")
	}
	if fields.Interval == "" {
		missingFields = append(missingFields, "Interval")
	}
	if fields.User == "" {
		missingFields = append(missingFields, "User")
	}
	if fields.Group == "" {
		missingFields = append(missingFields, "Group")
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missingFields, ", "))
	}

	return nil
}

// NewFieldsFromFlags creates a new Fields struct from command line flags.
func NewFieldsFromFlags(flags map[string]string) (*Fields, error) {
	fields := &Fields{
		ServiceID:  flags["service-id"],
		Script:     flags["script"],
		TagPrefix:  flags["tag-prefix"],
		Interval:   flags["interval"],
		Token:      flags["token"],
		ConsulAddr: flags["consul-addr"],
		User:       flags["user"],
		Group:      flags["group"],
	}

	if err := validateFields(fields); err != nil {
		return nil, err
	}

	return fields, nil
}

// GetRequiredFlags returns a list of required flag names.
func GetRequiredFlags() []string {
	return []string{"service-id", "script", "tag-prefix", "interval", "user", "group"}
}

// GetOptionalFlags returns a list of optional flag names.
func GetOptionalFlags() []string {
	return []string{"token", "consul-addr"}
}
