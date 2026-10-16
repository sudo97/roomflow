{{- define "roomflow.name" -}}{{ .Release.Name }}{{- end }}

{{- define "roomflow.labels" -}}
app.kubernetes.io/name: roomflow
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end }}

{{- define "roomflow.selector" -}}
app.kubernetes.io/name: roomflow
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
