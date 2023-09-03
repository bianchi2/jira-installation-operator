package k8s

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNamespace(jira appv1.Jira) (namespace corev1.Namespace) {
	namespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name,
			Labels:          map[string]string{"owned_by": jira.Name},
			Annotations:     map[string]string{"owned_by": jira.Name},
			OwnerReferences: GetOwnerReferences(jira),
		},
	}
	return namespace
}
