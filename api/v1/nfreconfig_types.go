/*
Copyright 2025.

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

type ClusterType string

const (
	ClusterTypeCUCP ClusterType = "CUCP"
	ClusterTypeCUUP ClusterType = "CUUP"
	ClusterTypeDU   ClusterType = "DU"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NFReconfigSpec defines the desired state of NFReconfig
type NFReconfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of NFReconfig. Edit nfreconfig_types.go to remove/update
	// +optional
	Provider         string               `json:"provider"`
	Interfaces       []NFInterface        `json:"interfaces,omitempty"`
	NetworkInstances []NetworkInstance    `json:"networkInstances,omitempty"`
	ParametersRefs   []ParameterReference `json:"parametersRefs,omitempty"`
	ClusterInfo      []ClusterInfo        `json:"clusterInfo"`
}

type ClusterInfo struct {
	Name          string        `json:"name"`
	Repo          string        `json:"repo"`
	NFDeployment  NFDeployment  `json:"nfDeployment,omitempty"`
	ClusterType   ClusterType   `json:"clusterType"`
	TargetCluster TargetCluster `json:"targetCluster,omitempty"`
	ConfigRef     ConfigRef     `json:"configRef,omitempty"`
}

type TargetCluster struct {
	Name string `json:"name"`
	Repo string `json:"repo"`
}

type NFDeployment struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ConfigRef struct {
	Name         string       `json:"name"`
	Namespace    string       `json:"namespace"`
	NFDeployment NFDeployment `json:"nfDeployment,omitempty"`
}

type NFInterface struct {
	Name          string `json:"name"`
	NameNad       string `json:"nameNad"`
	NamespaceNad  string `json:"namespaceNad"`
	HostInterface string `json:"hostInterface"`
	IPv4          *IPv4  `json:"ipv4,omitempty"`
}

type IPv4 struct {
	Address string `json:"address"`
	Gateway string `json:"gateway"`
}

type NetworkInstance struct {
	Name       string   `json:"name"`
	Interfaces []string `json:"interfaces,omitempty"`
}

type ParameterReference struct {
	Name       string `json:"name"`
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
}

// NFReconfigStatus defines the observed state of NFReconfig.
type NFReconfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the NFReconfig resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []NFCondition `json:"conditions,omitempty"`
}

type NFCondition struct {
	// +kubebuilder:validation:Required
	Type               string      `json:"type"`
	Status             string      `json:"status,omitempty"` // True, False, Unknown
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NFReconfig is the Schema for the nfreconfigs API
type NFReconfig struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of NFReconfig
	// +required
	Spec NFReconfigSpec `json:"spec"`

	// status defines the observed state of NFReconfig
	// +optional
	Status NFReconfigStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// NFReconfigList contains a list of NFReconfig
type NFReconfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NFReconfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NFReconfig{}, &NFReconfigList{})
}
