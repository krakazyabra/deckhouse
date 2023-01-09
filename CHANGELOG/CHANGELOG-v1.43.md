# Changelog v1.43

## [MALFORMED]


 - #3381 unknown section "global"
 - #3435 missing section, missing summary, missing type, unknown section ""

## Know before update


 - Components will be restarted in the following modules:
    * every module using `csi-external-attacher`, `csi-external-provisioner`, `csi-external-resizer`, `csi-external-snapshotter`, `csi-livenessprobe`, `csi-node-registrar`, `kube-rbac-proxy`
    * `basic-auth`
    * `chrony`
    * `cilium-hubble`
    * `cloud-provider-aws`
    * `cloud-provider-azure`
    * `cloud-provider-gcp`
    * `cloud-provider-openstack`
    * `cloud-provider-vsphere`
    * `cni-cilium`
    * `control-plane-manager`
    * `dashboard`
    * `deckhouse`
    * `deckhouse-web`
    * `extended-monitoring`
    * `flant-integration`
    * `ingress-nginx`
    * `istio`
    * `keepalived`
    * `kube-dns`
    * `kube-proxy`
    * `linstor`
    * `log-shipper`
    * `metallb`
    * `monitoring-kubernetes`
    * `monitoring-ping`
    * `network-gateway`
    * `node-local-dns`
    * `node-manager`
    * `openvpn`
    * `prometheus`
    * `registrypackages`
    * `terraform-manager`
    * `upmeter`
    * `user-authn`
    * `user-authz`

## Fixes


 - **[candi]** Bump shell-operator to 1.1.3 & update base images to mitigate found CVEs [#3335](https://github.com/deckhouse/deckhouse/pull/3335)
    Components will be restarted in the following modules:
    * every module using `csi-external-attacher`, `csi-external-provisioner`, `csi-external-resizer`, `csi-external-snapshotter`, `csi-livenessprobe`, `csi-node-registrar`, `kube-rbac-proxy`
    * `basic-auth`
    * `chrony`
    * `cilium-hubble`
    * `cloud-provider-aws`
    * `cloud-provider-azure`
    * `cloud-provider-gcp`
    * `cloud-provider-openstack`
    * `cloud-provider-vsphere`
    * `cni-cilium`
    * `control-plane-manager`
    * `dashboard`
    * `deckhouse`
    * `deckhouse-web`
    * `extended-monitoring`
    * `flant-integration`
    * `ingress-nginx`
    * `istio`
    * `keepalived`
    * `kube-dns`
    * `kube-proxy`
    * `linstor`
    * `log-shipper`
    * `metallb`
    * `monitoring-kubernetes`
    * `monitoring-ping`
    * `network-gateway`
    * `node-local-dns`
    * `node-manager`
    * `openvpn`
    * `prometheus`
    * `registrypackages`
    * `terraform-manager`
    * `upmeter`
    * `user-authn`
    * `user-authz`
 - **[prometheus]** make each grafana dashboard unique by UID [#3255](https://github.com/deckhouse/deckhouse/pull/3255)

## Chore


 - **[deckhouse]** Changed deckhouse_registry hook to get registry data form secret docker-registry. The global values of the registry is refactored for all modules. [#3193](https://github.com/deckhouse/deckhouse/pull/3193)
 - **[terraform-manager]** Rebuild image only if openapi spec is changed [#3432](https://github.com/deckhouse/deckhouse/pull/3432)

