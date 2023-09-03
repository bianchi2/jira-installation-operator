package controllers

import (
	"context"
	database "github.com/crossplane-contrib/provider-aws/apis/database/v1beta1"
	ec2 "github.com/crossplane-contrib/provider-aws/apis/ec2/v1alpha1"
	efs "github.com/crossplane-contrib/provider-aws/apis/efs/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *JiraReconciler) getRdsStatus(rdsInstance database.RDSInstance, rdsObjKey client.ObjectKey) (status string, err error) {
	err = r.Get(context.TODO(), rdsObjKey, &rdsInstance)
	if err != nil {
		return "", err
	}
	status = rdsInstance.Status.AtProvider.DBInstanceStatus
	return status, nil
}

func (r *JiraReconciler) getEbsVolumeId(volume ec2.Volume, volumeObjKey client.ObjectKey) (volumeId string, err error) {
	err = r.Get(context.TODO(), volumeObjKey, &volume)
	if err != nil {
		return "", err
	}
	if volume.Status.AtProvider.VolumeID != nil {
		volumeId = *volume.Status.AtProvider.VolumeID
		return volumeId, nil
	}
	return "", nil
}

func (r *JiraReconciler) getMountTargetStatus(mountTarget efs.MountTarget, rdsObjKey client.ObjectKey) (status *string, err error) {
	err = r.Get(context.TODO(), rdsObjKey, &mountTarget)
	if err != nil {
		return nil, err
	}
	status = mountTarget.Status.AtProvider.LifeCycleState
	return status, nil
}

func (r *JiraReconciler) getFilesystemId(fileSystem efs.FileSystem, rdsObjKey client.ObjectKey) (id *string, err error) {
	err = r.Get(context.TODO(), rdsObjKey, &fileSystem)
	if err != nil {
		return nil, err
	}
	id = fileSystem.Status.AtProvider.FileSystemID
	return id, nil
}

func (r *JiraReconciler) getRdsEndpoint(rdsInstance database.RDSInstance, rdsObjKey client.ObjectKey) (endpoint string, err error) {
	err = r.Get(context.TODO(), rdsObjKey, &rdsInstance)
	if err != nil {
		return "", err
	}
	endpoint = rdsInstance.Status.AtProvider.Endpoint.Address
	return endpoint, nil
}

func (r *JiraReconciler) getPodStatus(pod corev1.Pod, podName string, namespace string) (status string, err error) {
	err = r.Get(context.TODO(), client.ObjectKey{Name: podName, Namespace: namespace}, &pod)
	if err != nil {
		return "", err
	}
	status = string(pod.Status.Phase)
	return status, nil
}

func (r *JiraReconciler) getJobSucceededReplicas(job batchv1.Job, jobName string, namespace string) (replicas int32, err error) {
	err = r.Get(context.TODO(), client.ObjectKey{Name: jobName, Namespace: namespace}, &job)
	if err != nil {
		return replicas, err
	}
	replicas = job.Status.Succeeded
	return replicas, nil
}

func (r *JiraReconciler) getSvcClusterIp(svc corev1.Service, name string, namespace string) (ip string, err error) {
	err = r.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: namespace}, &svc)
	if err != nil {
		return ip, err
	}
	ip = svc.Spec.ClusterIP
	return ip, nil
}

func (r *JiraReconciler) getStsReadyReplicas(sts appsv1.StatefulSet, name string, namespace string) (readyReplicas int32, err error) {
	err = r.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: namespace}, &sts)
	if err != nil {
		return 0, err
	}
	readyReplicas = sts.Status.ReadyReplicas
	return readyReplicas, nil
}
