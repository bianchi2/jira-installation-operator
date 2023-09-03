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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type DatabaseSpec struct {
	DBInstanceClass  string `json:"dBInstanceClass,omitempty"`
	AllocatedStorage int    `json:"allocatedStorage,omitempty"`
	Engine           string `json:"engine,omitempty"`
	EngineVersion    string `json:"engineVersion,omitempty"`
	SnapshotID       string `json:"snapshotId,omitempty"`
}

type SharedFS struct {
	SnapshotId                string `json:"snapshotId,omitempty"`
	VolumeSize                int64  `json:"volumeSize,omitempty"`
	NfsServerAvailabilityZone string `json:"nfsServerAvailabilityZone,omitempty"`
	EfsStorageClassName       string `json:"efsStorageClassName,omitempty"`
	EbsStorageClassName       string `json:"ebsStorageClassName,omitempty"`
	EfsCsiDriverName          string `json:"efsCsiDriverName,omitempty"`
}

type HelmValues struct {
	GitRepo         string   `json:"gitRepo,omitempty"`
	GitRevision     string   `json:"gitRevision,omitempty"`
	HelmValuesFiles []string `json:"helmValuesFiles,omitempty"`
	ValueOverrides  string   `json:"valueOverrides,omitempty"`
}

type HelmChart struct {
	Version string `json:"version,omitempty"`
	RepoURL string `json:"repoUrl,omitempty"`
}

type SyncPolicy struct {
	AutoSync           bool `json:"autoSync,omitempty"`
	ApplyOutOfSyncOnly bool `json:"applyOutOfSyncOnly,omitempty"`
}

type ArgoCDSpec struct {
	HelmValues     HelmValues `json:"helmValues,omitempty"`
	HelmChart      HelmChart  `json:"helmChart,omitempty"`
	Namespace      string     `json:"namespace,omitempty"`
	Project        string     `json:"project,omitempty"`
	SyncPolicy     SyncPolicy `json:"syncPolicy,omitempty"`
	RetainOnDelete bool       `json:"retainOnDelete,omitempty"`
}

type Network struct {
	SubnetIDs        []string `json:"subnetIds,omitempty"`
	SecurityGroupIds []string `json:"securityGroupIds,omitempty"`
}

// JiraSpec defines the desired state of Jira
type JiraSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Jira. Edit jira_types.go to remove/update
	AWSRegion      string       `json:"awsRegion,omitempty"`
	RetainOnDelete bool         `json:"retainOnDelete,omitempty"`
	Database       DatabaseSpec `json:"database,omitempty"`
	Hostname       string       `json:"hostname,omitempty"`
	ArgoCD         ArgoCDSpec   `json:"argocd,omitempty"`
	SharedFS       SharedFS     `json:"sharedFs,omitempty"`
	Network        Network      `json:"network,omitempty"`
	KMSKeyId       string       `json:"kmsKeyId,omitempty"`
}

type RDSStatus struct {
	Status                 string `json:"status,omitempty"`
	Endpoint               string `json:"endpoint,omitempty"`
	LiquibaseJobStatus     string `json:"liquibaseJobStatus,omitempty"`
	ResetRdsCredsJobStatus string `json:"resetRdsCredsJobStatus,omitempty"`
}

type AppStatus struct {
	Health string `json:"health,omitempty"`
	Sync   string `json:"sync,omitempty"`
}

// JiraStatus defines the observed state of Jira
type JiraStatus struct {
	RDS       RDSStatus `json:"rds,omitempty"`
	AppStatus AppStatus `json:"app,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Jira is the Schema for the jiras API
type Jira struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JiraSpec   `json:"spec,omitempty"`
	Status JiraStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JiraList contains a list of Jira
type JiraList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jira `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jira{}, &JiraList{})
}
