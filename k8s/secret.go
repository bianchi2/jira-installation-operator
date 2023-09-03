package k8s

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetRdsMasterSecret(jira appv1.Jira, namespace string) (rdsMasterPasswordSecret corev1.Secret) {
	rdsMasterPasswordData := map[string][]byte{
		"password": []byte(GeneratePasswd(26)),
	}
	rdsMasterPasswordSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jira.Name + "-rds-master-password",
			Namespace: namespace,
		},
		Data: rdsMasterPasswordData,
	}
	return rdsMasterPasswordSecret
}

func GetJiraUserRdsSecret(jira appv1.Jira, namespace, rdsHostname string) (databaseSecret corev1.Secret) {
	jiraRdsSecretData := map[string][]byte{
		"password": []byte(GeneratePasswd(26)),
		"username": []byte("jira"),
		"jdbcUrl":  []byte("jdbc:postgresql://" + rdsHostname + "/jira"),
	}
	databaseSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jira-database-secret",
			Namespace: namespace,
		},
		Data: jiraRdsSecretData,
	}
	return databaseSecret
}
