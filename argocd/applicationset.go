package argocd

import (
	"fmt"
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	"os"
	"strconv"
	"strings"
	"text/template"
)

func ProcessApplicationSetTemplate(jira appv1.Jira) (err error) {
	subnets := strings.Join(jira.Spec.Network.SubnetIDs, ",")
	albTags := strings.Join([]string{
		"service_name=" + jira.Name,
		"Name=" + jira.Name,
		"business_unit=WorkplaceTechnology",
		"resource_owner=mgibson",
	}, ",")

	ingressAnnotations := map[string]interface{}{
		"alb.ingress.kubernetes.io/certificate-arn":         "arn:aws:acm:ap-southeast-2:629205377521:certificate/7d398889-d2ed-42e4-94e7-16fded6498f1",
		"alb.ingress.kubernetes.io/healthcheck-path":        "/status",
		"alb.ingress.kubernetes.io/listen-ports":            "[{\"HTTP\": 80}, {\"HTTPS\": 443}]",
		"alb.ingress.kubernetes.io/scheme":                  "internal",
		"alb.ingress.kubernetes.io/ssl-policy":              "ELBSecurityPolicy-FS-1-2-Res-2020-10",
		"alb.ingress.kubernetes.io/subnets":                 subnets,
		"alb.ingress.kubernetes.io/tags":                    albTags,
		"alb.ingress.kubernetes.io/target-group-attributes": "stickiness.enabled=true,stickiness.lb_cookie.duration_seconds=43200",
		"alb.ingress.kubernetes.io/target-type":             "ip",
		"external-dns.alpha.kubernetes.io/hostname":         jira.Spec.Hostname,
	}

	vars := make(map[string]interface{})
	vars["namespace"] = jira.Name
	vars["argoCDNamespace"] = jira.Spec.ArgoCD.Namespace
	vars["argoCDProject"] = jira.Spec.ArgoCD.Project
	vars["autoSync"] = jira.Spec.ArgoCD.SyncPolicy.AutoSync
	vars["retainOnDelete"] = jira.Spec.ArgoCD.RetainOnDelete
	vars["uid"] = jira.UID
	vars["applyOutOfSyncOnly"] = strconv.FormatBool(jira.Spec.ArgoCD.SyncPolicy.ApplyOutOfSyncOnly)
	vars["helmChartRepo"] = jira.Spec.ArgoCD.HelmChart.RepoURL
	vars["helmChartVersion"] = jira.Spec.ArgoCD.HelmChart.Version
	vars["helmValuesGitRepo"] = jira.Spec.ArgoCD.HelmValues.GitRepo
	vars["helmValuesRepoRevision"] = jira.Spec.ArgoCD.HelmValues.GitRevision
	vars["valuesFiles"] = jira.Spec.ArgoCD.HelmValues.HelmValuesFiles
	vars["appHostname"] = jira.Spec.Hostname
	vars["IngressAnnotations"] = ingressAnnotations
	vars["inLineValues"] = jira.Spec.ArgoCD.HelmValues.ValueOverrides

	tmplFile := "argocd/applicationset.yaml.tpl"
	outputFile := fmt.Sprintf("argocd/applicationset-%s.yaml", jira.Name)
	tmplContent, err := os.ReadFile(tmplFile)
	if err != nil {
		return err
	}
	tmpl, err := template.New("applicationset").Funcs(template.FuncMap{
		"split": strings.Split,
	}).Parse(string(tmplContent))
	if err != nil {
		return err
	}
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	err = tmpl.Execute(file, vars)
	if err != nil {
		return err
	}
	return nil
}
