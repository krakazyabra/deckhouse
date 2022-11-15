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

package template_tests

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/helm"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "")
}

const (
	globalValues = `
  enabledModules: ["vertical-pod-autoscaler-crd"]
  highAvailability: true
  modules:
    placement: {}
  discovery:
    kubernetesVersion: 1.21.9
    d8SpecificNodeCountByRole:
      worker: 3
      master: 3
`
	moduleValues = `
  internal:
    webhookCert:
      ca: |
        -----BEGIN CERTIFICATE-----
        MIIBbTCCARSgAwIBAgIUNY8AHPMngGERxYdy9OQvB/C5Z2swCgYIKoZIzj0EAwIw
        FTETMBEGA1UEAxMKbGluc3Rvci1jYTAeFw0yMjAyMDgxOTQwMDBaFw0zMjAyMDYx
        OTQwMDBaMBUxEzARBgNVBAMTCmxpbnN0b3ItY2EwWTATBgcqhkjOPQIBBggqhkjO
        PQMBBwNCAAR/god/1bNYEJbbI4Ss3eDXxco6ztt/nTA71AcYUF0+8KaqqEgB1b4d
        h6BeqkHFtGcDLdFu4DIVlTcrsVNgzcVwo0IwQDAOBgNVHQ8BAf8EBAMCAQYwDwYD
        VR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUX/S16dEkvqDWVE3i07jOMYBxhtAwCgYI
        KoZIzj0EAwIDRwAwRAIgcNKc5Bt0Fd5z4jFL3LXyaQtQeinjYZiMcqLMrGv+NNoC
        IDJid8dT06cHhi8ltGgLZzXGw25qOu5oZSSJIRw6+QcZ
        -----END CERTIFICATE-----
      cert: |
        -----BEGIN CERTIFICATE-----
        MIIBsDCCAVWgAwIBAgIUR6gMYo0dyTWRiEKMnDYmAJeW7ZwwCgYIKoZIzj0EAwIw
        FTETMBEGA1UEAxMKbGluc3Rvci1jYTAeFw0yMjAyMDgxOTQwMDBaFw0yMzAyMDgx
        OTQwMDBaMBkxFzAVBgNVBAMTDmxpbnN0b3ItY2xpZW50MFkwEwYHKoZIzj0CAQYI
        KoZIzj0DAQcDQgAEalDjr7NfrwdjoSh1qo5vfYccFjZQxMTEy+rVH+pSEIMgp+ef
        Ipz24bDQZ/6qwZbpbiT1lywYVWDpWVxeFcV+FaN/MH0wDgYDVR0PAQH/BAQDAgWg
        MB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMB0G
        A1UdDgQWBBRFObmL7G6CSOLmpI2Tog79nkyzEjAfBgNVHSMEGDAWgBRf9LXp0SS+
        oNZUTeLTuM4xgHGG0DAKBggqhkjOPQQDAgNJADBGAiEAgZAQv6TBsg3PGji2u6MO
        /V46YliV5HVbtEaZG1l/10sCIQCwQOC1/9+2mOOypS6lYywJAo/l+MlbZMWITySC
        A8aK1g==
        -----END CERTIFICATE-----
      key: |
        -----BEGIN EC PRIVATE KEY-----
        MHcCAQEEIPFImbnfYGVkjAoMJrT91lAzX122Z53AXh5bFwCnNVsfoAoGCCqGSM49
        AwEHoUQDQgAEalDjr7NfrwdjoSh1qo5vfYccFjZQxMTEy+rVH+pSEIMgp+efIpz2
        4bDQZ/6qwZbpbiT1lywYVWDpWVxeFcV+FQ==
        -----END EC PRIVATE KEY-----`
)

var _ = Describe("Module :: kubevirt :: helm template ::", func() {
	f := SetupHelmConfig(``)

	Context("Standard setup with SSL", func() {
		BeforeEach(func() {
			f.ValuesSetFromYaml("global", globalValues)
			f.ValuesSet("global.modulesImages", GetModulesImages())
			f.ValuesSetFromYaml("virtualization", moduleValues)
			f.HelmRender()
		})

		It("Everything must render properly", func() {
			Expect(f.RenderError).ShouldNot(HaveOccurred())
		})

	})
})
