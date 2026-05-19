# Pocket Casts — Recent Activity

_Generated: {{ .Generated }}_

{{ if .History -}}
## Listening history

{{ range .History -}}
- **{{ if .URL }}[{{ .Title }}]({{ .URL }}){{ else }}{{ .Title }}{{ end }}**{{ if .PodcastTitle }} — [{{ .PodcastTitle }}]({{ .PodcastURL }}){{ end }}{{ if .PlayedUpToFormatted }} · played {{ .PlayedUpToFormatted }}{{ end }}{{ if .PublishedFormatted }} · published {{ .PublishedFormatted }}{{ end }}
{{ end }}
{{- end }}
{{ if .Starred -}}
## Starred

{{ range .Starred -}}
- **{{ if .URL }}[{{ .Title }}]({{ .URL }}){{ else }}{{ .Title }}{{ end }}**{{ if .PodcastTitle }} — [{{ .PodcastTitle }}]({{ .PodcastURL }}){{ end }}{{ if .PublishedFormatted }} · published {{ .PublishedFormatted }}{{ end }}
{{ end }}
{{- end }}
