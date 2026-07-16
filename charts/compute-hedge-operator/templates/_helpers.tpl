{{- define "chp.name" -}}
{{- default "compute-hedge-operator" .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "chp.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default "compute-hedge-operator" .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "chp.labels" -}}
app.kubernetes.io/name: {{ include "chp.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "chp.selectorLabels" -}}
app: {{ include "chp.fullname" . }}
{{- end -}}
