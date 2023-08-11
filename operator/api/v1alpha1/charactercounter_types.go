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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CharacterCounterSpec defines the desired state of CharacterCounter
type CharacterCounterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`

	Frontend CharacterCounterComponent `json:"frontend"`
	Server   CharacterCounterComponent `json:"server"`
}

type CharacterCounterComponent struct {
	Name         string                 `json:"name,omitempty"`
	Image        string                 `json:"image,omitempty"`
	Replicas     *int32                 `json:"replicas,omitempty"`
	Selector     map[string]string      `json:"selector,omitempty"`
	Ports        []corev1.ContainerPort `json:"ports,omitempty"`
	ServicePorts []corev1.ServicePort   `json:"servicePorts,omitempty"`
}

type CharacterCounterCondition string

// These are valid conditions of a deployment.
const (
	DeploymentAvailable   CharacterCounterCondition = "Available"
	DeploymentProgressing CharacterCounterCondition = "Progressing"
)

// CharacterCounterStatus defines the observed state of CharacterCounter
type CharacterCounterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Condition CharacterCounterCondition `json:"condition"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CharacterCounter is the Schema for the charactercounters API
type CharacterCounter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CharacterCounterSpec   `json:"spec,omitempty"`
	Status CharacterCounterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CharacterCounterList contains a list of CharacterCounter
type CharacterCounterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CharacterCounter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CharacterCounter{}, &CharacterCounterList{})
}
