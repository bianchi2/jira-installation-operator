package k8s

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetRdsSecret(jira appv1.Jira, rdsHostname string, namespace string) (rdsMasterPasswordSecret corev1.Secret) {
	secretData := map[string][]byte{
		"password":                []byte(GeneratePasswd(26)),
		"username":                []byte("postgres"),
		"url":                     []byte("jdbc:postgresql://" + rdsHostname + "/postgres"),
		"jdbcUrl":                 []byte("jdbc:postgresql://" + rdsHostname + "/jira"),
		"hostname":                []byte(rdsHostname),
		"changeLogFile":           []byte("changelog.yml"),
		"classpath":               []byte("changelog"),
		"parameter.appUsername":   []byte("jira"),
		"parameter.appRoUsername": []byte("jira-ro"),
		"parameter.appPassword":   []byte(GeneratePasswd(26)),
		"parameter.appRoPassword": []byte(GeneratePasswd(26)),
	}
	rdsMasterPasswordSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "jira-database-secret",
			Namespace:       namespace,
			OwnerReferences: GetOwnerReferences(jira),
		},
		Data: secretData,
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
			Name:            "jira-database-secret",
			Namespace:       namespace,
			OwnerReferences: GetOwnerReferences(jira),
		},
		Data: jiraRdsSecretData,
	}
	return databaseSecret
}
