package systemd

import (
	"bytes"
	"text/template"
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

// serviceTemplate is the template for the systemd service.
var serviceTemplate = `
[Unit]
Description=Tagit {{ .ServiceID }}
After=network.target
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/tagit run -s {{ .ServiceID }} -x {{ .Script }} -p {{ .TagPrefix }} -i {{ .Interval }}{{- if .Token }} -t {{ .Token }}{{- end }}{{ if .ConsulAddr }} -c {{ .ConsulAddr }}{{- end }}
Environment=HOME=/var/run/taggit/{{ .ServiceID }}
Restart=always
User={{ .User }}
Group={{ .Group }}

[Install]
WantedBy=multi-user.target
`

// RenderTemplate renders the template for the systemd service.
func RenderTemplate(fields *Fields) (string, error) {
	tmpl, err := template.New("serviceTemplate").Parse(serviceTemplate)
	if err != nil {
		return "", err
	}

	var tmplBuffer bytes.Buffer
	err = tmpl.Execute(&tmplBuffer, fields)
	if err != nil {
		return "", err
	}

	return tmplBuffer.String(), nil
}
