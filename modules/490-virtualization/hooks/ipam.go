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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/deckhouse/deckhouse/modules/490-virtualization/api/v1alpha1"
)

const (
	ipLeasesSnapshot = "ipamLeases"
	ipClaimsSnapshot = "ipamClaims"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: "/modules/virtualization/ipam-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       ipLeasesSnapshot,
			ApiVersion: gv,
			Kind:       "VirtualMachineIPAddressLease",
			FilterFunc: applyVirtualMachineIPAddressLeaseFilter,
		},
		{
			Name:       ipClaimsSnapshot,
			ApiVersion: gv,
			Kind:       "VirtualMachineIPAddressClaim",
			FilterFunc: applyVirtualMachineIPAddressClaimFilter,
		},
	},
}, doIPAM)

type VirtualMachineIPAddressLeaseSnapshot struct {
	Name           string
	ClaimName      string
	ClaimNamespace string
	Address        string
	Phase          string
}

type VirtualMachineIPAddressClaimSnapshot struct {
	Name      string
	Namespace string
	LeaseName string
	Address   string
	Phase     string
	VMName    string
	Static    *bool
}

func applyVirtualMachineIPAddressLeaseFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	lease := &v1alpha1.VirtualMachineIPAddressLease{}
	leaseSnap := new(VirtualMachineIPAddressLeaseSnapshot)
	err := sdk.FromUnstructured(obj, lease)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachineIPAddressLease: %v", err)
	}

	leaseSnap.Address = nameToIP(lease.Name)
	if leaseSnap.Address == "" {
		return nil, fmt.Errorf("cannot convert VirtualMachineIPAddressLease name to IP address: %s", lease.Name)
	}

	leaseSnap.Name = lease.Name
	leaseSnap.Phase = lease.Status.Phase
	if lease.Spec.ClaimRef != nil {
		leaseSnap.ClaimName = lease.Spec.ClaimRef.Name
		leaseSnap.ClaimNamespace = lease.Spec.ClaimRef.Namespace
	}

	return leaseSnap, nil
}

func applyVirtualMachineIPAddressClaimFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	claim := &v1alpha1.VirtualMachineIPAddressClaim{}
	err := sdk.FromUnstructured(obj, claim)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachineIPAddressClaim: %v", err)
	}

	return &VirtualMachineIPAddressClaimSnapshot{
		Name:      claim.Name,
		Namespace: claim.Namespace,
		LeaseName: claim.Spec.LeaseName,
		Static:    claim.Spec.Static,
		Address:   claim.Spec.Address,
		Phase:     claim.Status.Phase,
		VMName:    claim.Status.VMName,
	}, nil
}

// doIPAM
//
// synopsis:
//   This hook performs IPAM (IP Address Management) for VirtualMachines.
//   It takes free IP addresses from `virtualization.vmCIDRs` and assigning them to Virtual Machines.
//   Every VM required static IP address obtans static IP lease with specific IP address,
//   otherwise the next free IP address assigned to the VM.
//   Additionaly this hook performs the check to make sure that requested IP address is not
// 	 assigned to other Virtual Machine.

