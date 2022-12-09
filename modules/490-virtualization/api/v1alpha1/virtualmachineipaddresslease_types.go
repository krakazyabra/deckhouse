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
)

// VirtualMachineIPAddressLeaseSpec defines the desired state of VirtualMachineIPAddressLease
type VirtualMachineIPAddressLeaseSpec struct {
	// Static represents the static claim
	ClaimRef *corev1.ObjectReference `json:"claimRef,omitempty"`
}

// VirtualMachineIPAddressLeaseStatus defines the observed state of VirtualMachineIPAddressLease
type VirtualMachineIPAddressLeaseStatus struct {
	Phase string `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".spec.static",name=Static,type=string
//+kubebuilder:printcolumn:JSONPath=".spec.vmName",name=VM,type=string
//+kubebuilder:resource:scope=Cluster,shortName={"vmipl","vmipls"}

// VirtualMachineIPAddressLease is the Schema for the virtualmachineipaddressleases API
type VirtualMachineIPAddressLease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineIPAddressLeaseSpec   `json:"spec,omitempty"`
	Status VirtualMachineIPAddressLeaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VirtualMachineIPAddressLeaseList contains a list of VirtualMachineIPAddressLease
type VirtualMachineIPAddressLeaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineIPAddressLease `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachineIPAddressLease{}, &VirtualMachineIPAddressLeaseList{})
}
