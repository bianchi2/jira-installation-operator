package k8s

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	database "github.com/crossplane-contrib/provider-aws/apis/database/v1beta1"
	ec2 "github.com/crossplane-contrib/provider-aws/apis/ec2/v1alpha1"
	aws "github.com/crossplane-contrib/provider-aws/pkg/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/rand"
	"time"
)

type Tag struct {
	Key   string
	Value string
}

func GetTags(jira appv1.Jira) (resourceTags []*ec2.Tag) {
	resourceTags = []*ec2.Tag{
		{
			Key:   aws.String("resource_owner"),
			Value: aws.String("mgibson"),
		},
		{
			Key:   aws.String("service_name"),
			Value: aws.String("itplat_infrastructure"),
		},
		{
			Key:   aws.String("created_by"),
			Value: aws.String("jira_operator"),
		},
		{
			Key:   aws.String("business_unit"),
			Value: aws.String("Workplace Technology"),
		},
		{
			Key:   aws.String("Name"),
			Value: aws.String(jira.Name + "-" + string(jira.UID)),
		},
	}
	return resourceTags
}

func GetOwnerReferences(jira appv1.Jira) (ownerReferences []metav1.OwnerReference) {
	blockOwnerDeletion := true
	ownerReferences = []metav1.OwnerReference{
		{
			APIVersion:         jira.APIVersion,
			Kind:               jira.Kind,
			Name:               jira.Name,
			UID:                jira.UID,
			BlockOwnerDeletion: &blockOwnerDeletion,
		},
	}
	return ownerReferences
}

func GetDbTags(jira appv1.Jira) []database.Tag {
	return []database.Tag{
		{
			Key:   "resource_owner",
			Value: "mgibson",
		},
		{
			Key:   "service_name",
			Value: "itplat_infrastructure",
		},
		{
			Key:   "created_by",
			Value: "jira_operator",
		},
		{
			Key:   "business_unit",
			Value: "Workplace Technology",
		},
		{
			Key:   "Name",
			Value: jira.Name + "-" + string(jira.UID),
		},
	}
}

func GeneratePasswd(stringLength int) (passwd string) {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyz" + "%()$#" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "0123456789")
	length := stringLength
	buf := make([]rune, length)
	for i := range buf {
		buf[i] = chars[rand.Intn(len(chars))]
	}
	passwd = string(buf)
	return passwd
}
