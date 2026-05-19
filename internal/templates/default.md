# Pocket Casts — Recent Activity

_Generated: {{ .Generated }}_

{{ if .History -}}
## Listening history

{{ range .History -}}
- **{{ if .URL }}[{{ .Title }}]({{ .URL }}){{ else }}{{ .Title }}{{ end }}** — [{{ .PodcastTitle }}]({{ .PodcastURL }}) · played {{ .PlayedUpToFormatted }}{{ if .Published }} · published {{ .PublishedFormatted }}{{ end }}
{{ end }}
{{- end }}
{{ if .Starred -}}
## Starred

{{ range .Starred -}}
- **{{ if .URL }}[{{ .Title }}]({{ .URL }}){{ else }}{{ .Title }}{{ end }}** — [{{ .PodcastTitle }}]({{ .PodcastURL }})
{{ end }}
{{- end }}
