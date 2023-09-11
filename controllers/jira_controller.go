/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	appv1 "github.com/atlassian-labs/jira-operator/api/v1"
	"github.com/atlassian-labs/jira-operator/argocd"
	"github.com/atlassian-labs/jira-operator/crossplane"
	"github.com/atlassian-labs/jira-operator/k8s"
	database "github.com/crossplane-contrib/provider-aws/apis/database/v1beta1"
	ec2 "github.com/crossplane-contrib/provider-aws/apis/ec2/v1alpha1"
	efs "github.com/crossplane-contrib/provider-aws/apis/efs/v1alpha1"
	rds "github.com/crossplane-contrib/provider-aws/apis/rds/v1alpha1"
	snapshot "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"time"
)

// JiraReconciler reconciles a Jira object
type JiraReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.atlassian.com,resources=jiras,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.atlassian.com,resources=jiras/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.atlassian.com,resources=jiras/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *JiraReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	jira := &appv1.Jira{}
	err := r.Get(ctx, req.NamespacedName, jira)
	if err != nil {
		logger.Info("Failed to get custom resource")
		return ctrl.Result{}, nil
	}

	// create namespace
	namespace := k8s.GetNamespace(*jira)
	err = r.Create(context.TODO(), &namespace)
	if err != nil && !errors.IsAlreadyExists(err) {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// create DBParameterGroup
	dbParameterGroup := crossplane.GetDbParameterGroup(*jira)
	err = r.Create(context.TODO(), &dbParameterGroup)
	if err != nil && !errors.IsAlreadyExists(err) {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// create DBSubnetGroup
	dbSubnetGroup := crossplane.GetDbSubnetGroup(*jira)
	err = r.Create(context.TODO(), &dbSubnetGroup)
	if err != nil && !errors.IsAlreadyExists(err) {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// create database secret which crossplane, liquibase and Jira will use
	rdsSecret := k8s.GetRdsSecret(*jira, "replaceme", namespace.Name)
	err = r.Create(context.TODO(), &rdsSecret)

	if err != nil && !errors.IsAlreadyExists(err) {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// create RDS instance
	rdsInstance := crossplane.GetRdsInstance(*jira, dbSubnetGroup, dbParameterGroup, namespace.Name)
	err = r.Create(context.TODO(), &rdsInstance)
	if err != nil && !errors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}

	// get RDS status
	rdsObjKey := client.ObjectKey{
		Name: jira.Name + "-" + string(jira.UID),
	}
	rdsStatus, err := r.getRdsStatus(rdsInstance, rdsObjKey)
	if err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// get current RDS status from custom resource and update it if it differs from the one in crossplane resource status
	currentCRStatus := jira.Status.RDS.Status
	if currentCRStatus != rdsStatus {
		jira.Status.RDS.Status = rdsStatus
		err = r.Status().Update(context.TODO(), jira)
		logger.Info("Updating RDS status to: " + rdsStatus)
		if err != nil {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
	}

	// to proceed RDS status must be available, let's check again in 30 seconds
	if rdsStatus != "available" {
		logger.Info("Waiting for RDS available status: " + rdsInstance.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// get RDS hostname and update custom resource status with it
	rdsHostname, err := r.getRdsEndpoint(rdsInstance, rdsObjKey)
	if err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// RDS is being provisioned, requeue in 10 seconds
	// we expect endpoint to be there because the status should be available
	if rdsHostname == "" {
		logger.Info("Waiting for RDS to be available")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// at this point we should have RDS endpoint, let's update database secret and Jira custom resource status with it
	existingRdsSecretData, err := r.getSecretData(rdsSecret)
	rdsHostnameInSecret := string(existingRdsSecretData["hostname"])
	if rdsHostnameInSecret != rdsHostname {
		rdsSecret := k8s.GetRdsSecret(*jira, rdsHostname, namespace.Name)
		logger.Info("Updating RDS hostname in jira-database-secret:" + rdsHostname)
		err = r.Client.Update(context.TODO(), &rdsSecret)
		if err != nil {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
	}

	existingRdsStatusEndpoint := jira.Status.RDS.Endpoint
	if existingRdsStatusEndpoint != rdsHostname {
		jira.Status.RDS.Endpoint = rdsHostname
		logger.Info("Updating RDS endpoint in Jira status: " + rdsHostname)
		err = r.Status().Update(context.TODO(), jira)
		if err != nil {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
	}

	// when RDS is created from a snapshot root password is not automatically reset
	// with a root password defined in the secret, so we need to run a k8s job to do it with aws cli
	if jira.Spec.Database.SnapshotID != "" {

		serviceAccount := k8s.GetServiceAccount(*jira, namespace.Name)
		err = r.Create(context.TODO(), &serviceAccount)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}

		changeRootPasswordJob := k8s.GetChangeRootPasswordJob(*jira, namespace.Name, rdsInstance.Name)
		err = r.Create(context.TODO(), &changeRootPasswordJob)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}

		jobSucceededReplicas, err := r.getJobSucceededReplicas(changeRootPasswordJob)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
		if jobSucceededReplicas < 1 {
			logger.Info("Reset RDS creds job has the following number of succeeded replicas: " + strconv.Itoa(int(jobSucceededReplicas)))
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
		jira.Status.RDS.ResetRdsCredsJobStatus = "Succeeded"
		err = r.Status().Update(context.TODO(), jira)
		if err != nil {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
	}

	// get liquibase changelog configmap definition and create it
	liquibaseConfigMap, err := k8s.GetLiquibaseConfigMap(*jira, namespace.Name)
	if err != nil {
		logger.Error(err, "Failed to read liquibase changelog file")
		return ctrl.Result{RequeueAfter: 10 * time.Minute}, err
	}
	err = r.Create(context.TODO(), &liquibaseConfigMap)

	if errors.IsAlreadyExists(err) {
		existingLiquibaseConfigMapData, err := r.getConfigMapData(liquibaseConfigMap)
		if err != nil {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
		cmData := existingLiquibaseConfigMapData["changelog.yml"]

		if cmData != liquibaseConfigMap.Data["changelog.yml"] {
			logger.Info("Updating Liquibase changelog ConfigMap")
			err = r.Client.Update(context.TODO(), &liquibaseConfigMap)
			if err != nil {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}
		}
	} else if err != nil {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	// create liquibase job
	liquibaseJob := k8s.GetLiquibaseJob(*jira, namespace.Name)
	err = r.Create(context.TODO(), &liquibaseJob)
	if err != nil && !errors.IsAlreadyExists(err) {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
	}

	liquibaseJobSucceededReplicas, err := r.getJobSucceededReplicas(liquibaseJob)
	if err != nil {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	if liquibaseJobSucceededReplicas < 1 {
		logger.Info("Liquibase changeset job has the following number of succeeded replicas: " + strconv.Itoa(int(liquibaseJobSucceededReplicas)))
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	jira.Status.RDS.LiquibaseJobStatus = "Succeeded"
	err = r.Status().Update(context.TODO(), jira)
	if err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	if jira.Spec.SharedFS.Ebs.SnapshotId != "" {
		// create EBS volume from a snapshot
		ebsVolume := crossplane.GetEbsVolume(*jira)
		err = r.Create(context.TODO(), &ebsVolume)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}

		ebsVolumeId, err := r.getEbsVolumeId(ebsVolume, client.ObjectKey{Name: jira.Name + "-" + string(jira.UID)})
		if err != nil {
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}

		if ebsVolumeId == "" {
			logger.Info("Ebs volume ID is not yet available. Retrying in 5 seconds")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}

		// create nfs-server PersistentVolume using EBS volume handle
		nfsPersistentVolume := k8s.GetEbsPersistentVolume(*jira, ebsVolumeId, jira.Name+"-nfs-server", strconv.Itoa(int(jira.Spec.SharedFS.VolumeSize)), namespace.Name)
		err = r.Create(context.TODO(), &nfsPersistentVolume)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}

		// create nfs-server PersistentVolumeClaim
		nfsPersistentVolumeClaim := k8s.GetPersistentVolumeClaim(*jira, jira.Name+"-nfs-server", namespace.Name, nfsPersistentVolume.Name, jira.Spec.SharedFS.Ebs.EbsStorageClassName, strconv.Itoa(int(jira.Spec.SharedFS.VolumeSize)), corev1.ReadWriteOnce)
		err = r.Create(context.TODO(), &nfsPersistentVolumeClaim)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}

		// create nfs-server svc
		nfsServerService := k8s.GetNfSServerService(*jira, namespace.Name)
		err = r.Create(context.TODO(), &nfsServerService)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}

		// get nfs server svc cluster IP
		nfsServerIp, err := r.getSvcClusterIp(nfsServerService)
		if err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		if nfsServerIp == "" {
			logger.Info("No ClusterIP available for nfs server service")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		// create nfs-server StatefulSet
		nfsServerStatefulSet := k8s.GetNfsServerStatefulSet(*jira, namespace.Name)
		err = r.Create(context.TODO(), &nfsServerStatefulSet)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}

		// get nfs-server statefulset status
		nfsReadyReplicas, err := r.getStsReadyReplicas(nfsServerStatefulSet)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
		}
		if nfsReadyReplicas < 1 {
			logger.Info("Waiting for NFS server to be up and running")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, err
		}

		// create nfs jira shared-home PV
		jiraSharedHomeNfsPv := k8s.GetNfsPersistentVolume(*jira, nfsServerIp, strconv.Itoa(int(jira.Spec.SharedFS.VolumeSize)), namespace.Name)
		err = r.Create(context.TODO(), &jiraSharedHomeNfsPv)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
		}

		// create jira shared home pvc bound to nfs shared home pv
		sharedHomePvcAccessMode := corev1.ReadWriteMany
		jiraSharedHomePvcNfs := k8s.GetPersistentVolumeClaim(*jira, "jira-shared-home", namespace.Name, jiraSharedHomeNfsPv.Name, jira.Spec.SharedFS.Efs.EfsStorageClassName, strconv.Itoa(int(jira.Spec.SharedFS.VolumeSize)), sharedHomePvcAccessMode)
		err = r.Create(context.TODO(), &jiraSharedHomePvcNfs)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		currentEbsId := jira.Status.SharedFilesystemStatus.EbsId
		if currentEbsId != ebsVolumeId {
			logger.Info("Updating Jira status with EBS ID: " + ebsVolumeId)
			jira.Status.SharedFilesystemStatus.EbsId = ebsVolumeId
			err = r.Status().Update(context.TODO(), jira)
			if err != nil {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}
		}

	} else if jira.Spec.SharedFS.Fsx.SnapshotId != "" {

		// create VolumeSnapshot from existing vol handle

		volumeSnapshotContent := k8s.GetFsxVolumeSnapshotContent(*jira, namespace.Name)
		err = r.Create(context.TODO(), &volumeSnapshotContent)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		volumeSnapshot := k8s.GetFsxVolumeSnapshot(*jira, namespace.Name)
		err = r.Create(context.TODO(), &volumeSnapshot)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		fsxPvc := k8s.GetFsxPersistentVolumeClaimFromSnapshot(*jira, "jira-shared-home", namespace.Name, "1")
		err = r.Create(context.TODO(), &fsxPvc)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		fsxPvcStatus, err := r.getPvcStatus(fsxPvc)
		if err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		if fsxPvcStatus != "Bound" {
			logger.Info("Waiting for FSX PVC jira-shared-home to be in Bound state. Current status: " + fsxPvcStatus)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		fsxVolumeName, err := r.getFsxVolumeName(fsxPvc)
		if err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		currentFsxId := jira.Status.SharedFilesystemStatus.FsxId

		if currentFsxId != fsxVolumeName {
			logger.Info("Updating Jira status with FSX volume: " + fsxVolumeName)
			jira.Status.SharedFilesystemStatus.FsxId = fsxVolumeName
			err = r.Status().Update(context.TODO(), jira)
			if err != nil {
				return ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}
		}

	} else {
		// create a new EFS
		sharedFileSystem := crossplane.GetFileSystem(*jira, namespace.Name)
		err = r.Create(context.TODO(), &sharedFileSystem)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}

		fileSystemId, err := r.getFilesystemId(sharedFileSystem, client.ObjectKey{Name: jira.Name + "-" + string(jira.UID)})
		if err != nil {
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
		}

		if *fileSystemId == "" {
			logger.Info("No filesystem ID is available. Requeue after 5 seconds")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}

		// create mountTargets
		for i, subnetId := range jira.Spec.Network.SubnetIDs {
			mountTarget := crossplane.GetMountTargets(*jira, *fileSystemId, subnetId, strconv.Itoa(i))
			err = r.Create(context.TODO(), &mountTarget)
			if err != nil && !errors.IsAlreadyExists(err) {
				return ctrl.Result{RequeueAfter: 10 * time.Second}, err
			}
			mountTargetStatus, err := r.getMountTargetStatus(mountTarget, client.ObjectKey{Name: jira.Name + strconv.Itoa(i) + "-" + string(jira.UID)})
			if err != nil {
				return ctrl.Result{RequeueAfter: 10 * time.Second}, err
			}
			if *mountTargetStatus != "available" {
				logger.Info("Mount target is not available: " + mountTarget.Name)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}
		}

		// create Persistent Volume using efs id
		efsPersistentVolume := k8s.GetEfsPersistentVolume(*jira, *fileSystemId, namespace.Name)
		err = r.Create(context.TODO(), &efsPersistentVolume)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		sharedHomePvcAccessMode := corev1.ReadWriteMany
		efsPersistentVolumeClaim := k8s.GetPersistentVolumeClaim(*jira, "jira-shared-home", namespace.Name, efsPersistentVolume.Name, jira.Spec.SharedFS.Efs.EfsStorageClassName, "10", sharedHomePvcAccessMode)
		err = r.Create(context.TODO(), &efsPersistentVolumeClaim)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		currentEfsId := jira.Status.SharedFilesystemStatus.EfsId
		if currentEfsId != *fileSystemId {
			logger.Info("Updating Jira status with EFS ID: " + *fileSystemId)
			jira.Status.SharedFilesystemStatus.EfsId = *fileSystemId
			err = r.Status().Update(context.TODO(), jira)
			if err != nil {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}
		}
	}

	// Argo CD dependencies conflict with other k8s deps in this project, see: https://github.com/argoproj/argo-cd/issues/14727
	// as a result, rather than creating argo Applicationset in Go, we will process a template and kubectl apply the resulting file
	err = argocd.ProcessApplicationSetTemplate(*jira)
	if err != nil {
		return ctrl.Result{RequeueAfter: 60 * time.Second}, err
	}
	argoFilePath := fmt.Sprintf("argocd/applicationset-%s.yaml", jira.Name)
	args := []string{"apply", "-f", argoFilePath}
	output, err := k8s.RunKubectl(args)
	if err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
	}
	logger.Info("Applicationset: " + string(output))

	// get sync status of Jira application
	getAppKubectlArgs := []string{"get", fmt.Sprintf("application/%s", jira.Name), "-n", fmt.Sprintf("%s", jira.Spec.ArgoCD.Namespace), "-o", "jsonpath={.status.sync.status}"}
	syncStatus, err := k8s.RunKubectl(getAppKubectlArgs)
	if err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
	}

	// update jira with sync status
	currentAppSyncStatus := jira.Status.AppStatus.Sync
	if currentAppSyncStatus != string(syncStatus) {
		logger.Info("Updating app sync status to: " + string(syncStatus))
		jira.Status.AppStatus.Sync = string(syncStatus)
		err = r.Status().Update(context.TODO(), jira)
		if err != nil {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
	}

	// get sync status of Jira application
	getAppKubectlArgs = []string{"get", fmt.Sprintf("application/%s", jira.Name), "-n", fmt.Sprintf("%s", jira.Spec.ArgoCD.Namespace), "-o", "jsonpath={.status.health.status}"}
	healthStatus, err := k8s.RunKubectl(getAppKubectlArgs)
	if err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
	}

	// update jira with sync status
	currentAppHealthStatus := jira.Status.AppStatus.Health
	if currentAppHealthStatus != string(healthStatus) {
		logger.Info("Updating app health status to: " + string(healthStatus))
		jira.Status.AppStatus.Health = string(healthStatus)
		err = r.Status().Update(context.TODO(), jira)
		if err != nil {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
	}

	// requeue if app is not yet healthy
	if string(healthStatus) != "Healthy" {
		logger.Info("Application is not yet healthy. Current status: " + string(healthStatus))
	}

	// end of reconciliation loop
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JiraReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.Jira{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.Secret{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&database.RDSInstance{}).
		Owns(&database.DBSubnetGroup{}).
		Owns(&rds.DBParameterGroup{}).
		Owns(&ec2.Volume{}).
		Owns(&efs.FileSystem{}).
		Owns(&snapshot.VolumeSnapshot{}).
		Owns(&snapshot.VolumeSnapshotContent{}).
		Complete(r)
}
