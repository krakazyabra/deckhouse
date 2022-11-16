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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	virtv1 "kubevirt.io/api/core/v1"
)

// VirtualMachineSpec defines the desired state of VirtualMachine
type VirtualMachineSpec struct {
	Running         *bool                          `json:"running,omitempty" optional:"true"`
	StaticIPAddress string                         `json:"staticIPAddress,omitempty"`
	Resources       v1.ResourceList                `json:"resources,omitempty"`
	UserName        string                         `json:"userName,omitempty"`
	SSHPublicKey    string                         `json:"sshPublicKey,omitempty"`
	BootDisk        *BootDisk                      `json:"bootDisk,omitempty"`
	CloudInit       *virtv1.CloudInitNoCloudSource `json:"cloudInit,omitempty"`
	DiskAttachments *[]DiskSource                  `json:"diskAttachments,omitempty"`
}

// VirtualMachineStatus defines the observed state of VirtualMachine
type VirtualMachineStatus struct {
	// Phase is a human readable, high-level representation of the status of the virtual machine
	Phase virtv1.VirtualMachinePrintableStatus `json:"phase,omitempty"`
	// NodeName is the name where the VirtualMachineInstance is currently running.
	NodeName string `json:"nodeName,omitempty"`
	// IP address of Virtual Machine
	IPAddress string `json:"ipAddress,omitempty"`
}

// Represents the source of a boot disk
// Only one of its members may be specified.
type BootDisk struct {
	Source *corev1.TypedLocalObjectReference `json:"source"`
	// Type represents the type for newly created disk
	StorageClassName string `json:"storageClassName,omitempty"`
	// Type represents the size for newly created disk
	Size resource.Quantity `json:"size"`
	// Should boot disk be removed with VM
	Ephemeral bool `json:"ephemeral,omitempty"`
	// Hotpluggable indicates whether the volume can be hotplugged and hotunplugged.
	// +optional
	Hotpluggable bool `json:"hotpluggable,omitempty"`
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi, usb.
	Bus string `json:"bus,omitempty"`
}

// ImageSourceScope represents the source of the image.
// +enum
type ImageSourceScope string

const (
	// ImageSourceScopePublic indicates that disk should be
	// created from public image. This is the default mode.
	ImageSourceScopePublic ImageSourceScope = "public"

	// ImageSourceScopePrivate indicates that disk should be
	// created from private image from the same namespace.
	ImageSourceScopePrivate ImageSourceScope = "private"
)

// Represents the source of existing disk
type DiskSource struct {
	// Name represents the name of the Disk in the same namespace
	Name string `json:"name"`
	// Hotpluggable indicates whether the volume can be hotplugged and hotunplugged.
	// +optional
	Hotpluggable bool `json:"hotpluggable,omitempty"`
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi, usb.
	Bus string `json:"bus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName={"vm","vms"}

// VirtualMachine is the Schema for the virtualmachines API
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec,omitempty"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VirtualMachineList contains a list of VirtualMachine
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}
