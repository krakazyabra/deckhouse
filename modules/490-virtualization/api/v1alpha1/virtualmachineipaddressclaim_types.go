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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualMachineIPAddressClaimSpec defines the desired state of VirtualMachineIPAddressClaim
type VirtualMachineIPAddressClaimSpec struct {
	// Static represents the static claim
	//+kubebuilder:default:=true
	//+kubebuilder:validation:Required
	Static    *bool  `json:"static"`
	LeaseName string `json:"leaseName,omitempty"`
	Address   string `json:"address,omitempty"`
}

// VirtualMachineIPAddressClaimStatus defines the observed state of VirtualMachineIPAddressClaim
type VirtualMachineIPAddressClaimStatus struct {
	Phase  string `json:"phase,omitempty"`
	VMName string `json:"vmName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".spec.address",name=Address,type=string
//+kubebuilder:printcolumn:JSONPath=".spec.static",name=Static,type=string
//+kubebuilder:printcolumn:JSONPath=".status.phase",name=Status,type=string
//+kubebuilder:printcolumn:JSONPath=".status.vmName",name=VM,type=string
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:resource:shortName={"vmip","vmips"}

// VirtualMachineIPAddressClaim is the Schema for the virtualmachineipaddressleases API
type VirtualMachineIPAddressClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineIPAddressClaimSpec   `json:"spec,omitempty"`
	Status VirtualMachineIPAddressClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VirtualMachineIPAddressClaimList contains a list of VirtualMachineIPAddressClaim
type VirtualMachineIPAddressClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineIPAddressClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachineIPAddressClaim{}, &VirtualMachineIPAddressClaimList{})
}
