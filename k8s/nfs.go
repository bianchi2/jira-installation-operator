package k8s

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	"github.com/aws/aws-sdk-go/aws"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNfSServerService(jira appv1.Jira, namespace string) (svc corev1.Service) {
	svc = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + "-nfs-server",
			Namespace:       namespace,
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "nfs",
					Protocol: "TCP",
					Port:     2049,
				},
			},
			Selector: map[string]string{
				"app": "nfs-server",
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	return svc
}

func GetNfsServerStatefulSet(jira appv1.Jira, namespace string) (sts appsv1.StatefulSet) {
	sts = appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + "-nfs-server",
			Namespace:       namespace,
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: appsv1.StatefulSetSpec{
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Replicas: aws.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nfs-server",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nfs-server",
					},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: aws.Int64(0),
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "topology.kubernetes.io/zone",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{jira.Spec.AWSRegion + jira.Spec.SharedFS.Ebs.AvailabilityZone},
										},
										{
											Key:      "topology.kubernetes.io/region",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{jira.Spec.AWSRegion},
										},
									},
								},
							},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: jira.Name + "-nfs-server",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "nfs-server",
							Image: "atlassian/nfs-server-test:2.1",
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"DAC_READ_SEARCH", "SYS_RESOURCE"},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "nfs",
									ContainerPort: 2049,
									Protocol:      "TCP",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/srv/nfs",
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"/usr/local/bin/docker-entrypoint.sh", "healthcheck"},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       1,
								FailureThreshold:    30,
							},
						},
					},
				},
			},
			ServiceName: "nfs-server",
		},
	}
	return sts
}
