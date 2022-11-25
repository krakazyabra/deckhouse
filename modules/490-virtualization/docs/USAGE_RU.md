---
title: "Модуль virtualization: примеры конфигурации"
---

## Настройка хранения образов

Для наиболее оптимального хранения образов и дисков виртуальных машин, рекуомендуется использовать блочные тома с возможностью конкуретного доступа (режим `ReadWriteMany`).
Следующие опции можно задать в StorageClass, чтобы модуль виртуализации создавал тома с оптимальными параметрами:

```yaml
virtualization.deckhouse.io/volumeMode: Block
virtualization.deckhouse.io/accessModes: ReadWriteMany
```

При использовании linstor в качестве хранилища, автоматически созданные StorageClasses уже содержат данные опции.

> **Внимание:** не все хранилища поддерживают данные режимы.

## Получение списка доступных имаджей

Deckhouse поставляется уже с несколькими базовыми образами, которые вы можете использовать для создания виртуальных машин. Для того чтобы получить их список, выполните:

```bash
kubectl get publicimagesources.deckhouse.io
```

пример вывода:
```bash
# kubectl get publicimagesources.deckhouse.io
NAME           DISTRO         VERSION    AGE
alpine-3.16    Alpine Linux   3.16       29m
centos-7       CentOS         7          29m
centos-8       CentOS         8          29m
debian-9       Debian         9          29m
debian-10      Debian         10         29m
fedora-36      Fedora         36         29m
rocky-9        Rocky Linux    9          29m
ubuntu-20.04   Ubuntu         20.04      29m
ubuntu-22.04   Ubuntu         22.04      29m
```


## Создание VirtualMachine

Минимальный ресурс для создания виртуальной машины выглядит так:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm1
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
    ephemeral: true
```

В качестве источника для bootDisk, можно указать и существующий диск виртуальной машины. В этом случае он будет подключен к ней напрямую без выполнения операции клонирования.

Параметр `ephemeral` позволяет определить, должен ли диск быть удалён после удаления виртуальной машины.

## Назначение статического IP-адреса

Для того чтобы назначить статический IP-адрес, достаточно добавить поле `staticIPAddress` с желаемым IP-адресом:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm1
  namespace: default
spec:
  running: true
  staticIPAddress: 10.10.10.8
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
    ephemeral: true
```

Желаемый IP-адрес должен находиться в пределах одного из `vmCIDR` определённого в конфигурации модуля и не быть в использовании какой-либо другой виртуальной машины.

После удаления VM, статический IP-адрес остаётся зарезервированным в неймспейсе, посмотреть список всех выданных IP-адресов, можно следующим образом:

```bash
kubectl get vmip
```

пример вывода команды:
```bash
# kubectl get vmip
NAME             STATIC   VM
ip-10-10-10-0    true
ip-10-10-10-1             vm1
ip-10-10-10-2    true
ip-10-10-10-88   true
ip-10-10-10-99   true
```

Для того чтобы освободить адрес, удалите ресурс `IPAddressLease`:

```bash
kubectl delete vmip ip-10-10-10-88
```

## Создание дисков для хранения персистентных данных

Дополнительные диски необходимо создавать вручную

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineDisk
metadata:
  name: mydata
spec:
  storageClassName: linstor-slow
  size: 10Gi
```

Имеется возможность создать диск из существующего образа, для этого достаточно указать source:

```yaml
source:
  kind: ClusterVirtualMachineImage
  name: centos-7
```

Подключение дополнительных дисков выполняется с помощью параметра `diskAttachments`:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm1
  namespace: default
spec:
  running: true
  staticIPAddress: 10.10.10.8
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
    ephemeral: true
  diskAttachments:
  - name: mydata
    bus: virtio
```

## Использование cloud-init

При необходимости вы можете передать конфигурацию cloud-init:

```yaml
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
  bootDisk:
    image:
      name: ubuntu-20.04
      size: 10Gi
      type: linstor-slow
  cloudInit:
    userData: |-
      chpasswd: { expire: False }
```

Конфигцрацию cloud-init, можно также положить в секрет и передать виртуальной машине следулющим образом:

```yaml
  cloudInit:
    secretRef:
      name: my-vmi-secret
```
