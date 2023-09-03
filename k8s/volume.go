package k8s

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNfsPersistentVolume(jira appv1.Jira, nfsClusterIp string, size string, namespace string) (pv corev1.PersistentVolume) {
	pv = corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "jira-shared-home-pv",
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(size + "Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server: nfsClusterIp,
					Path:   "/srv/nfs",
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			ClaimRef: &corev1.ObjectReference{
				Kind:      "PersistentVolumeClaim",
				Namespace: namespace,
				Name:      "jira-shared-home",
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              jira.Spec.SharedFS.EbsStorageClassName,
		},
	}
	return pv
}

func GetEfsPersistentVolume(jira appv1.Jira, efsId string, namespace string) (pv corev1.PersistentVolume) {
	pv = corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "jira-shared-home-pv",
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("10Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:       jira.Spec.SharedFS.EfsCsiDriverName,
					VolumeHandle: efsId,
					ReadOnly:     false,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			ClaimRef: &corev1.ObjectReference{
				Kind:      "PersistentVolumeClaim",
				Namespace: namespace,
				Name:      "jira-shared-home",
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              jira.Spec.SharedFS.EfsStorageClassName,
		},
	}
	return pv
}

func GetEbsPersistentVolume(jira appv1.Jira, ebsVolumeId string, name string, size string, namespace string) (pv corev1.PersistentVolume) {
	pv = corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(size + "Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				AWSElasticBlockStore: &corev1.AWSElasticBlockStoreVolumeSource{
					VolumeID: ebsVolumeId,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			ClaimRef: &corev1.ObjectReference{
				Kind:      "PersistentVolumeClaim",
				Namespace: namespace,
				Name:      name,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              jira.Spec.SharedFS.EbsStorageClassName,
			NodeAffinity: &corev1.VolumeNodeAffinity{Required: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "topology.kubernetes.io/zone",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{jira.Spec.AWSRegion + jira.Spec.SharedFS.NfsServerAvailabilityZone},
						},
						{
							Key:      "topology.kubernetes.io/region",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{jira.Spec.AWSRegion},
						},
					},
				},
			}}},
		},
	}
	return pv
}

func GetPersistentVolumeClaim(jira appv1.Jira, name string, namespace string, volumeName string, storageClassName string, size string, accessMode corev1.PersistentVolumeAccessMode) (pvc corev1.PersistentVolumeClaim) {
	pvc = corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size + "Gi"),
				},
			},
			VolumeName:       volumeName,
			StorageClassName: &storageClassName,
		},
	}
	return pvc
}
