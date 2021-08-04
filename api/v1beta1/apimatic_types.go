/*
Copyright 2021 APIMatic.io.

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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// APIMaticSpec defines the desired state of APIMatic
type APIMaticSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// replicas is the desired number of instances of APIMatic
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Required
	PodSpec APIMaticPodSpec `json:"podspec"`

	// Resource Requirements represents the compute resource requirements of the APIMatic container
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// volumeClaimTemplates is a list of claims that pods are allowed to reference.
	// The StatefulSet controller is responsible for mapping network identities to
	// claims in a way that maintains the identity of a pod. Every claim in
	// this list must have at least one matching (by name) volumeMount in one
	// container in the template. A claim in this list takes precedence over
	// any volumes in the template, with the same name.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems:=1
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
}

// APIMaticStatus defines the observed state of APIMatic
type APIMaticStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// APIMaticPodSpec contains configuration for created APIMatic pods
type APIMaticPodSpec struct {
	// APIMatic container image
	// +kubebuilder:validation:Required
	Image string `json:"image"`

 // PullPolicy describes a policy for if/when to pull a container image. Valid values are Always, Never and IfNotPresent
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`

	// sidecars are the collection of sidecar containers in addition to the main APIMatic container
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=true
	SideCars []corev1.Container `json:"sideCars,omitempty"`

	// More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=true
	InitContainers []corev1.Container `json:"initContainers,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// APIMatic is the Schema for the apimatics API
type APIMatic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   APIMaticSpec   `json:"spec,omitempty"`
	Status APIMaticStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// APIMaticList contains a list of APIMatic
type APIMaticList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []APIMatic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&APIMatic{}, &APIMaticList{})
}
