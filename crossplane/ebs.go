package crossplane

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	"github.com/atlassian-labs/jira-operator/k8s"
	ec2 "github.com/crossplane-contrib/provider-aws/apis/ec2/v1alpha1"
	aws "github.com/crossplane-contrib/provider-aws/pkg/clients"
	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetEbsVolume(jira appv1.Jira) (ebsVolume ec2.Volume) {
	encrypted := false
	if jira.Spec.KMSKeyId != "" {
		encrypted = true
	}

	ebsResourceSpec := xpv1.ResourceSpec{
		ProviderConfigReference: &v1.Reference{
			Name: jira.Spec.CrossplaneAwsProviderName,
		},
	}

	if jira.Spec.RetainOnDelete {
		ebsResourceSpec.DeletionPolicy = "Orphan"
	}

	ebsVolume = ec2.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + "-" + string(jira.UID),
			OwnerReferences: k8s.GetOwnerReferences(jira),
		},
		Spec: ec2.VolumeSpec{
			ResourceSpec: ebsResourceSpec,
			ForProvider: ec2.VolumeParameters{
				Region:           jira.Spec.AWSRegion,
				AvailabilityZone: aws.String(jira.Spec.AWSRegion + jira.Spec.SharedFS.Ebs.AvailabilityZone),
				Encrypted:        &encrypted,
				Size:             &jira.Spec.SharedFS.VolumeSize,
				SnapshotID:       &jira.Spec.SharedFS.Ebs.SnapshotId,
				TagSpecifications: []*ec2.TagSpecification{
					{
						ResourceType: aws.String("volume"),
						Tags:         k8s.GetTags(jira),
					},
				},
				CustomVolumeParameters: ec2.CustomVolumeParameters{
					KMSKeyID: &jira.Spec.KMSKeyId,
				},
			},
		},
	}
	return ebsVolume
}
