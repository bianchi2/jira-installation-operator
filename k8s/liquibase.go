package k8s

import (
	"fmt"
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	aws "github.com/crossplane-contrib/provider-aws/pkg/clients"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func GetLiquibaseSecret(jira appv1.Jira, namespace, rdsHostname string, masterPassword []byte, appUserJDBCPassword []byte) (liquibaseSecret corev1.Secret) {
	liquibaseSecretData := map[string][]byte{
		"password":                masterPassword,
		"username":                []byte("postgres"),
		"url":                     []byte("jdbc:postgresql://" + rdsHostname + "/postgres"),
		"hostname":                []byte(rdsHostname),
		"changeLogFile":           []byte("changelog.yml"),
		"classpath":               []byte("changelog"),
		"parameter.appUsername":   []byte("jira"),
		"parameter.appRoUsername": []byte("jira-ro"),
		"parameter.appPassword":   appUserJDBCPassword,
		"parameter.appRoPassword": []byte(GeneratePasswd(26)),
	}

	liquibaseSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jira.Name + "-liquibase-properties-secret",
			Namespace: namespace,
		},
		Data: liquibaseSecretData,
	}
	return liquibaseSecret
}

func GetLiquibaseConfigMap(jira appv1.Jira, namespace string) (liquibaseConfigMap corev1.ConfigMap, err error) {
	liquibaseFilePath := "config/liquibase/changelog.yml"
	configContent, err := os.ReadFile(liquibaseFilePath)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	liquibaseConfigMap = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jira.Name + "-liquibase-changelog",
			Namespace: namespace,
		},
		Data: map[string]string{
			"changelog.yml": string(configContent),
		},
	}
	return liquibaseConfigMap, nil
}

func GetServiceAccount(jira appv1.Jira, namespace string) (serviceAccout corev1.ServiceAccount) {
	serviceAccout = corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jira.Name + "-rds-reset-sa",
			Namespace: namespace,
			Annotations: map[string]string{
				"eks.amazonaws.com/role-arn": "arn:aws:iam::629205377521:role/crossplane",
			},
		},
	}
	return serviceAccout
}

func GetChangeRootPasswordJob(jira appv1.Jira, namespace string, dbInstanceIdentifier string) (changeRootPasswordJob batchv1.Job) {
	awsCliCommand := fmt.Sprintf("aws rds modify-db-instance --db-instance-identifier=%s --master-user-password $PGPASSWORD --region %s --apply-immediately", dbInstanceIdentifier, jira.Spec.AWSRegion)
	changeRootPasswordJob = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jira.Name + "-reset-rds-credentials",
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: aws.Int32(20),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"owner": jira.Name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: jira.Name + "-rds-reset-sa",
					Containers: []corev1.Container{{
						Name:    "reset-creds",
						Image:   "amazon/aws-cli:2.13.14",
						Command: []string{"/bin/sh"},
						Args:    []string{"-c", awsCliCommand},
						Env: []corev1.EnvVar{
							{
								Name: "PGPASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: jira.Name + "-rds-master-password",
										},
										Key: "password",
									},
								},
							},
						},
					}},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	return changeRootPasswordJob
}

func GetLiquibaseJob(jira appv1.Jira, namespace string) (liquibaseJob batchv1.Job) {
	liquibaseJob = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jira.Name + "-liquibase-changeset",
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"owner": jira.Name,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "liquibase-properties-secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: jira.Name + "-liquibase-properties-secret",
								},
							},
						},
						{
							Name: "liquibase-changelog-configmap",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: jira.Name + "-liquibase-changelog",
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    jira.Name + "-liquibase",
							Image:   "liquibase/liquibase:4.21.0",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{"cd /liquibase/changelog/properties; grep '' * | sed 's/:/: /1' > /liquibase/liquibase.properties; cd /liquibase;  ./docker-entrypoint.sh --defaultsFile=liquibase.properties update;"},
							Env: []corev1.EnvVar{
								{
									Name: "PGPASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: jira.Name + "-liquibase-properties-secret",
											},
											Key: "password",
										},
									},
								},
								{
									Name: "JDBC_URL",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: jira.Name + "-liquibase-properties-secret",
											},
											Key: "url",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "liquibase-properties-secret",
									MountPath: "/liquibase/changelog/properties",
								},
								{
									Name:      "liquibase-changelog-configmap",
									MountPath: "/liquibase/changelog/changelog.yml",
									SubPath:   "changelog.yml",
								},
							},
						},
					},
					RestartPolicy: "Never",
				},
			},
		},
	}
	return liquibaseJob
}
