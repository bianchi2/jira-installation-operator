package k8s

import (
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	aws "github.com/crossplane-contrib/provider-aws/pkg/clients"
	snapshot "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNfsPersistentVolume(jira appv1.Jira, nfsClusterIp string, size string, namespace string) (pv corev1.PersistentVolume) {
	pv = corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "jira-shared-home-pv" + "-" + string(jira.UID),
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
			StorageClassName:              jira.Spec.SharedFS.Ebs.EbsStorageClassName,
		},
	}
	return pv
}

func GetEfsPersistentVolume(jira appv1.Jira, efsId string, namespace string) (pv corev1.PersistentVolume) {
	pv = corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "jira-shared-home-pv" + "-" + string(jira.UID),
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("10Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:       jira.Spec.SharedFS.Efs.EfsCsiDriverName,
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
			StorageClassName:              jira.Spec.SharedFS.Efs.EfsStorageClassName,
		},
	}
	return pv
}

func GetEbsPersistentVolume(jira appv1.Jira, ebsVolumeId string, name string, size string, namespace string) (pv corev1.PersistentVolume) {
	pv = corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name + "-" + string(jira.UID),
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(size + "Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				AWSElasticBlockStore: &corev1.AWSElasticBlockStoreVolumeSource{
					VolumeID: ebsVolumeId,
					FSType:   jira.Spec.SharedFS.Ebs.EbsFsType,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			ClaimRef: &corev1.ObjectReference{
				Kind:      "PersistentVolumeClaim",
				Namespace: namespace,
				Name:      name,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              jira.Spec.SharedFS.Ebs.EbsStorageClassName,
			NodeAffinity: &corev1.VolumeNodeAffinity{Required: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{
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

func GetFsxVolumeSnapshotContent(jira appv1.Jira, namespace string) (snapshotContent snapshot.VolumeSnapshotContent) {
	snapshotContent = snapshot.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + "-" + string(jira.UID),
			Namespace:       namespace,
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: snapshot.VolumeSnapshotContentSpec{
			VolumeSnapshotRef: corev1.ObjectReference{
				Kind:      "VolumeSnapshot",
				Namespace: namespace,
				Name:      jira.Name + "-" + string(jira.UID),
			},
			Driver:                  jira.Spec.SharedFS.Fsx.FsxCsiDriverName,
			VolumeSnapshotClassName: &jira.Spec.SharedFS.Fsx.FsxVolumeSnapshotClassName,
			Source: snapshot.VolumeSnapshotContentSource{
				SnapshotHandle: &jira.Spec.SharedFS.Fsx.SnapshotId,
			},
			DeletionPolicy: snapshot.VolumeSnapshotContentRetain,
		},
	}
	return snapshotContent
}

func GetFsxVolumeSnapshot(jira appv1.Jira, namespace string) (volumeSnapshot snapshot.VolumeSnapshot) {
	volumeSnapshot = snapshot.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jira.Name + "-" + string(jira.UID),
			Namespace:       namespace,
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: snapshot.VolumeSnapshotSpec{
			Source: snapshot.VolumeSnapshotSource{
				VolumeSnapshotContentName: aws.String(jira.Name + "-" + string(jira.UID)),
			},
			VolumeSnapshotClassName: &jira.Spec.SharedFS.Fsx.FsxVolumeSnapshotClassName,
		},
	}
	return volumeSnapshot
}

func GetFsxPersistentVolumeClaimFromSnapshot(jira appv1.Jira, name string, namespace string, size string) (pvc corev1.PersistentVolumeClaim) {
	pvc = corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: GetOwnerReferences(jira),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size + "Gi"),
				},
			},
			StorageClassName: &jira.Spec.SharedFS.Fsx.FsxRestoreStorageClassName,
			DataSource: &corev1.TypedLocalObjectReference{
				APIGroup: aws.String("snapshot.storage.k8s.io"),
				Kind:     "VolumeSnapshot",
				Name:     jira.Name + "-" + string(jira.UID),
			},
		},
	}
	return pvc
}
