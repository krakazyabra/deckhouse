# Patches

#### `000-bundle-extra-images.patch`

Iternal patch which adds `libguestfs`, `virt-exportserver` and `virt-exportproxy`
to images bundle target.

#### `001-allow-specify-image.patch`

Ability to specify component images using environment variables

- https://github.com/kubevirt/kubevirt/pull/8390

#### `002-deckhouse-registry.patch`

Internal patch which adds deckhouse ImagePullSecrets to kubevirt VMs

- https://github.com/kubevirt/kubevirt/issues/8302

#### `003-network-aware-livemigration.patch`

Allow live-migration for pod network in bridge mode

- https://github.com/kubevirt/community/pull/182
- https://github.com/kubevirt/kubevirt/pull/7768

#### `004-network-aware-livemigration-for-macvtap.patch`

Same as above but also enables live-migration for macvtap interfaces

#### `005-macvtap-binding.patch`

This PR adds macvtap networking mode for binding podNetwork.

- https://github.com/kubevirt/community/pull/186
- https://github.com/kubevirt/kubevirt/pull/7648
