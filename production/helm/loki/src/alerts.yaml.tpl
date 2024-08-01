---
groups:
  - name: "loki_alerts"
    rules:
{{- with .Values.monitoring.rules }}
{{- if not (.disabled.LokiRequestErrors | default (not .configs.LokiRequestErrors.enabled)) }}
      {{- with .configs.LokiRequestErrors }}
      - alert: "LokiRequestErrors"
        annotations:
          message: |
            {{`{{`}} $labels.job {{`}}`}} {{`{{`}} $labels.route {{`}}`}} is experiencing {{`{{`}} printf "%.2f" $value {{`}}`}}% errors.
        expr: |
          100 * sum(rate(loki_request_duration_seconds_count{status_code=~"5.."}[{{ .lookbackPeriod }}])) by (namespace, job, route)
            /
          sum(rate(loki_request_duration_seconds_count[{{ .lookbackPeriod }}])) by (namespace, job, route)
            > {{ .threshold }}
        for: {{ .for }}
        labels:
          severity: {{ .severity }}
      {{- end }}
{{- if .additionalRuleLabels }}
{{ toYaml .additionalRuleLabels | indent 10 }}
{{- end }}
{{- end }}
{{- if not (.disabled.LokiRequestPanics | default (not .configs.LokiRequestPanics.enabled)) }}
      {{- with .configs.LokiRequestPanics }}
      - alert: "LokiRequestPanics"
        annotations:
          message: |
            {{`{{`}} $labels.job {{`}}`}} is experiencing {{`{{`}} printf "%.2f" $value {{`}}`}}% increase of panics.
        expr: |
          sum(increase(loki_panic_total[{{ .lookbackPeriod }}])) by (namespace, job) > {{ .threshold }}
        labels:
          severity: {{ .severity }}
      {{- end }}
{{- if .additionalRuleLabels }}
{{ toYaml .additionalRuleLabels | indent 10 }}
{{- end }}
{{- end }}
{{- if not (.disabled.LokiRequestLatency | default (not .configs.LokiRequestLatency.enabled)) }}
      {{- with .configs.LokiRequestLatency }}
      - alert: "LokiRequestLatency"
        annotations:
          message: |
            {{`{{`}} $labels.job {{`}}`}} {{`{{`}} $labels.route {{`}}`}} is experiencing {{`{{`}} printf "%.2f" $value {{`}}`}}s 99th percentile latency.
        expr: |
          namespace_job_route:loki_request_duration_seconds:99quantile{route!~"(?i).*tail.*"} > {{ .threshold }}
        for: {{ .for }}
        labels:
          severity: {{ .severity }}
      {{- end }}
{{- if .additionalRuleLabels }}
{{ toYaml .additionalRuleLabels | indent 10 }}
{{- end }}
{{- end }}
{{- if not (.disabled.LokiTooManyCompactorsRunning | default (not .configs.LokiTooManyCompactorsRunning.enabled)) }}
      {{- with .configs.LokiTooManyCompactorsRunning }}
      - alert: "LokiTooManyCompactorsRunning"
        annotations:
          message: |
            {{`{{`}} $labels.cluster {{`}}`}} {{`{{`}} $labels.namespace {{`}}`}} has had {{`{{`}} printf "%.0f" $value {{`}}`}} compactors running for more than 5m. Only one compactor should run at a time.
        expr: |
          sum(loki_boltdb_shipper_compactor_running) by (namespace, cluster) > 1
        for: {{ .for }}
        labels:
          severity: {{ .severity }}
      {{- end }}
{{- if .additionalRuleLabels }}
{{ toYaml .additionalRuleLabels | indent 10 }}
{{- end }}
{{- end }}
{{- if not (.disabled.LokiCanaryLatency | default (not .configs.LokiCanaryLatency.enabled)) }}
  {{- with .configs.LokiCanaryLatency }}
  - name: "loki_canaries_alerts"
    rules:
      - alert: "LokiCanaryLatency"
        annotations:
          message: |
            {{`{{`}} $labels.job {{`}}`}} is experiencing {{`{{`}} printf "%.2f" $value {{`}}`}}s 99th percentile latency.
        expr: |
          histogram_quantile(0.99, sum(rate(loki_canary_response_latency_seconds_bucket[5m])) by (le, namespace, job)) > {{ .threshold }}
        for: {{ .for }}
        labels:
          severity: {{ .severity }}
  {{- end }}
{{- if .additionalRuleLabels }}
{{ toYaml .additionalRuleLabels | indent 10 }}
{{- end }}
{{- end }}
{{- end }}
