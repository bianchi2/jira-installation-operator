package crossplane

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	"github.com/atlassian-labs/jira-operator/k8s"
	efs "github.com/crossplane-contrib/provider-aws/apis/efs/v1alpha1"
	aws "github.com/crossplane-contrib/provider-aws/pkg/clients"
	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetMountTargets(jira appv1.Jira, filesystemId string, subnetId string, index string) (mountTarget efs.MountTarget) {
	mountTargetResourceSpec := xpv1.ResourceSpec{
		ProviderConfigReference: &v1.Reference{
			Name: jira.Spec.CrossplaneAwsProviderName,
		},
	}
	if jira.Spec.RetainOnDelete {
		mountTargetResourceSpec.DeletionPolicy = "Orphan"
	}
	mountTarget = efs.MountTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + index + "-" + string(jira.UID),
			OwnerReferences: k8s.GetOwnerReferences(jira),
		},
		Spec: efs.MountTargetSpec{
			ResourceSpec: mountTargetResourceSpec,
			ForProvider: efs.MountTargetParameters{
				Region: jira.Spec.AWSRegion,
				CustomMountTargetParameters: efs.CustomMountTargetParameters{
					SecurityGroups: jira.Spec.Network.SecurityGroupIds,
					FileSystemID:   &filesystemId,
					SubnetID:       &subnetId,
				},
			},
		},
	}
	return mountTarget
}

func GetFileSystem(jira appv1.Jira, namespace string) (sharedFileSystem efs.FileSystem) {
	efsResourceSpec := xpv1.ResourceSpec{
		WriteConnectionSecretToReference: &v1.SecretReference{
			Name:      jira.Name + "-efs-secret",
			Namespace: namespace,
		},
		ProviderConfigReference: &v1.Reference{
			Name: jira.Spec.CrossplaneAwsProviderName,
		},
	}

	if jira.Spec.RetainOnDelete {
		efsResourceSpec.DeletionPolicy = "Orphan"
	}

	sharedFileSystem = efs.FileSystem{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + "-" + string(jira.UID),
			OwnerReferences: k8s.GetOwnerReferences(jira),
		},
		Spec: efs.FileSystemSpec{
			ResourceSpec: efsResourceSpec,
			ForProvider: efs.FileSystemParameters{
				Region:    jira.Spec.AWSRegion,
				KMSKeyID:  &jira.Spec.KMSKeyId,
				Encrypted: aws.Bool(true),
				Tags: []*efs.Tag{
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
				},
			},
		},
	}
	return sharedFileSystem
}
