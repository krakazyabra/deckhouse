package hooks

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/deckhouse/deckhouse/modules/490-virtualization/api/v1alpha1"
	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	virtv1 "kubevirt.io/api/core/v1"
)

func nameToIP(name string) string {
	a := strings.Split(name, "-")
	if a[0] != "ip" {
		return ""
	}
	// IPv4 address
	if len(a) == 5 {
		return fmt.Sprintf("%s.%s.%s.%s", a[1], a[2], a[3], a[4])
	}
	// IPv6 address
	if len(a) == 9 {
		return fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s", a[1], a[2], a[3], a[4], a[5], a[6], a[7], a[8])
	}
	return ""
}

func ipToName(ip string) string {
	addr := net.ParseIP(ip)
	if addr.To4() != nil {
		// IPv4 address
		return "ip-" + strings.ReplaceAll(addr.String(), ".", "-")
	}
	if addr.To16() != nil {
		// IPv6 address
		dst := make([]byte, hex.EncodedLen(len(addr)))
		_ = hex.Encode(dst, addr)
		return fmt.Sprintf("ip-" +
			string(dst[0:4]) + "-" +
			string(dst[4:8]) + "-" +
			string(dst[8:12]) + "-" +
			string(dst[12:16]) + "-" +
			string(dst[16:20]) + "-" +
			string(dst[20:24]) + "-" +
			string(dst[24:28]) + "-" +
			string(dst[28:]))
	}
	return ""
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

func getIPClaim(snapshot *[]go_hook.FilterResult, namespace, name string) *VirtualMachineIPAddressClaimSnapshot {
	for _, dRaw := range *snapshot {
		claim := dRaw.(*VirtualMachineIPAddressClaimSnapshot)
		if claim.Namespace == namespace && claim.Name == name {
			return claim
		}
	}
	return nil
}

func getStorageClass(snapshot *[]go_hook.FilterResult, name string) *StorageClassSnapshot {
	for _, dRaw := range *snapshot {
		storageClass := dRaw.(*StorageClassSnapshot)
		if name != "" {
			if storageClass.Name == name {
				return storageClass
			}
		} else {
			if storageClass.DefaultStorageClass {
				return storageClass
			}
		}
	}
	return nil
}

func getClusterImage(snapshot *[]go_hook.FilterResult, name string) *ClusterVirtualMachineImageSnapshot {
	for _, dRaw := range *snapshot {
		clusterImage := dRaw.(*ClusterVirtualMachineImageSnapshot)
		if clusterImage.Name == name {
			return clusterImage
		}
	}
	return nil
}

func getDisk(snapshot *[]go_hook.FilterResult, namespace, name string) *VirtualMachineDiskSnapshot {
	for _, dRaw := range *snapshot {
		disk := dRaw.(*VirtualMachineDiskSnapshot)
		if disk.Namespace == namespace && disk.Name == name {
			return disk
		}
	}
	return nil
}

func getDataVolume(snapshot *[]go_hook.FilterResult, namespace, name string) *DataVolumeSnapshot {
	for _, dRaw := range *snapshot {
		dataVolume := dRaw.(*DataVolumeSnapshot)
		if dataVolume.Namespace == namespace && dataVolume.Name == name {
			return dataVolume
		}
	}
	return nil
}

func getPVC(snapshot *[]go_hook.FilterResult, namespace, name string) *PVCSnapshot {
	for _, dRaw := range *snapshot {
		pvc := dRaw.(*PVCSnapshot)
		if pvc.Namespace == namespace && pvc.Name == name {
			return pvc
		}
	}
	return nil
}