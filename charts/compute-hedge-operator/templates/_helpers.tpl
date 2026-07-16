{{- define "chp.name" -}}
compute-hedge-operator
{{- end -}}

{{- define "chp.labels" -}}
app.kubernetes.io/name: {{ include "chp.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "chp.selectorLabels" -}}
app: {{ include "chp.name" . }}
{{- end -}}
