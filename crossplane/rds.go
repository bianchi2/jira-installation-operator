package crossplane

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	"github.com/atlassian-labs/jira-operator/k8s"
	database "github.com/crossplane-contrib/provider-aws/apis/database/v1beta1"
	rds "github.com/crossplane-contrib/provider-aws/apis/rds/v1alpha1"
	aws "github.com/crossplane-contrib/provider-aws/pkg/clients"
	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func GetRdsInstance(jira appv1.Jira, dbSubnetGroup database.DBSubnetGroup, dbParameterGroup rds.DBParameterGroup, namespace string) (rdsInstance database.RDSInstance) {

	rdsParams := database.RDSInstanceParameters{
		Region:               &jira.Spec.AWSRegion,
		AllocatedStorage:     &jira.Spec.Database.AllocatedStorage,
		DBInstanceClass:      jira.Spec.Database.DBInstanceClass,
		VPCSecurityGroupIDs:  jira.Spec.Network.SecurityGroupIds,
		DBParameterGroupName: &dbParameterGroup.Name,
		DBSubnetGroupName:    &dbSubnetGroup.Name,
		Engine:               jira.Spec.Database.Engine,
		EngineVersion:        &jira.Spec.Database.EngineVersion,
		KMSKeyID:             &jira.Spec.KMSKeyId,
		MasterUsername:       aws.String("postgres"),
		MasterPasswordSecretRef: &xpv1.SecretKeySelector{
			SecretReference: xpv1.SecretReference{
				Name:      "jira-database-secret",
				Namespace: namespace,
			},
			Key: "password",
		},
		SkipFinalSnapshotBeforeDeletion: aws.Bool(true),
		Tags:                            k8s.GetDbTags(jira),
		ApplyModificationsImmediately:   aws.Bool(true),
	}

	if jira.Spec.Database.SnapshotID != "" {
		restoreFrom := &database.RestoreBackupConfiguration{
			Snapshot: &database.SnapshotRestoreBackupConfiguration{
				SnapshotIdentifier: &jira.Spec.Database.SnapshotID,
			},
			Source: aws.String("Snapshot"),
		}
		rdsParams.RestoreFrom = restoreFrom
	}

	rdsResourceSpec := xpv1.ResourceSpec{
		WriteConnectionSecretToReference: &v1.SecretReference{
			Name:      jira.Name + "-db-secret",
			Namespace: namespace,
		},
		ProviderConfigReference: &v1.Reference{
			Name: jira.Spec.CrossplaneAwsProviderName,
		},
	}

	if jira.Spec.RetainOnDelete {
		rdsResourceSpec.DeletionPolicy = "Orphan"
	}

	rdsInstance = database.RDSInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + "-" + string(jira.UID),
			OwnerReferences: k8s.GetOwnerReferences(jira),
		},
		Spec: database.RDSInstanceSpec{
			ResourceSpec: rdsResourceSpec,
			ForProvider:  rdsParams,
		},
	}
	return rdsInstance
}

func GetDbSubnetGroup(jira appv1.Jira) (dbSubnetGroup database.DBSubnetGroup) {
	return database.DBSubnetGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + "-" + string(jira.UID),
			OwnerReferences: k8s.GetOwnerReferences(jira),
		},
		Spec: database.DBSubnetGroupSpec{
			ForProvider: database.DBSubnetGroupParameters{
				Region:      &jira.Spec.AWSRegion,
				Description: "DB Subnet group for " + jira.Name + " RDS instance",
				SubnetIDs:   jira.Spec.Network.SubnetIDs,
				Tags:        k8s.GetDbTags(jira),
			},
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &v1.Reference{
					Name: jira.Spec.CrossplaneAwsProviderName,
				},
			},
		},
	}
}

func GetDbParameterGroup(jira appv1.Jira) (dbParameterGroup rds.DBParameterGroup) {

	paramaterFamilyVersion := strings.Split(jira.Spec.Database.EngineVersion, ".")[0]
	return rds.DBParameterGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + "-" + string(jira.UID),
			OwnerReferences: k8s.GetOwnerReferences(jira),
		},
		Spec: rds.DBParameterGroupSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &v1.Reference{
					Name: jira.Spec.CrossplaneAwsProviderName,
				},
			},
			ForProvider: rds.DBParameterGroupParameters{
				Region:      jira.Spec.AWSRegion,
				Description: aws.String("DB Parameter Group created by Jira Operator"),
				Tags: []*rds.Tag{
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
				},
				CustomDBParameterGroupParameters: rds.CustomDBParameterGroupParameters{
					Parameters: []rds.CustomParameter{
						{
							ParameterName:  aws.String("log_statement"),
							ParameterValue: aws.String("ddl"),
							ApplyMethod:    aws.String("immediate"),
						},
						{
							ParameterName:  aws.String("log_min_duration_statement"),
							ParameterValue: aws.String("8000"),
							ApplyMethod:    aws.String("immediate"),
						},
						{
							ParameterName:  aws.String("rds.log_retention_period"),
							ParameterValue: aws.String("10080"),
							ApplyMethod:    aws.String("immediate"),
						},
					},
					DBParameterGroupFamily: aws.String("postgres" + paramaterFamilyVersion),
				},
			},
		},
	}
}
