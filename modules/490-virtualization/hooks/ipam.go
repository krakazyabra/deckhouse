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
	"net"

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/shell-operator/pkg/kube/object_patch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/deckhouse/deckhouse/modules/490-virtualization/api/v1alpha1"
)

const (
	ipsSnapshot = "ipamIP"
	vmsSnapshot = "ipamVM"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: "/modules/virtualization/ipam-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       ipsSnapshot,
			ApiVersion: gv,
			Kind:       "VirtualMachineIPAddressLease",
			FilterFunc: applyVirtualMachineIPAddressLeaseFilter,
		},
		{
			Name:       vmsSnapshot,
			ApiVersion: gv,
			Kind:       "VirtualMachine",
			FilterFunc: applyVirtualMachineFilter,
		},
	},
}, handleVMsAndIPs)

type VirtualMachineIPAddressLeaseSnapshot struct {
	Name      string
	Namespace string
	Address   string
	Static    bool
	VMName    string
}

type VirtualMachineSnapshot struct {
	Name            string
	Namespace       string
	StaticIPAddress string
	StatusIPAddress string
}

func applyVirtualMachineIPAddressLeaseFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	claim := &v1alpha1.VirtualMachineIPAddressLease{}
	err := sdk.FromUnstructured(obj, claim)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachineIPAddressLease: %v", err)
	}
	address := nameToIP(claim.Name)
	if address == "" {
		return nil, fmt.Errorf("cannot convert VirtualMachineIPAddressLease name to IP address: %s", claim.Name)
	}

	return &VirtualMachineIPAddressLeaseSnapshot{
		Name:      claim.Name,
		Namespace: claim.Namespace,
		Address:   address,
		Static:    claim.Spec.Static,
		VMName:    claim.Spec.VMName,
	}, nil
}

func applyVirtualMachineFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	vm := &v1alpha1.VirtualMachine{}
	err := sdk.FromUnstructured(obj, vm)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachine: %v", err)
	}

	return &VirtualMachineSnapshot{
		Name:            vm.Name,
		Namespace:       vm.Namespace,
		StaticIPAddress: vm.Spec.StaticIPAddress,
		StatusIPAddress: vm.Status.IPAddress,
	}, nil
}

