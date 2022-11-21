---
title: "Модуль virtualization: примеры конфигурации"
---

## Создание DiskType

DiskType - необходимая сущность для описания типа создаваемых дисков.  
Разные DiskTypes могут использоваться для создания дисков с различными характеристиками, к примеру: `slow`, `fast` и т.д.

При использовании linstor, укажите следующие параметры:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: DiskType
metadata:
  name: linstor-slow
spec:
  accessModes: [ "ReadWriteMany" ]
  volumeMode: Block
  storageClassName: linstor-data-r2
```

Где `storageClassName` - это желаемый StorageClass.

Необходимо создать хотя бы один DiskType на весь кластер. При желании его можно назначить DiskType по умолчанию:

```bash
kubectl annotate disktype.deckhouse.io linstor-slow deckhouse.io/is-default-type=true
```

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
    image:
      name: ubuntu-20.04
      size: 10Gi
      type: linstor-slow
```

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
    image:
      name: ubuntu-20.04
      size: 10Gi
      type: linstor-slow
```

Желаемый IP-адрес должен находиться в пределах одного из `vmCIDR` определённого в конфигурации модуля и не быть в использовании какой-либо другой виртуальной машины.

После удаления VM, статический IP-адрес остаётся зарезервированным в неймспейсе, посмотреть список всех выданных IP-адресов, можно следующим образом:

```bash
kubectl get ipaddressleases.deckhouse.io
```

пример вывода команды:
```bash
# kubectl get ipaddressleases.deckhouse.io
NAME            STATIC   VM     AGE
ip-10-10-10-1   false    vm5    29m
ip-10-10-10-2   false           34m
ip-10-10-10-4   true     vm4    21m
ip-10-10-10-8   true            10m
```

Для того чтобы освободить адрес, удалите ресурс `IPAddressLease`:

```bash
kubectl delete ipaddressleases.deckhouse.io ip-10-10-10-8
```

## Создание дисков для хранения персистентных данных

Дополнительные диски необходимо создавать вручную

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: Disk
metadata:
  name: mydata
spec:
  type: linstor-slow
  size: 10Gi
```

Подключение дополнительных дисков выполняется с помощью параметра `disks`:

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
  disks:
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

При желании конфигцрацию cloud-init, можно положить в секрет и передать виртуальной машине следулющим образом:

```yaml
  cloudInit:
    secretRef:
      name: my-vmi-secret
```
