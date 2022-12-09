/*
Copyright 2022 Flant JSC

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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Modules :: virtualization :: hooks :: vm_handler ::", func() {
	f := HookExecutionConfigInit(initValuesString, initConfigValuesString)
	f.RegisterCRD("deckhouse.io", "v1alpha1", "VirtualMachineDisk", true)
	f.RegisterCRD("deckhouse.io", "v1alpha1", "VirtualMachine", true)
	f.RegisterCRD("deckhouse.io", "v1alpha1", "VirtualMachineIPAddressClaim", true)
	f.RegisterCRD("kubevirt.io", "v1", "VirtualMachine", true)

	// Set Kind for binding.
	vmHandlerHookConfig.Kubernetes[0].Kind = "VirtualMachine"
	vmHandlerHookConfig.Kubernetes[0].ApiVersion = "kubevirt.io/v1"

	Context("Empty cluster", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(
				f.KubeStateSet(``),
			)
			f.RunHook()
		})

		It("ExecuteSuccessfully", func() {
			Expect(f).To(ExecuteSuccessfully())
		})
	})

	Context("VMS creation", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(
				f.KubeStateSet(`
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineIPAddressClaim
metadata:
  name: mysql
  namespace: default
spec:
  static: false
  address: 10.10.10.2
  leaseName: ip-10-10-10-2
status:
  phase: Bound
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm1
  namespace: default
spec:
  running: true
  resources:
    memory: 512M
    cpu: "1"
  userName: admin
  sshPublicKey: "ssh-rsa asdasdkflkasddf..."
  ipAddressClaimName: mysql
  bootDisk:
    source:
      kind: ClusterVirtualMachineImage
      name: ubuntu-20.04
    size: 10Gi
    storageClassName: linstor-slow
    autoDelete: true
  cloudInit:
    userData: |-
      chpasswd: { expire: False }
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineDisk
metadata:
  name: mydata
  namespace: ns1
spec:
  source:
    kind: ClusterVirtualMachineImage
    name: centos-7
  storageClassName: linstor-slow
  size: 10Gi
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm2
  namespace: default
spec:
  running: true
  resources:
    memory: 512M
    cpu: "1"
  userName: admin
  sshPublicKey: "ssh-rsa asdasdkflkasddf..."
  bootDisk:
    source:
      kind: ClusterVirtualMachineImage
      name: ubuntu-20.04
    size: 10Gi
    storageClassName: linstor-slow
    ephemeral: false
  cloudInit:
    userData: |-
      chpasswd: { expire: False }
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineIPAddressClaim
metadata:
  name: vm2
  namespace: default
spec:
  static: false
  address: 10.10.10.3
  leaseName: ip-10-10-10-3
status:
  phase: Bound
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineDisk
metadata:
  name: vm2-boot
  namespace: default
spec:
  source:
    kind: ClusterVirtualMachineImage
    name: ubuntu-20.04
  storageClassName: linstor-slow
  size: 10Gi
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineDisk
metadata:
  name: vm3-boot
  namespace: default
spec:
  source:
    kind: ClusterVirtualMachineImage
    name: ubuntu-20.04
  storageClassName: linstor-slow
  size: 10Gi
status:
  ephemeral: false
  vmName: vm3
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineDisk
metadata:
  name: vm4-boot
  namespace: default
spec:
  source:
    kind: ClusterVirtualMachineImage
    name: ubuntu-20.04
  storageClassName: linstor-slow
  size: 10Gi
status:
  ephemeral: true
  vmName: vm4
`),
			)
			f.RunHook()
		})

		It("Creates VirtualMachine and boot Disk", func() {
			Expect(f).To(ExecuteSuccessfully())
			disk := f.KubernetesResource("VirtualMachineDisk", "default", "vm1-boot")
			Expect(disk).To(Not(BeEmpty()))
			Expect(disk.Field(`status.vmName`).String()).To(Equal("vm1"))
			Expect(disk.Field(`status.ephemeral`).Bool()).To(BeTrue())
			vm := f.KubernetesResource("virtualmachines.kubevirt.io", "default", "vm1")
			Expect(vm).To(Not(BeEmpty()))
			Expect(vm.Field(`apiVersion`).String()).To(Equal("kubevirt.io/v1"))

			By("should update fields for existing disk")
			disk2 := f.KubernetesResource("VirtualMachineDisk", "default", "vm2-boot")
			Expect(disk2).To(Not(BeEmpty()))
			Expect(disk2.Field(`status.vmName`).String()).To(Equal("vm2"))
			Expect(disk2.Field(`status.ephemeral`).Bool()).To(BeFalse())

			By("should keep ephemeral disks")
			disk3 := f.KubernetesResource("VirtualMachineDisk", "default", "vm3-boot")
			Expect(disk3).To(Not(BeEmpty()))
			Expect(disk3.Field(`status.vmName`).String()).To(Equal(""))

			By("should delete ephemeral disks")
			disk4 := f.KubernetesResource("VirtualMachineDisk", "default", "vm4-boot")
			Expect(disk4).To(BeEmpty())
		})
	})

})
