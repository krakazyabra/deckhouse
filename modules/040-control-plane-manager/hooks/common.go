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

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/shell-operator/pkg/kube_events_manager/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"

	"github.com/deckhouse/deckhouse/go_lib/certificate"
	"github.com/deckhouse/deckhouse/go_lib/dependency"
	"github.com/deckhouse/deckhouse/go_lib/dependency/etcd"
)

const (
	moduleQueue        = "/modules/control-plane-manager"
	defaultEtcdMaxSize = 2 * 1024 * 1024 * 1024 // 2GB
)

type maintenanceEtc struct {
	endpoint  string
	maxDbSize int64
}

func getETCDClient(input *go_hook.HookInput, dc dependency.Container, endpoints []string) (etcd.Client, error) {
	snap := input.Snapshots["etcd-certificate"]
	if len(snap) == 0 {
		return nil, fmt.Errorf("etcd credentials not found")
	}

	cert := snap[0].(certificate.Certificate)

	if cert.CA == "" || cert.Cert == "" || cert.Key == "" {
		return nil, fmt.Errorf("etcd credentials not found")
	}

	caCert, clientCert, err := certificate.ParseCertificatesFromPEM(cert.CA, cert.Cert, cert.Key)
	if err != nil {
		return nil, err
	}

	return dc.GetEtcdClient(endpoints, etcd.WithClientCert(clientCert, caCert), etcd.WithInsecureSkipVerify())
}

func getETCDClientFromSnapshots(input *go_hook.HookInput, dc dependency.Container) (etcd.Client, error) {
	snap := input.Snapshots["etcd_pods"]

	if len(snap) == 0 {
		return nil, fmt.Errorf("empty etc endpoints")
	}

	endpoints := make([]string, 0, len(snap))

	for _, e := range snap {
		endpoints = append(endpoints, e.(string))
	}

	return getETCDClient(input, dc, endpoints)
}

var (
	etcdSecretK8sConfig = go_hook.KubernetesConfig{
		Name:       "etcd-certificate",
		ApiVersion: "v1",
		Kind:       "Secret",
		NamespaceSelector: &types.NamespaceSelector{
			NameSelector: &types.NameSelector{
				MatchNames: []string{"kube-system"},
			},
		},
		NameSelector:                 &types.NameSelector{MatchNames: []string{"d8-pki"}},
		ExecuteHookOnSynchronization: pointer.BoolPtr(false),
		ExecuteHookOnEvents:          pointer.BoolPtr(false),
		FilterFunc:                   syncEtcdFilter,
	}

	etcdMaintenanceConfig = getEtcdEndpointConfig(func(unstructured *unstructured.Unstructured) (go_hook.FilterResult, error) {
		var pod corev1.Pod

		err := sdk.FromUnstructured(unstructured, &pod)
		if err != nil {
			return nil, err
		}

		var ip string
		if pod.Spec.HostNetwork {
			ip = pod.Status.HostIP
		} else {
			ip = pod.Status.PodIP
		}

		for _, c := range pod.Spec.Containers {
			if c.Name != "etcd" {
				continue
			}

			for _, arg := range c.Command {
				if
			}
		}

		return etcdEndpointString(ip), nil
	})
)

func getEtcdEndpointConfig(filter go_hook.FilterFunc) go_hook.KubernetesConfig {
	return go_hook.KubernetesConfig{
		Name:       "etcd_endpoints",
		ApiVersion: "v1",
		Kind:       "Pod",
		NamespaceSelector: &types.NamespaceSelector{
			NameSelector: &types.NameSelector{
				MatchNames: []string{"kube-system"},
			},
		},
		LabelSelector: &v1.LabelSelector{
			MatchLabels: map[string]string{
				"component": "etcd",
				"tier":      "control-plane",
			},
		},
		FieldSelector: &types.FieldSelector{
			MatchExpressions: []types.FieldSelectorRequirement{
				{
					Field:    "status.phase",
					Operator: "Equals",
					Value:    "Running",
				},
			},
		},
		FilterFunc: filter,
	}
}

func etcdEndpointString(ip string) string {
	return fmt.Sprintf("https://%s:2379", ip)
}

func syncEtcdFilter(unstructured *unstructured.Unstructured) (go_hook.FilterResult, error) {
	var sec corev1.Secret

	err := sdk.FromUnstructured(unstructured, &sec)
	if err != nil {
		return nil, err
	}

	var cert certificate.Certificate

	if ca, ok := sec.Data["etcd-ca.crt"]; ok {
		cert.CA = string(ca)
		cert.Cert = string(ca)
	}

	if key, ok := sec.Data["etcd-ca.key"]; ok {
		cert.Key = string(key)
	}

	return cert, nil
}
