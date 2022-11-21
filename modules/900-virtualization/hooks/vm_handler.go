/*
Copyright 2021 Flant JSC

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

package hooks

import (
	"fmt"
	"strconv"

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/shell-operator/pkg/kube_events_manager/types"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	virtv1 "kubevirt.io/api/core/v1"

	"github.com/deckhouse/deckhouse/modules/900-virtualization/api/v1alpha1"
)

const (
	deckhouseVMSnapshot    = "vmHandlerDeckhouseVM"
	kubevirtVMSnapshot     = "vmHandlerKubevirtVM"
	kubevirtVMsCRDSnapshot = "vmHandlerKubevirtVMCRD"
	vmDisksSnapshot        = "vmDisksSnapshot"
)

var vmHandlerHookConfig = &go_hook.HookConfig{
	Queue: "/modules/virtualization/vm-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		// A binding with dynamic kind has index 0 for simplicity.
		{
			Name:       kubevirtVMSnapshot,
			ApiVersion: "",
			Kind:       "",
			FilterFunc: applyKubevirtVMFilter,
		},
		{
			Name:       deckhouseVMSnapshot,
			ApiVersion: gv,
			Kind:       "VirtualMachine",
			FilterFunc: applyDeckhouseVMFilter,
		},
		{
			Name:       vmDisksSnapshot,
			ApiVersion: gv,
			Kind:       "VirtualMachineDisk",
			FilterFunc: applyVirtualMachineDisksFilter,
		},
		{
			Name:       kubevirtVMsCRDSnapshot,
			ApiVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
			NameSelector: &types.NameSelector{
				MatchNames: []string{"virtualmachines.kubevirt.io"},
			},
			FilterFunc: applyCRDExistenseFilter,
		},
	},
}

var _ = sdk.RegisterFunc(vmHandlerHookConfig, handleVMs)

type InstanceClassCrdInfo struct {
	Name string
	Spec interface{}
}

func applyKubevirtVMFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	vm := &virtv1.VirtualMachine{}
	err := sdk.FromUnstructured(obj, vm)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachine: %v", err)
	}

	return vm, nil
}

func applyDeckhouseVMFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	vm := &v1alpha1.VirtualMachine{}
	err := sdk.FromUnstructured(obj, vm)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachine: %v", err)
	}

	return vm, nil
}

func applyVirtualMachineDisksFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	disk := &v1alpha1.VirtualMachineDisk{}
	err := sdk.FromUnstructured(obj, disk)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachineDisk: %v", err)
	}

	return &VirtualMachineDiskSnapshot{
		Name:      disk.Name,
		Namespace: disk.Namespace,
		VMName:    disk.Spec.VMName,
		Ephemeral: disk.Spec.Ephemeral,
	}, nil
}

// handleVMs
//
// synopsis:
//   TODO
func handleVMs(input *go_hook.HookInput) error {
	// KubeVirt manages it's own CRDs, so we need to wait for them before starting the watch
	if vmHandlerHookConfig.Kubernetes[0].Kind == "" {
		if len(input.Snapshots[kubevirtVMsCRDSnapshot]) > 0 {
			// KubeVirt installed
			input.LogEntry.Infof("KubeVirt VirtualMachine CRD installed, update kind for binding VirtualMachines.kubevirt.io")
			*input.BindingActions = append(*input.BindingActions, go_hook.BindingAction{
				Name:       kubevirtVMSnapshot,
				Action:     "UpdateKind",
				ApiVersion: "kubevirt.io/v1",
				Kind:       "VirtualMachine",
			})
			// Save new kind as current kind.
			vmHandlerHookConfig.Kubernetes[0].Kind = "VirtualMachine"
			vmHandlerHookConfig.Kubernetes[0].ApiVersion = "kubevirt.io/v1"
			// Binding changed, hook will be restarted with new objects in snapshot.
			return nil
		}
		// KubeVirt is not yet installed, do nothing
		return nil
	}

	// Start main hook logic
	kubevirtVMSnap := input.Snapshots[kubevirtVMSnapshot]
	deckhouseVMSnap := input.Snapshots[deckhouseVMSnapshot]
	diskSnap := input.Snapshots[vmDisksSnapshot]

	if len(kubevirtVMSnap) == 0 && len(deckhouseVMSnap) == 0 && len(diskSnap) == 0 {
		input.LogEntry.Warnln("VirtualMachines and VirtualMachineDisks not found. Skip")
		return nil
	}

	for _, sRaw := range deckhouseVMSnap {
		d8vm := sRaw.(*v1alpha1.VirtualMachine)
		if d8vm.Status.IPAddress == "" {
			// IPAddress is not set by IPAM, nothing to do
			continue
		}
		// Handle boot disk
		var bootVirtualMachineDiskName string
		if d8vm.Spec.BootDisk != nil {
			var disk *VirtualMachineDiskSnapshot

			switch d8vm.Spec.BootDisk.Source.Kind {
			case "VirtualMachineDisk":
				bootVirtualMachineDiskName = d8vm.Spec.BootDisk.Source.Name
				disk = getDisk(&diskSnap, d8vm.Namespace, d8vm.Spec.BootDisk.Source.Name)
				if disk == nil {
					input.LogEntry.Warnln("Disk not found")
				}

			case "ClusterVirtualMachineImage":
				bootVirtualMachineDiskName = d8vm.Name + "-boot"
				disk = getDisk(&diskSnap, d8vm.Namespace, bootVirtualMachineDiskName)

				if disk != nil {

					// Disk found, ensure ephemeral is set
					if d8vm.Spec.BootDisk.Ephemeral != disk.Ephemeral {
						patch := map[string]interface{}{"spec": map[string]bool{"ephemeral": d8vm.Spec.BootDisk.Ephemeral}}
						input.PatchCollector.MergePatch(patch, gv, "VirtualMachineDisk", disk.Namespace, disk.Name)
					}
				} else {
					// Disk not found, create a new VirtualMachineDisk
					disk := &v1alpha1.VirtualMachineDisk{
						TypeMeta: metav1.TypeMeta{
							Kind:       "VirtualMachineDisk",
							APIVersion: gv,
						},
						ObjectMeta: v1.ObjectMeta{
							Name:      bootVirtualMachineDiskName,
							Namespace: d8vm.Namespace,
						},
						Spec: v1alpha1.VirtualMachineDiskSpec{
							StorageClassName: d8vm.Spec.BootDisk.StorageClassName,
							Size:             d8vm.Spec.BootDisk.Size,
							Source:           d8vm.Spec.BootDisk.Source,
							VMName:           d8vm.Name,
							Ephemeral:        d8vm.Spec.BootDisk.Ephemeral,
						},
					}
					input.PatchCollector.Create(disk)
				}

			default:
				input.LogEntry.Warnln("Unknown source kind")
			}

			if disk != nil {
				err := checkAndApplyDiskPatches(input, d8vm, disk)
				if err != nil {
					return err
				}
			}
		}

		// handle other disks
		if d8vm.Spec.DiskAttachments != nil {
			for _, diskSource := range *d8vm.Spec.DiskAttachments {
				disk := getDisk(&diskSnap, d8vm.Namespace, diskSource.Name)
				err := checkAndApplyDiskPatches(input, d8vm, disk)
				if err != nil {
					return err
				}
			}
		}

		// handle KubeVirt VirtualMachine
		kvvm := getKubevirtVM(&kubevirtVMSnap, d8vm.Namespace, d8vm.Name)
		if kvvm != nil {
			// KubeVirt VirtualMachine found
			apply := func(u *unstructured.Unstructured) (*unstructured.Unstructured, error) {
				vm := &virtv1.VirtualMachine{}
				err := sdk.FromUnstructured(u, vm)
				if err != nil {
					return nil, err
				}
				err = setVMFields(d8vm, vm, bootVirtualMachineDiskName)
				if err != nil {
					return nil, err
				}
				return sdk.ToUnstructured(&vm)
			}
			input.PatchCollector.Filter(apply, "kubevirt.io/v1", "VirtualMachine", d8vm.Namespace, d8vm.Name)
		} else {
			// KubeVirt VirtualMachine not found, needs to create a new one
			vm := &virtv1.VirtualMachine{}
			err := setVMFields(d8vm, vm, bootVirtualMachineDiskName)
			if err != nil {
				return err
			}
			input.PatchCollector.Create(vm)
		}
	}

	// Disks cleanup loop
	for _, sRaw := range diskSnap {
		disk := sRaw.(*VirtualMachineDiskSnapshot)
		if disk.VMName != "" && getD8VM(&deckhouseVMSnap, disk.Namespace, disk.VMName) == nil {
			if disk.Ephemeral {
				// Delete disk
				input.PatchCollector.Delete(gv, "VirtualMachineDisk", disk.Namespace, disk.Name)
			} else {
				// Remove vmName
				patch := map[string]interface{}{"spec": map[string]string{"vmName": ""}}
				input.PatchCollector.MergePatch(patch, gv, "VirtualMachineDisk", disk.Namespace, disk.Name)
			}
		}
	}

	return nil
}

func setVMFields(d8vm *v1alpha1.VirtualMachine, vm *virtv1.VirtualMachine, bootVirtualMachineDiskName string) error {
	vm.TypeMeta = metav1.TypeMeta{
		Kind:       "VirtualMachine",
		APIVersion: "kubevirt.io/v1",
	}
	vm.SetName(d8vm.Name)
	vm.SetNamespace(d8vm.Namespace)
	vm.SetOwnerReferences([]v1.OwnerReference{{
		APIVersion:         gv,
		BlockOwnerDeletion: pointer.Bool(true),
		Controller:         pointer.Bool(true),
		Kind:               "VirtualMachine",
		Name:               d8vm.Name,
		UID:                d8vm.UID,
	}})

	cloudInit := make(map[string]interface{})
	if d8vm.Spec.CloudInit != nil {
		err := yaml.Unmarshal([]byte(d8vm.Spec.CloudInit.UserData), &cloudInit)
		if err != nil {
			return fmt.Errorf("cannot parse cloudInit config for VirtualMachine: %v", err)
		}
	}
	if d8vm.Spec.SSHPublicKey != "" {
		cloudInit["ssh_authorized_keys"] = []string{d8vm.Spec.SSHPublicKey}
	}
	if d8vm.Spec.UserName != "" {
		cloudInit["user"] = d8vm.Spec.UserName
	}
	cloudInitRaw, _ := yaml.Marshal(cloudInit)

	vm.Spec.Running = d8vm.Spec.Running
	vm.Spec.Template = &virtv1.VirtualMachineInstanceTemplateSpec{
		ObjectMeta: v1.ObjectMeta{
			Annotations: map[string]string{
				"cni.cilium.io/ipAddrs":  d8vm.Status.IPAddress,
				"cni.cilium.io/macAddrs": "f6:e1:74:94:b8:1a",
			},
		},
		Spec: virtv1.VirtualMachineInstanceSpec{
			Domain: virtv1.DomainSpec{
				Devices: virtv1.Devices{
					Interfaces: []virtv1.Interface{{
						Name:  "default",
						Model: "virtio",
						InterfaceBindingMethod: virtv1.InterfaceBindingMethod{
							Macvtap: &virtv1.InterfaceMacvtap{},
						},
					}},
					Disks: []virtv1.Disk{
						{
							Name: "boot",
							DiskDevice: virtv1.DiskDevice{
								Disk: &virtv1.DiskTarget{
									Bus: "virtio",
								},
							},
						},
						{
							Name: "cloudinit",
							DiskDevice: virtv1.DiskDevice{
								Disk: &virtv1.DiskTarget{
									Bus: "virtio",
								},
							},
						},
					},
				},
				Resources: virtv1.ResourceRequirements{
					Requests: d8vm.Spec.Resources,
				},
			},
			Networks: []virtv1.Network{{
				Name: "default",
				NetworkSource: virtv1.NetworkSource{
					Pod: &virtv1.PodNetwork{},
				}}},
			Volumes: []virtv1.Volume{
				{
					Name: "boot",
					VolumeSource: virtv1.VolumeSource{
						DataVolume: &virtv1.DataVolumeSource{
							Name: "disk-" + bootVirtualMachineDiskName,
							// Workaround for gh:#7710
							Hotpluggable: true,
						},
					},
				},
				{
					Name: "cloudinit",
					VolumeSource: virtv1.VolumeSource{
						CloudInitNoCloud: &virtv1.CloudInitNoCloudSource{
							// TODO handle cloudinit from secret
							UserData: fmt.Sprintf("#cloud-config\n%s", cloudInitRaw),
						},
					},
				},
			},
		},
	}

	// attach extra disks
	if d8vm.Spec.DiskAttachments != nil {
		for i, disk := range *d8vm.Spec.DiskAttachments {
			diskName := "disk-" + strconv.Itoa(i+1)
			vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, virtv1.Disk{
				Name: diskName,
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{
						Bus: disk.Bus,
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: diskName,
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "disk-" + disk.Name,
						// Workaround for gh:#7710
						Hotpluggable: true, // disk.Hotpluggable
					},
				},
			})
		}
	}
	return nil
}

func getKubevirtVM(snapshot *[]go_hook.FilterResult, namespace, name string) *virtv1.VirtualMachine {
	for _, dRaw := range *snapshot {
		vm := dRaw.(*virtv1.VirtualMachine)
		if vm.Namespace == namespace && vm.Name == name {
			return vm
		}
	}
	return nil
}

func getD8VM(snapshot *[]go_hook.FilterResult, namespace, name string) *v1alpha1.VirtualMachine {
	for _, dRaw := range *snapshot {
		vm := dRaw.(*v1alpha1.VirtualMachine)
		if vm.Namespace == namespace && vm.Name == name {
			return vm
		}
	}
	return nil
}

func checkAndApplyDiskPatches(input *go_hook.HookInput, d8vm *v1alpha1.VirtualMachine, disk *VirtualMachineDiskSnapshot) error {
	if disk.VMName != "" && disk.VMName != d8vm.Name {
		return fmt.Errorf("disk already attached to other VirtualMachine: %v", disk.VMName)
	}
	if disk.VMName != d8vm.Name {
		patch := map[string]interface{}{"spec": map[string]string{"vmName": d8vm.Name}}
		input.PatchCollector.MergePatch(patch, gv, "VirtualMachineDisk", disk.Namespace, disk.Name)
	}
	return nil
}
