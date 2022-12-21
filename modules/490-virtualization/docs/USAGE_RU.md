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
  name: vm100
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
      name: ubuntu-22.04
    size: 10Gi
    storageClassName: linstor-thindata-r2
    autoDelete: true
```

В качестве источника для bootDisk, можно указать и существующий диск виртуальной машины. В этом случае он будет подключен к ней напрямую без выполнения операции клонирования.

Параметр `autoDelete` позволяет определить, должен ли диск быть удалён после удаления виртуальной машины.

## Работа с IP-адресами

Для каждой виртуальной машины назначается отдельный IP-адрес, который она использует на протяжении всей своей жизни.  
Для этого используется механизм IPAM (IP Address Management), который представляет ссобой два ресурса: `VirtualMachineIPAddressClaim` и `VirtualMachineIPAddressLease`

Когда `VirtualMachineIPAddressLease` является кластерным ресурсом и отражает сам факт выданного адреса для виртуальной машины. То `VirtualMachineIPAddressClaim` является пользовательским ресурсом и используется для запроса такого адреса. Создав `VirtualMachineIPAddressClaim` вы можете запросить желаемый IP-адрес для виртуальной машины, пример:

Желаемый IP-адрес должен находиться в пределах одного из `vmCIDR` определённого в конфигурации модуля и не быть в использовании какой-либо другой виртуальной машины.

```
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineIPAddressClaim
metadata:
  name: vm100
  namespace: default
spec:
  address: 10.10.10.10
  leaseName: ip-10-10-10-10
  static: true
```

В случае если `VirtualMachineIPAddressClaim` не был создан пользователем заранее, то он создастся автоматически при создании виртуальной машины.  
В этом случае будет назначен следующий свободный IP-адрес из диапазона vmCIDR.  
При удалении виртуальной машины удалится так же и связанный с ней `VirtualMachineIPAddressClaim`

Для того чтобы этого не происходило нужно пометить такой IP-аддресс как статический.  
Для этого нужно отредактировать созданный `VirtualMachineIPAddressClaim` и установить в нём поле `static: true`.

После удаления VM, статический IP-адрес остаётся зарезервированным в неймспейсе, посмотреть список всех выданных IP-адресов, можно следующим образом:

```bash
kubectl get vmip
```

пример вывода команды:
```bash
# kubectl get vmip
NAME    ADDRESS       STATIC   STATUS   VM      AGE
vm1     10.10.10.0    false    Bound    vm1     9d
vm100   10.10.10.10   true     Bound    vm100   172m
```

Для того чтобы освободить адрес, просто удалите ресурс `VirtualMachineIPAddressClaim`:

```bash
kubectl delete vmip vm100
```

`VirtualMachineIPAddressClaim` по умолчанию называются также как и виртуальная машина, но есть возможность передать и любое другое произвольное имя, для этого в спеке виртуальной машины необходимо указать:

```
ipAddressClaimName: <имя>
```

## Создание дисков для хранения персистентных данных

Дополнительные диски необходимо создавать вручную

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineDisk
metadata:
  name: mydata
spec:
  storageClassName: linstor-data
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
  name: vm100
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
      name: ubuntu-22.04
    size: 10Gi
    storageClassName: linstor-fast
    autoDelete: true
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
      type: linstor-data
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
