apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: {{ .namespace }}
  namespace: {{ .argoCDNamespace }}
{{- if not .retainOnDelete }}
  ownerReferences:
  - apiVersion: app.atlassian.com/v1
    blockOwnerDeletion: true
    controller: true
    kind: Jira
    name: {{ .namespace }}
    uid: {{ .uid }}
{{- end }}
spec:
  generators:
  - list:
      elements:
      - namespace: {{ .namespace }}
  template:
    metadata:
      name: '{{"{{ namespace }}"}}'
    spec:
      project: {{ .argoCDProject }}
      syncPolicy:
        syncOptions:
        - ApplyOutOfSyncOnly={{ .applyOutOfSyncOnly }}
{{- if .autoSync }}
        automated: {}
{{- end }}
      sources:
        - chart: jira
          repoURL: {{ .helmChartRepo }}
          targetRevision: {{ .helmChartVersion }}
          helm:
            releaseName: '{{"{{ namespace }}"}}'
            values: |
            {{- range $line := split .inLineValues "\n" }}
                {{ $line }}
            {{- end -}}
              ingress:
                    host: {{ .appHostname }}
                    annotations:
                        {{- range $key, $value := .IngressAnnotations }}
                        '{{ $key }}': '{{ $value }}'
                        {{- end }}
            valueFiles:
                {{- range .valuesFiles }}
                - {{ . }}
                {{- end }}
        - repoURL: {{ .helmValuesGitRepo }}
          targetRevision: {{ .helmValuesRepoRevision }}
          ref: values
      destination:
        server: "https://kubernetes.default.svc"
        namespace: '{{"{{ namespace }}"}}'