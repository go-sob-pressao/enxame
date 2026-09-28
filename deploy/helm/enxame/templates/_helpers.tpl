{{- define "enxame.rotulos" -}}
app.kubernetes.io/name: enxamed
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/* Os endereços gRPC de todos os nós, para os workers (Cap. 32). */}}
{{- define "enxame.nos" -}}
{{- $nos := list -}}
{{- range $i := until (int .Values.replicas) -}}
{{- $nos = append $nos (printf "%s-%d.%s-nos:7233" $.Release.Name $i $.Release.Name) -}}
{{- end -}}
{{- join "," $nos -}}
{{- end }}
