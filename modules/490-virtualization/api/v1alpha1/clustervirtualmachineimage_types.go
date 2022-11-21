/*
Copyright 2022.

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
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

// VirtualMachineDiskSpec defines the desired state of VirtualMachineDisk
type ClusterVirtualMachineImageSpec struct {
	Remote *cdiv1.DataVolumeSource `json:"remote,omitempty"`
	Source *TypedObjectReference   `json:"source,omitempty"`
}

type TypedObjectReference struct {
	corev1.TypedLocalObjectReference `json:",inline"`
	Namespace                        string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
}

// ClusterVirtualMachineImageStatus defines the observed state of ClusterVirtualMachineImage
type ClusterVirtualMachineImageStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:resource:shortName={"cvmi","cvmimage","cvmimages"}

// ClusterVirtualMachineImage is the Schema for the clustervirtualmachineimages API
type ClusterVirtualMachineImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterVirtualMachineImageSpec   `json:"spec,omitempty"`
	Status ClusterVirtualMachineImageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterVirtualMachineImageList contains a list of ClusterVirtualMachineImage
type ClusterVirtualMachineImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterVirtualMachineImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterVirtualMachineImage{}, &ClusterVirtualMachineImageList{})
}
