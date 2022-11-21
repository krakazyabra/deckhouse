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
	"github.com/deckhouse/deckhouse/testing/library/object_store"
)

var _ = Describe("Modules :: virtualization :: hooks :: ipam ::", func() {
	f := HookExecutionConfigInit(initValuesString, initConfigValuesString)
	f.RegisterCRD("deckhouse.io", "v1alpha1", "VirtualMachineIPAddressLease", true)
	f.RegisterCRD("deckhouse.io", "v1alpha1", "VirtualMachine", true)

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

	Context("IPAM", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(
				f.KubeStateSet(`
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineIPAddressLease
metadata:
  name: ip-10-10-10-0
  namespace: ns1
spec:
  vmName: existing-vm
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachine
metadata:
  name: existing-vm
  namespace: ns1
status:
  ipAddress: 10.10.10.10
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineIPAddressLease
metadata:
  name: ip-10-10-10-1
  namespace: ns1
spec:
  vmName: removed-vm
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineIPAddressLease
metadata:
  name: ip-10-10-10-2
  namespace: ns1
spec:
  static: true
  vmName: missing-vm
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineIPAddressLease
metadata:
  name: ip-10-10-10-123
  namespace: ns1
spec:
  vmName: missing-vm2
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm1
  namespace: ns2
spec:
  running: true
  staticIPAddress: 10.10.10.1
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm2
  namespace: ns2
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm3
  namespace: ns2
`),
			)
			f.RunHook()
		})

		It("Manages VirtualMachineIPAddressLeases", func() {
			Expect(f).To(ExecuteSuccessfully())
			var claim object_store.KubeObject
			var vm object_store.KubeObject

			By("Checking existing VM, VirtualMachineIPAddressLease is not static, should be kept")
			claim = f.KubernetesResource("VirtualMachineIPAddressLease", "ns1", "ip-10-10-10-0")
			Expect(claim).To(Not(BeEmpty()))
			Expect(claim.Field(`spec.static`).Bool()).To(BeFalse())
			Expect(claim.Field(`spec.vmName`).String()).To(Equal("existing-vm"))

			By("Checking VM which was removed, should remove VirtualMachineIPAddressLease as well")
			claim = f.KubernetesResource("VirtualMachineIPAddressLease", "ns1", "ip-10-10-10-1")
			Expect(claim).To(BeEmpty())

			By("Checking VM which was removed, but VirtualMachineIPAddressLease is static, should be kept")
			claim = f.KubernetesResource("VirtualMachineIPAddressLease", "ns1", "ip-10-10-10-2")
			Expect(claim).To(Not(BeEmpty()))
			Expect(claim.Field(`spec.static`).Bool()).To(BeTrue())
			Expect(claim.Field(`spec.vmName`).String()).To(BeEmpty())

			By("Checking new VM with static IP address assigned, should allocate requested address")
			claim = f.KubernetesResource("VirtualMachineIPAddressLease", "ns2", "ip-10-10-10-1")
			Expect(claim).To(Not(BeEmpty()))
			Expect(claim.Field(`spec.static`).Bool()).To(BeTrue())
			Expect(claim.Field(`spec.vmName`).String()).To(Equal("vm1"))
			vm = f.KubernetesResource("VirtualMachine", "ns2", "vm1")
			Expect(vm).To(Not(BeEmpty()))
			Expect(vm.Field(`status.ipAddress`).String()).To(Equal("10.10.10.1"))

			By("Checking new VM without static IP address assigned, should allocate a new one")
			claim = f.KubernetesResource("VirtualMachineIPAddressLease", "ns2", "ip-10-10-10-3")
			Expect(claim).To(Not(BeEmpty()))
			Expect(claim.Field(`spec.static`).Bool()).To(BeFalse())
			Expect(claim.Field(`spec.vmName`).String()).To(Equal("vm2"))
			vm = f.KubernetesResource("VirtualMachine", "ns2", "vm2")
			Expect(vm).To(Not(BeEmpty()))
			Expect(vm.Field(`status.ipAddress`).String()).To(Equal("10.10.10.3"))

			By("Checking new VM without static IP address assigned, should allocate a new one")
			claim = f.KubernetesResource("VirtualMachineIPAddressLease", "ns2", "ip-10-10-10-4")
			Expect(claim).To(Not(BeEmpty()))
			Expect(claim.Field(`spec.static`).Bool()).To(BeFalse())
			Expect(claim.Field(`spec.vmName`).String()).To(Equal("vm3"))
			vm = f.KubernetesResource("VirtualMachine", "ns2", "vm3")
			Expect(vm).To(Not(BeEmpty()))
			Expect(vm.Field(`status.ipAddress`).String()).To(Equal("10.10.10.4"))
		})
	})

})