// handleDisks
//
// synopsis:
//   TODO
func handleVMsAndIPs(input *go_hook.HookInput) error {
	ipSnap := input.Snapshots[ipsSnapshot]
	vmSnap := input.Snapshots[vmsSnapshot]
	if len(ipSnap) == 0 && len(vmSnap) == 0 {
		input.LogEntry.Warnln("VirtualMachineIPAddressLease and VirtualMachine not found. Skip")
		return nil
	}

	allocatedIPs := make(map[string]string)

CLAIM_LOOP:
	// Handle VirtualMachineIPAddressLeases
	for _, sRaw := range ipSnap {
		claim := sRaw.(*VirtualMachineIPAddressLeaseSnapshot)

		// Address is static, but currently not in use
		if claim.Static && claim.VMName == "" {
			allocatedIPs[claim.Address] = ""
			continue CLAIM_LOOP
		}

		for _, dRaw := range vmSnap {
			vm := dRaw.(*VirtualMachineSnapshot)

			if claim.Namespace != vm.Namespace {
				continue
			}
			if claim.VMName != vm.Name {
				continue
			}
			// VM found

			// Handle case when VM object contains other StaticIPAddress
			if vm.StaticIPAddress != "" && vm.StaticIPAddress != claim.Address {
				input.LogEntry.Warnf("VM %s/%s for IP %s is found, but other IP %s requested, releasing", vm.Namespace, vm.Name, claim.Name, vm.StaticIPAddress)
				if claim.Static {
					patch := map[string]interface{}{"spec": map[string]string{"vmName": ""}}
					input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressLease", claim.Namespace, claim.Name)
				} else {
					input.PatchCollector.Delete(gv, "VirtualMachineIPAddressLease", claim.Namespace, claim.Name)
					continue CLAIM_LOOP
				}
			}

			// VM requested static IP, mark claim as static
			if vm.StaticIPAddress == claim.Address && !claim.Static {
				patch := map[string]interface{}{"spec": map[string]bool{"static": true}}
				input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressLease", claim.Namespace, claim.Name)
			}

			if vm.StatusIPAddress != claim.Address {
				patch := map[string]interface{}{"status": map[string]string{"ipAddress": claim.Address}}
				input.PatchCollector.MergePatch(patch, gv, "VirtualMachine", vm.Namespace, vm.Name, object_patch.WithSubresource("/status"))
			}

			// Add IP to allocation map
			allocatedIPs[claim.Address] = claim.Namespace + "/" + claim.VMName
			continue CLAIM_LOOP
		}

		// VM is not found, release the dynamic lease
		if !claim.Static {
			input.PatchCollector.Delete(gv, "VirtualMachineIPAddressLease", claim.Namespace, claim.Name)
			continue CLAIM_LOOP
		}

		// VM is not found, preserve the static lease
		if claim.VMName != "" {
			patch := map[string]interface{}{"spec": map[string]string{"vmName": ""}}
			input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressLease", claim.Namespace, claim.Name)
		}

		// Add IP to allocation map
		allocatedIPs[claim.Address] = ""
		continue CLAIM_LOOP
	}

	// Load CIDRs from config
	var parsedCIDRs []*net.IPNet
	for _, cidr := range input.Values.Get("virtualization.vmCIDRs").Array() {
		_, parsedCIDR, err := net.ParseCIDR(cidr.String())
		if err != nil || parsedCIDR == nil {
			return fmt.Errorf("Can not parse CIDR %s", cidr)
		}
		parsedCIDRs = append(parsedCIDRs, parsedCIDR)
	}

	// Handle VMs
	for _, sRaw := range vmSnap {
		vm := sRaw.(*VirtualMachineSnapshot)
		var newLease bool
		var ip string

		// Handle case when VM requested static IP
		if vm.StaticIPAddress != "" {
			ip = vm.StaticIPAddress
			vmString := vm.Namespace + "/" + vm.Name
			v, found := allocatedIPs[ip]
			if v != "" && v != vmString {
				// Static Claim is found, but it is in use by other VM
				input.LogEntry.Warnf("VM %s/%s requested IP %s, but it is already allocated for %s", vm.Namespace, vm.Name, ip, vmString)
				continue
			}
			if v == "" && found {
				// Static Claim is found, needs to update vmName
				patch := map[string]interface{}{"spec": map[string]interface{}{"vmName": vm.Name, "static": true}}
				input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressLease", vm.Namespace, ipToName(ip))
				allocatedIPs[ip] = vmString
			}
			if v == "" && !found {
				newLease = true
			}
		}

		if vm.StaticIPAddress == "" {
			ip, newLease = findIPForVM(&parsedCIDRs, allocatedIPs, vm.Namespace+"/"+vm.Name)
			if ip == "" {
				input.LogEntry.Errorf("Error allocating new IP Address for VM %s/%s", vm.Namespace, vm.Name)
				continue
			}
		}

		if vm.StatusIPAddress != ip {
			patch := map[string]interface{}{"status": map[string]interface{}{"ipAddress": ip}}
			input.PatchCollector.MergePatch(patch, gv, "VirtualMachine", vm.Namespace, vm.Name, object_patch.WithSubresource("/status"))
		}

		// Claim already created
		if !newLease {
			continue
		}

		// Claim is not found, create a new one
		claim := &v1alpha1.VirtualMachineIPAddressLease{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VirtualMachineIPAddressLease",
				APIVersion: gv,
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      ipToName(ip),
				Namespace: vm.Namespace,
			},
			Spec: v1alpha1.VirtualMachineIPAddressLeaseSpec{
				VMName: vm.Name,
			},
		}
		if vm.StaticIPAddress != "" {
			claim.Spec.Static = true
		}
		input.PatchCollector.Create(claim)

		// Add IP to allocation map
		allocatedIPs[ip] = vm.Namespace + "/" + vm.Name
	}

	return nil
}

func findIPForVM(parsedCIDRs *[]*net.IPNet, allocatedIPs map[string]string, vmString string) (string, bool) {
	for k, v := range allocatedIPs {
		if v == vmString {
			return k, false
		}
	}

	for _, cidr := range *parsedCIDRs {
		ip := cidr.IP
		for ip := ip.Mask(cidr.Mask); cidr.Contains(ip); inc(ip) {
			if _, ok := allocatedIPs[ip.String()]; !ok {
				return ip.String(), true
			}
		}
	}
	return "", false
}

//  http://play.golang.org/p/m8TNTtygK0
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
