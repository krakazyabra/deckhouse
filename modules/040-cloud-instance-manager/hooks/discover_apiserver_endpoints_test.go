package hooks

import (
	. "github.com/deckhouse/deckhouse/testing/hooks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Modules :: cloud-instance-manager :: hooks :: discover_apiserver_endpoints ::", func() {
	const (
		stateSingleAddress = `
apiVersion: v1
kind: Endpoints
metadata:
  name: kubernetes
  namespace: default
subsets:
- addresses:
  - ip: 10.0.3.192
  ports:
  - name: https
    port: 6443
    protocol: TCP
`

		stateMultipleAddresses = `
apiVersion: v1
kind: Endpoints
metadata:
  name: kubernetes
  namespace: default
subsets:
- addresses:
  - ip: 10.0.3.192
  - ip: 10.0.3.193
  - ip: 10.0.3.194
  ports:
  - name: https
    port: 6443
    protocol: TCP
`

		stateMultupleAddressesWithDifferentPorts = `
apiVersion: v1
kind: Endpoints
metadata:
  name: kubernetes
  namespace: default
subsets:
- addresses:
  - ip: 10.0.3.192
  - ip: 10.0.3.193
  ports:
  - name: https
    port: 6443
    protocol: TCP
- addresses:
  - ip: 10.0.3.194
  ports:
  - name: https
    port: 6444
    protocol: TCP
`
	)

	f := HookExecutionConfigInit(`{"cloudInstanceManager":{"internal": {}}}`, `{}`)

	Context("Endpoint default/kubernetes has single address in .subsets[]", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateSingleAddress))
			f.RunHook()
		})

		It("`cloudInstanceManager.internal.clusterMasterAddresses` must be ['10.0.3.192:6443']", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("cloudInstanceManager.internal.clusterMasterAddresses").String()).To(MatchJSON(`["10.0.3.192:6443"]`))
		})

		Context("Someone added additional addresses to .subsets[]", func() {
			BeforeEach(func() {
				f.BindingContexts.Set(f.KubeStateSet(stateMultipleAddresses))
				f.RunHook()
			})

			It("`cloudInstanceManager.internal.clusterMasterAddresses` must be ['10.0.3.192:6443','10.0.3.193:6443','10.0.3.194:6443']", func() {
				Expect(f).To(ExecuteSuccessfully())
				Expect(f.ValuesGet("cloudInstanceManager.internal.clusterMasterAddresses").String()).To(MatchJSON(`["10.0.3.192:6443","10.0.3.193:6443","10.0.3.194:6443"]`))
			})

			Context("Someone added address with different port", func() {
				BeforeEach(func() {
					f.BindingContexts.Set(f.KubeStateSet(stateMultupleAddressesWithDifferentPorts))
					f.RunHook()
				})

				It("`cloudInstanceManager.internal.clusterMasterAddresses` must be ['10.0.3.192:6443','10.0.3.193:6443','10.0.3.194:6444']", func() {
					Expect(f).To(ExecuteSuccessfully())
					Expect(f.ValuesGet("cloudInstanceManager.internal.clusterMasterAddresses").String()).To(MatchJSON(`["10.0.3.192:6443","10.0.3.193:6443","10.0.3.194:6444"]`))
				})
			})
		})
	})

	Context("Endpoint default/kubernetes has multiple addresses in .subsets[]", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateMultipleAddresses))
			f.RunHook()
		})

		It("`cloudInstanceManager.internal.clusterMasterAddresses` must be ['10.0.3.192:6443','10.0.3.193:6443','10.0.3.194:6443']", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("cloudInstanceManager.internal.clusterMasterAddresses").String()).To(MatchJSON(`["10.0.3.192:6443","10.0.3.193:6443","10.0.3.194:6443"]`))
		})

		Context("Someone set number of addresses in .subsets[] to one", func() {
			BeforeEach(func() {
				f.BindingContexts.Set(f.KubeStateSet(stateSingleAddress))
				f.RunHook()
			})

			It("`cloudInstanceManager.internal.clusterMasterAddresses` must be ['10.0.3.192:6443']", func() {
				Expect(f).To(ExecuteSuccessfully())
				Expect(f.ValuesGet("cloudInstanceManager.internal.clusterMasterAddresses").String()).To(MatchJSON(`["10.0.3.192:6443"]`))
			})
		})
	})
})
