package hooks

import (
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Istio hooks :: federation_discovery_ingressgateways ::", func() {
	f := HookExecutionConfigInit(`{"istio":{"federation":{}}}`, "")
	f.RegisterCRD("deckhouse.io", "v1alpha1", "IstioFederation", false)

	Context("Empty cluster and minimal settings", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(``))
			f.RunHook()
		})

		It("Hook must execute successfully", func() {
			Expect(f).To(ExecuteSuccessfully())

			stderrBuff := string(f.Session.Err.Contents())
			Expect(stderrBuff).To(Equal(""))
		})
	})

	Context("Empty cluster, minimal settings and federation is enabled", func() {
		BeforeEach(func() {
			f.ValuesSet("istio.federation.enabled", true)
			f.BindingContexts.Set(f.KubeStateSet(``))
			f.RunHook()
		})

		It("Hook must execute successfully", func() {
			Expect(f).To(ExecuteSuccessfully())

			stderrBuff := string(f.Session.Err.Contents())
			Expect(stderrBuff).To(Equal(""))
		})
	})

	Context("Proper federations only", func() {
		BeforeEach(func() {
			f.ValuesSet(`istio.federation.enabled`, true)
			f.BindingContexts.Set(f.KubeStateSet(`
---
apiVersion: deckhouse.io/v1alpha1
kind: IstioFederation
metadata:
  name: proper-federation-0
spec:
  trustDomain: "p.f0"
  federationMetadata:
    endpoint: "file:///tmp/proper-federation-0/"
status: {}
---
apiVersion: deckhouse.io/v1alpha1
kind: IstioFederation
metadata:
  name: proper-federation-1
spec:
  trustDomain: "p.f1"
  federationMetadata:
    endpoint: "file:///tmp/proper-federation-1/"
status:
  metadataCache:
    ingressGateways:
    - {"address": "some-outdated.host", "port": 111}
---
apiVersion: deckhouse.io/v1alpha1
kind: IstioFederation
metadata:
  name: proper-federation-2
spec:
  trustDomain: "p.f2"
  federationMetadata:
    endpoint: "file:///tmp/proper-federation-2/"
status:
  metadataCache:
    ingressGateways:
    - {"address": "some-actual.host-1", "port": 111}
    - {"address": "some-actual.host-2", "port": 111}
`))
			_ = os.Mkdir("/tmp/proper-federation-0", 0755)
			ioutil.WriteFile("/tmp/proper-federation-0/metadata-ingressgateways", []byte(`
{
  "ingressGateways": [
    {"address": "a.b.c", "port": 123},
    {"address": "1.2.3.4", "port": 234}
  ]
}
`), 0644)
			_ = os.Mkdir("/tmp/proper-federation-1", 0755)
			ioutil.WriteFile("/tmp/proper-federation-1/metadata-ingressgateways", []byte(`
{
  "ingressGateways": [
    {"address": "some-actual.host", "port": 111}
  ]
}
`), 0644)
			_ = os.Mkdir("/tmp/proper-federation-2", 0755)
			ioutil.WriteFile("/tmp/proper-federation-2/metadata-ingressgateways", []byte(`
{
  "ingressGateways": [
    {"address": "some-actual.host-2", "port": 111},
    {"address": "some-actual.host-1", "port": 111}
  ]
}
`), 0644)

			f.RunHook()
		})

		It("Hook must execute successfully", func() {
			Expect(f).To(ExecuteSuccessfully())

			stderrBuff := string(f.Session.Err.Contents())
			Expect(stderrBuff).To(Equal(""))

			t0, err := time.Parse(time.RFC3339, f.KubernetesGlobalResource("IstioFederation", "proper-federation-0").Field("status.metadataCache.ingressGatewaysLastFetchTimestamp").String())
			Expect(err).ShouldNot(HaveOccurred())
			t1, err := time.Parse(time.RFC3339, f.KubernetesGlobalResource("IstioFederation", "proper-federation-1").Field("status.metadataCache.ingressGatewaysLastFetchTimestamp").String())
			Expect(err).ShouldNot(HaveOccurred())

			Expect(t0).Should(BeTemporally("~", time.Now().UTC(), time.Minute))
			Expect(t1).Should(BeTemporally("~", time.Now().UTC(), time.Minute))

			Expect(f.KubernetesGlobalResource("IstioFederation", "proper-federation-0").Field("status.metadataCache.ingressGateways").String()).To(MatchJSON(`
            [
              {"address": "1.2.3.4", "port": 234},
              {"address": "a.b.c", "port": 123}
            ]
`))
			Expect(f.KubernetesGlobalResource("IstioFederation", "proper-federation-1").Field("status.metadataCache.ingressGateways").String()).To(MatchJSON(`
            [
              {"address": "some-actual.host", "port": 111}
            ]
`))
			Expect(f.KubernetesGlobalResource("IstioFederation", "proper-federation-2").Field("status.metadataCache.ingressGateways").String()).To(MatchJSON(`
            [
              {"address": "some-actual.host-1", "port": 111},
              {"address": "some-actual.host-2", "port": 111}
            ]
`))
			// Expect(f.KubernetesResourcePatch).To(HaveLen(2))
		})
	})

	Context("Improper federation", func() {
		BeforeEach(func() {
			f.ValuesSet(`istio.federation.enabled`, true)
			f.BindingContexts.Set(f.KubeStateSet(`
---
apiVersion: deckhouse.io/v1alpha1
kind: IstioFederation
metadata:
  name: improper-federation-0
spec:
  trustDomain: "i.f0"
  federationMetadata:
    endpoint: "https://some-improper-hostname-0/federation/"
`))

			f.RunHook()
		})

		It("Hook must execute successfully with proper warnings", func() {
			Expect(f).To(ExecuteSuccessfully())

			stderrBuff := string(f.Session.Err.Contents())
			Expect(stderrBuff).Should(ContainSubstring(`ERROR: Cannot fetch ingressgateways metadata endpoint https://some-improper-hostname-0/federation/metadata-ingressgateways for IstioFederation improper-federation-0.`))
		})
	})
})