func doIPAM(input *go_hook.HookInput) error {
	leaseSnap := input.Snapshots[ipLeasesSnapshot]
	claimSnap := input.Snapshots[ipClaimsSnapshot]
	if len(claimSnap) == 0 && len(leaseSnap) == 0 {
		input.LogEntry.Warnln("VirtualMachineIPAddressLease and VirtualMachineIPAddressClaim not found. Skip")
		return nil
	}

	allocatedIPs := make(map[string]struct{})

	// -------------------------------------
	// Handle VirtualMachineIPAddressLeases
	// -------------------------------------
	for _, sRaw := range leaseSnap {
		lease := sRaw.(*VirtualMachineIPAddressLeaseSnapshot)
		var claim *VirtualMachineIPAddressClaimSnapshot
		if lease.ClaimName != "" {
			claim = getIPClaim(&claimSnap, lease.ClaimNamespace, lease.ClaimName)
		}
		if claim == nil {
			// No claims found, we can remove lease
			input.PatchCollector.Delete(gv, "VirtualMachineIPAddressLease", "", lease.Name)
			continue
		}
		// Allocate IP address
		if _, ok := allocatedIPs[lease.Address]; ok {
			return fmt.Errorf("Duplicated IP address lease %s", lease.Address)
		}
		allocatedIPs[lease.Address] = struct{}{}
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

	// -------------------------------------
	// Handle VirtualMachineIPAddressClaims
	// -------------------------------------
	for _, sRaw := range claimSnap {
		claim := sRaw.(*VirtualMachineIPAddressClaimSnapshot)

		patch := make(map[string]interface{})
		patchSpec := make(map[string]interface{})
		patchStatus := make(map[string]interface{})
		if claim.Static == nil {
			patchSpec["static"] = true
		}

		// Check for already allocated IP address
		_, alreadyAllocated := allocatedIPs[claim.Address]
		if alreadyAllocated && claim.Phase != "Bound" && claim.Phase != "InUse" {
			// Wrong lease specified, remove leaseName field
			if claim.LeaseName != "" {
				patchSpec["leaseName"] = nil
			}
			patch := map[string]interface{}{"spec": patchSpec}
			if claim.Phase != "Conflict" {
				patch["status"] = map[string]interface{}{"phase": "Conflict"}
			}
			if len(patch) != 0 {
				input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressClaim", claim.Namespace, claim.Name, object_patch.WithSubresource("/status"))
			}
			// Stop processing conflicting claim
			continue
		}

		var lease *VirtualMachineIPAddressLeaseSnapshot

		if claim.LeaseName != "" {
			lease = getIPLease(&leaseSnap, claim.LeaseName)
		}
		if lease != nil {
			// Lease found
			leaseMatched := (lease.ClaimName == claim.Name && lease.ClaimNamespace == claim.Namespace)
			leaseWrong := (claim.Address != "" && lease.Address != claim.Address)
			if leaseWrong {
				if getIPClaim(&claimSnap, lease.ClaimNamespace, lease.ClaimName) == nil {
					// If old lease is removed, we can consider it as correct
					leaseWrong = false
				}
			}
			if leaseWrong || !leaseMatched {
				// TODO refactor this
				// Wrong lease specified, remove leaseName field
				patchSpec["leaseName"] = nil
				patch := map[string]interface{}{"spec": patchSpec}
				if !leaseMatched {
					patch["status"] = map[string]interface{}{"phase": "Conflict"}
				}
				input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressClaim", claim.Namespace, claim.Name, object_patch.WithSubresource("/status"))
				if !leaseMatched {
					// Stop processing conflicting claim
					continue
				}
			} else {
				// Correct lease specified, check fields
				if claim.Phase != "Bound" && claim.Phase != "InUse" {
					patchStatus["phase"] = "Bound"
				}
				if claim.Address == "" {
					patchSpec["address"] = lease.Address
				}
				if claim.Phase != "Bound" && claim.Phase != "InUse" {
					patch["status"] = map[string]string{"phase": "Bound"}
				}
				if len(patchStatus) != 0 {
					patch["status"] = patchStatus
				}
				if len(patchSpec) != 0 {
					patch["spec"] = patchSpec
				}
				if len(patch) != 0 {
					input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressClaim", claim.Namespace, claim.Name, object_patch.WithSubresource("/status"))
				}

				if lease.Phase != "Bound" {
					patch := map[string]interface{}{"status": map[string]interface{}{"phase": "Bound"}}
					input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressLease", "", lease.Name, object_patch.WithSubresource("/status"))
				}
				// nothing to do
				continue
			}
		}

		// Lease not found, create a new one

		var ip string
		if claim.Address != "" {
			ip = claim.Address
			if err := allocateIP(parsedCIDRs, allocatedIPs, claim.Address); err != nil {
				switch err.Error() {
				case "OutOfRange":
					input.LogEntry.Warnf("error allocating ip %s, not in CIDRs %s", ip, parsedCIDRs)
				case "Conflict":
					input.LogEntry.Warnf("error allocating ip %s, already allocated", ip)
				}
				patch := map[string]interface{}{"status": map[string]interface{}{"phase": err.Error()}}
				input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressClaim", claim.Namespace, claim.Name, object_patch.WithSubresource("/status"))
				continue
			}
		} else {
			var err error
			ip, err = allocateNewIP(parsedCIDRs, allocatedIPs)
			if err != nil {
				patch := map[string]interface{}{"status": map[string]interface{}{"phase": err.Error()}}
				input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressClaim", claim.Namespace, claim.Name, object_patch.WithSubresource("/status"))
			}
		}

		leaseName := ipToName(ip)

		newLease := &v1alpha1.VirtualMachineIPAddressLease{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VirtualMachineIPAddressLease",
				APIVersion: gv,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: leaseName,
			},
			Spec: v1alpha1.VirtualMachineIPAddressLeaseSpec{
				ClaimRef: &corev1.ObjectReference{
					Name:      claim.Name,
					Namespace: claim.Namespace,
				},
			},
		}
		input.PatchCollector.Create(newLease)

		patch = map[string]interface{}{"status": map[string]interface{}{"phase": "Bound"}}
		input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressLease", "", leaseName, object_patch.WithSubresource("/status"))

		patchSpec = make(map[string]interface{})
		if claim.Static == nil {
			patchSpec["static"] = true
		}
		if claim.LeaseName != leaseName {
			patchSpec["leaseName"] = leaseName
		}
		if claim.Address != ip {
			patchSpec["address"] = ip
		}
		if len(patchSpec) != 0 {
			patch["spec"] = patchSpec
		}
		input.PatchCollector.MergePatch(patch, gv, "VirtualMachineIPAddressClaim", claim.Namespace, claim.Name, object_patch.WithSubresource("/status"))
	}

	return nil
}

func allocateIP(parsedCIDRs []*net.IPNet, allocatedIPs map[string]struct{}, address string) error {
	ip := net.ParseIP(address)
	if _, ok := allocatedIPs[ip.String()]; !ok {
		for _, cidr := range parsedCIDRs {
			if cidr.Contains(ip) {
				allocatedIPs[ip.String()] = struct{}{}
				return nil
			}
		}
		return fmt.Errorf("OutOfRange")
	}
	return fmt.Errorf("Conflict")
}

func allocateNewIP(parsedCIDRs []*net.IPNet, allocatedIPs map[string]struct{}) (string, error) {
	for _, cidr := range parsedCIDRs {
		ip := cidr.IP
		for ip := ip.Mask(cidr.Mask); cidr.Contains(ip); inc(ip) {
			if _, ok := allocatedIPs[ip.String()]; !ok {
				allocatedIPs[ip.String()] = struct{}{}
				return ip.String(), nil
			}
		}
	}
	// controller_utils.go:260] Error while processing Node Add/Delete: failed to allocate cidr from cluster cidr at idx:0: CIDR allocation failed; there are no remaining CIDRs left to allocate in the accepted range
	return "", fmt.Errorf("NoRemaingingCIDRs")
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

func getIPLease(snapshot *[]go_hook.FilterResult, name string) *VirtualMachineIPAddressLeaseSnapshot {
	for _, dRaw := range *snapshot {
		lease := dRaw.(*VirtualMachineIPAddressLeaseSnapshot)
		if lease.Name == name {
			return lease
		}
	}
	return nil
}
