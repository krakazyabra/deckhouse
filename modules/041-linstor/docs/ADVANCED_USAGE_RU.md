---
title: "Модуль linstor: расширенная конфигурация"
---

[Упрощенное руководство](configuration.html#конфигурация-хранилища-linstor) содержит шаги, в результате выполнения которых автоматически создаются пулы хранения (storage-пулы) и StorageClass'ы, при появлении на узле LVM-группы томов или LVMThin-пула с тегом `linstor-<имя_пула>`. Далее рассматривается шаги по ручному созданию пулов хранения и StorageClass'ов.

Для выполнения дальнейших действий потребуется CLI-утилита `linstor`. Используйте один из следующих вариантов запуска утилиты `linstor`:
- Установите плагин [kubectl-linstor](https://github.com/piraeusdatastore/kubectl-linstor).
- Добавьте alias в BASH для запуска утилиты `linstor` из Pod'а контроллера linstor:

  ```shell
  alias linstor='kubectl exec -n d8-linstor deploy/linstor-controller -- linstor'
  ```

## Начальная конфигурация

После включения модуля `linstor` кластер и его узлы настраиваются на использование LINSTOR автоматически. Для того чтобы начать использовать хранилище, необходимо:

- [Создать пулы хранения](#создание-пулов-хранения)
- [Создать StorageClass](#создание-storageclass)

### Создание пулов хранения

1. Отобразите список всех узлов и блочных устройств для хранения.
   - Отобразите список всех узлов:

     ```shell
     linstor node list
     ```

     Пример вывода:
  
     ```text
     +----------------------------------------------------------------------------------------+
     | Node                                | NodeType   | Addresses                  | State  |
     |========================================================================================|
     | node01                              | SATELLITE  | 192.168.199.114:3367 (SSL) | Online |
     | node02                              | SATELLITE  | 192.168.199.60:3367 (SSL)  | Online |
     | node03                              | SATELLITE  | 192.168.199.74:3367 (SSL)  | Online |
     | linstor-controller-85455fcd76-2qhmq | CONTROLLER | 10.111.0.78:3367 (SSL)     | Online |
     +----------------------------------------------------------------------------------------+
     ```

   - Отобразите список всех доступных блочных устройств для хранения:

     ```shell
     linstor physical-storage list
     ```
  
     Пример вывода:
  
     ```text
     +----------------------------------------------------------------+
     | Size          | Rotational | Nodes                             |
     |================================================================|
     | 1920383410176 | False      | node01[/dev/nvme1n1,/dev/nvme0n1] |
     | 1920383410176 | False      | node02[/dev/nvme1n1,/dev/nvme0n1] |
     | 1920383410176 | False      | node03[/dev/nvme1n1,/dev/nvme0n1] |
     +----------------------------------------------------------------+
     ```

     > **Обратите внимание:** отображаются только пустые устройства, без какой-либо разметки.
     > Тем не менее, создание пулов хранения из разделов и других блочных устройств также поддерживается.
     >
     > Вы также можете [добавить](faq.html#как-добавить-существующий-lvm-или-lvmthin-пул) уже существующий пул LVM или LVMthin в кластер.

1. Создайте пулы LVM или LVMThin.

   На необходимых узлах хранилища создайте несколько пулов из устройств, полученных на предыдущем шаге. Их названия должны быть одинаковыми, в случае если вы хотите иметь один storageClass.

   - Пример команды создания **LVM-пула** хранения из двух устройств на одном из узлов:

     ```shell
     linstor physical-storage create-device-pool lvm node01 /dev/nvme0n1 /dev/nvme1n1 --pool-name linstor_data --storage-pool lvm
     ```

     , где:
     - `--pool-name` — имя VG/LV создаваемом на узле.
     - `--storage-pool` — то, как будет называться пул хранения в LINSTOR.

   - Пример команды создания **ThinLVM-пула** хранения из двух устройств на одном из узлов:

     ```shell
     linstor physical-storage create-device-pool lvmthin node01 /dev/nvme0n1 /dev/nvme1n1 --pool-name data --storage-pool lvmthin
     ```

     , где:
     - `--pool-name` — имя VG/LV создаваемом на узле.
     - `--storage-pool` — то, как будет называться пул хранения в LINSTOR.

1. Проверьте создание пулов хранения.

   Как только пулы хранения созданы, можете увидеть их выполнив следующую команду:

   ```shell
   linstor storage-pool list
   ```

   Пример вывода:

   ```text
   +---------------------------------------------------------------------------------------------------------------------------------+
   | StoragePool          | Node   | Driver   | PoolName          | FreeCapacity | TotalCapacity | CanSnapshots | State | SharedName |
   |=================================================================================================================================|
   | DfltDisklessStorPool | node01 | DISKLESS |                   |              |               | False        | Ok    |            |
   | DfltDisklessStorPool | node02 | DISKLESS |                   |              |               | False        | Ok    |            |
   | DfltDisklessStorPool | node03 | DISKLESS |                   |              |               | False        | Ok    |            |
   | lvmthin              | node01 | LVM_THIN | linstor_data/data |     3.49 TiB |      3.49 TiB | True         | Ok    |            |
   | lvmthin              | node02 | LVM_THIN | linstor_data/data |     3.49 TiB |      3.49 TiB | True         | Ok    |            |
   | lvmthin              | node03 | LVM_THIN | linstor_data/data |     3.49 TiB |      3.49 TiB | True         | Ok    |            |
   +---------------------------------------------------------------------------------------------------------------------------------+
   ```

### Создание StorageClass

Создайте StorageClass, где:
- в `parameters."linstor.csi.linbit.com/placementCount"` укажите необходимое количество реплик;
- в `parameters."linstor.csi.linbit.com/storagePool"` укажите имя пула хранения, в котором будут создаваться реплики.

Пример StorageClass:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: linstor-r2
parameters:
  linstor.csi.linbit.com/storagePool: lvmthin
  linstor.csi.linbit.com/placementCount: "2"
  property.linstor.csi.linbit.com/DrbdOptions/Net/rr-conflict: retry-connect
  property.linstor.csi.linbit.com/DrbdOptions/Resource/on-no-data-accessible: suspend-io
  property.linstor.csi.linbit.com/DrbdOptions/Resource/on-suspended-primary-outdated: force-secondary
  property.linstor.csi.linbit.com/DrbdOptions/auto-quorum: suspend-io
allowVolumeExpansion: true
provisioner: linstor.csi.linbit.com
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
```

## Резервное копирование в S3

> Использование данной возможности требует настроенного мастер-пароля (см. инструкции вначале страницы [конфигурации модуля](configuration.html)).
>
> Резервное копирование с помощью снапшотов поддерживается только для LVMThin-пулов.

### Используя механизм снапшотов в Kubernetes

Резервное копирование данных реализовано с помощью [снапшотов томов](https://kubernetes.io/docs/concepts/storage/volume-snapshots/). Поддержка работы снапшотов обеспечивается модулем [snapshot-controller](../045-snapshot-controller/), который включается автоматически для поддерживаемых CSI-драйверов в кластерах Kubernetes версий 1.20 и выше.

#### Создание резервной копии

Для создания снапшота тома и загрузки его в S3 выполните следующие шаги:

1. Создайте `VolumeSnapshotClass` и `Secret`, содержащий access key и secret key доступа к хранилищу S3.

   > VolumeSnapshotClass — ресурс на уровне кластера. Один и тот же VolumeSnapshotClass можно использовать для создания резервных копий разных PVC из разных пространств имен.

   Пример `VolumeSnapshotClass` и `Secret`:

   ```yaml
   kind: VolumeSnapshotClass
   apiVersion: snapshot.storage.k8s.io/v1
   metadata:
     name: linstor-csi-snapshot-class-s3
   driver: linstor.csi.linbit.com
   deletionPolicy: Retain
   parameters:
     snap.linstor.csi.linbit.com/type: S3
     snap.linstor.csi.linbit.com/remote-name: backup-remote               # Уникальное название backup-подключения в linstor.   
     snap.linstor.csi.linbit.com/allow-incremental: "false"               # Использовать ли инкрементальные копии. 
     snap.linstor.csi.linbit.com/s3-bucket: snapshot-bucket               # Название S3 bucket, для хранения данных.
     snap.linstor.csi.linbit.com/s3-endpoint: s3.us-west-1.amazonaws.com  # S3 endpoint URL.
     snap.linstor.csi.linbit.com/s3-signing-region: us-west-1             # Регион S3. 
     # Использовать virtual hosted–style или path-style S3 URL 
     # https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html
     snap.linstor.csi.linbit.com/s3-use-path-style: "false"    
     # Ссылка на Secret, содержащий access key и secret key доступа к S3 bucket.
     csi.storage.k8s.io/snapshotter-secret-name: linstor-csi-s3-access
     csi.storage.k8s.io/snapshotter-secret-namespace: storage
   ---
   kind: Secret
   apiVersion: v1
   metadata:
     name: linstor-csi-s3-access
     namespace: storage
   immutable: true
   type: linstor.csi.linbit.com/s3-credentials.v1
   stringData:
     access-key: *!ACCESS_KEY*  # Access key доступа к хранилищу S3.
     secret-key: *!SECRET_KEY*  # Secret key доступа к хранилищу S3.
   ```

1. Выберите (или создайте) `PersistentVolumeClaim`, данные которого нужно копировать.

   Пример `PersistentVolumeClaim`, который будет использоваться в примерах далее:

   ```yaml
   apiVersion: v1
   kind: PersistentVolumeClaim
   metadata:
     name: my-linstor-volume
     namespace: storage
   spec:
     accessModes:
     - ReadWriteOnce
     storageClassName: linstor-thindata-r2   # StorageClass хранилища linstor.
     resources:
       requests:
         storage: 2Gi
   ```

1. Создайте `VolumeSnapshot`.

   Пример `VolumeSnapshot`, использующего `VolumeSnapshotClass` созданный ранее:

   ```yaml
   apiVersion: snapshot.storage.k8s.io/v1
   kind: VolumeSnapshot
   metadata:
     name: my-linstor-snapshot
     namespace: storage
   spec:
     volumeSnapshotClassName: linstor-csi-snapshot-class-s3  # Имя VolumeSnapshotClass, с доступом к хранилищу S3.
     source:
       persistentVolumeClaimName: my-linstor-volume          # Имя PVC, данные с тома которого необходимо копировать. 
   ```

   После создания `VolumeSnapshot` связанного с `PersistentVolumeClaim` относящимся к существующему тому с данными, произойдет создание снапшота в linstor и загрузка его в хранилище S3.

1. Проверьте, статус выполнения резервного копирования.

   Пример:

   ```shell
   kubectl get volumesnapshot my-linstor-snapshot -n storage
   ```

   Если значение READYTOUSE `VolumeSnapshot` не `true`, то посмотрите причину, выполнив следующую команду:  

   ```shell
   kubectl describe volumesnapshot my-linstor-snapshot -n storage
   ```

Посмотреть список и состояние созданных снапшотов в linstor, можно выполнив следующую команду:

```shell
linstor snapshot list
```

#### Восстановление из резервной копии

Для восстановления данных в том же пространстве имен, в котором был создан VolumeSnapshot, достаточно создать PVC со ссылкой на необходимый VolumeSnapshot.

Пример PVC для восстановления из VolumeSnapshot `example-backup-from-s3` в том же пространстве имен:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: restored-data
  namespace: storage
spec:
  storageClassName: "linstor-thindata-r1" # Имя StorageClass тома для восстановления данных.  
  dataSource:
    name: example-backup-from-s3          # Имя созданного ранее VolumeSnapshot.
    kind: VolumeSnapshot
    apiGroup: snapshot.storage.k8s.io
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
```

Для восстановления данных из хранилища S3 в другом пространстве имен или кластере Kubernetes, выполните следующие шаги:

1. Создайте `VolumeSnapshotClass` и `Secret`, содержащий access key и secret key доступа к хранилищу S3, если они не были созданы ранее (например, если вы восстанавливаете данные в новом кластере).

   Пример `VolumeSnapshotClass` и `Secret`:

   ```yaml
   kind: VolumeSnapshotClass
   apiVersion: snapshot.storage.k8s.io/v1
   metadata:
     name: linstor-csi-snapshot-class-s3
   driver: linstor.csi.linbit.com
   deletionPolicy: Retain
   parameters:
     snap.linstor.csi.linbit.com/type: S3
     snap.linstor.csi.linbit.com/remote-name: backup-remote               # Уникальное название backup-подключения в linstor.   
     snap.linstor.csi.linbit.com/allow-incremental: "false"               # Использовать ли инкрементальные копии. 
     snap.linstor.csi.linbit.com/s3-bucket: snapshot-bucket               # Название S3 bucket, для хранения данных.
     snap.linstor.csi.linbit.com/s3-endpoint: s3.us-west-1.amazonaws.com  # S3 endpoint URL.
     snap.linstor.csi.linbit.com/s3-signing-region: us-west-1             # Регион S3. 
     # Использовать virtual hosted–style или path-style S3 URL 
     # https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html
     snap.linstor.csi.linbit.com/s3-use-path-style: "false"    
     # Ссылка на Secret, содержащий access key и secret key доступа к S3 bucket.
     csi.storage.k8s.io/snapshotter-secret-name: linstor-csi-s3-access
     csi.storage.k8s.io/snapshotter-secret-namespace: storage
   ---
   kind: Secret
   apiVersion: v1
   metadata:
     name: linstor-csi-s3-access
     namespace: storage
   immutable: true
   type: linstor.csi.linbit.com/s3-credentials.v1
   stringData:
     access-key: *!ACCESS_KEY*  # Access key доступа к хранилищу S3.
     secret-key: *!SECRET_KEY*  # Secret key доступа к хранилищу S3.
   ```

1. Получите id снапшота для восстановления одним из следующих способов:

   1. Получите список снапшотов в кластере linstor, и выберите нужный (колонка `SnapshotName`):

      ```shell
      linstor backup list <backup-remote-name>
      ```

      , где `<backup-remote-name>` — название backup-подключения, использованное в `VolumeSnapshotClass`.

   1. Получите id-снапшота из имени объекта в S3-бакете через UI-интерфейс или CLI-утилиты S3-сервиса.

1. Создайте `VolumeSnapshotContent`, указывающий на конкретный id снапшота.

   > VolumeSnapshotContent — ресурс на уровне кластера. Каждый VolumeSnapshotClass может быть связан только с одним VolumeSnapshot. Поэтому удостоверьтесь в уникальности его имени.

   Пример:

   ```yaml
   apiVersion: snapshot.storage.k8s.io/v1
   kind: VolumeSnapshotContent
   metadata:
     name: restored-snap-content-from-s3
   spec:
     deletionPolicy: Delete
     driver: linstor.csi.linbit.com
     source:
       snapshotHandle: *!snapshot_id*                        # ID снапшота.  
     volumeSnapshotClassName: linstor-csi-snapshot-class-s3  # Имя VolumeSnapshotClass, с доступом к хранилищу S3.
     volumeSnapshotRef:
       apiVersion: snapshot.storage.k8s.io/v1
       kind: VolumeSnapshot
       name: example-backup-from-s3                          # Имя VolumeSnapshot, который будет создан далее.
       namespace: storage
   ```

1. Создайте `VolumeSnapshot`, указывающий на созданный `VolumeSnapshotContent`.

   Пример:

   ```yaml
   apiVersion: snapshot.storage.k8s.io/v1
   kind: VolumeSnapshot
   metadata:
     name: example-backup-from-s3
     namespace: storage
   spec:
     source:
       volumeSnapshotContentName: restored-snap-content-from-s3 # Имя VolumeSnapshotContent, созданного ранее.
     volumeSnapshotClassName: linstor-csi-snapshot-class-s3     # Имя VolumeSnapshotClass, с доступом к хранилищу S3.
   ```

1. Создайте `PersistentVolumeClaim`.

   Пример:

   ```yaml
   apiVersion: v1
   kind: PersistentVolumeClaim
   metadata:
     name: restored-data
     namespace: storage
   spec:
     storageClassName: "linstor-thindata-r1" # Имя StorageClass тома для восстановления данных.  
     dataSource:
       name: example-backup-from-s3          # Имя созданного ранее VolumeSnapshot.
       kind: VolumeSnapshot
       apiGroup: snapshot.storage.k8s.io
     accessModes:
       - ReadWriteOnce
     resources:
       requests:
         storage: 2Gi
   ```

Используйте созданный `PersistentVolumeClaim` для доступа к копии восстановленных данных.

### Используя LINSTOR CLI

#### Remotes

In a LINSTOR cluster, the definition of a shipping target is called a remote.
S3 remotes are needed to ship snapshots to AWS S3, min.io or any other service using S3
compatible object storage.

To create an S3 remote, LINSTOR will need to know the endpoint
(that is, the URL of the target S3 server), the name of the target bucket, the region the S3
server is in, as well as the access-key and secret-key used to access the bucket. If the command
is sent without adding the secret-key, a prompt will pop up to enter it in. The command
should look like this:

```bash
# linstor remote create s3 myRemote s3.us-west-2.amazonaws.com \
  my-bucket us-west-2 admin password
```

TIP: Usually, LINSTOR uses the endpoint and bucket to create an URL using the virtual-hosted-style
for its access to the given bucket (for example my-bucket.s3.us-west-2.amazonaws.com). Should your setup not
allow access this way, change the remote to path-style access (for example s3.us-west-2.amazonaws.com/my-bucket)
by adding the `--use-path-style` argument to make LINSTOR combine the parameters accordingly.

To see all the remotes known to the local cluster, use `linstor remote list`. To delete a remote, use
`linstor remote delete myRemoteName`. Should an existing remote need altering, use `linstor remote
modify` to change it.

#### Shipping Backups to S3

All that is needed to ship a snapshot to S3 is to create an S3-remote that the current cluster can reach
as well as the resource that should be shipped. Then, simply use the following command to ship it there:

```bash
# linstor backup create myRemote myRsc
```

This command will create a snapshot of your resource and ship it to the given remote. If this
isn't the first time you shipped a backup of this resource (to that remote) and the snapshot
of the previous backup hasn't been deleted yet, an incremental backup will be shipped.
To force the creation of a full backup, add the `--full` argument to the command. Getting a
specific node to ship the backup is also possible by using `--node myNode`, but if the specified
node is not available or only has the resource diskless, a different node will be chosen.

To see which backups exist in a specific remote, use `linstor backup list myRemote`. A resource-name
can be added to the command as a filter to only show backups of that specific resource by using the
argument `--resource myRsc`. If you use the `--other` argument, only entries in the bucket that LINSTOR
does not recognize as a backup will be shown. LINSTOR always names backups in a certain way, and
as long as an item in the remote is named according to this schema, it is assumed that it is a backup
created by LINSTOR - so this list will show everything else.

There are several options when it comes to deleting backups:

* `linstor backup delete all myRemote`: This command deletes ALL S3-objects on the given remote,
provided that they are recognized to be backups, that is, fit the expected naming schema. There
is the option `--cluster` to only delete backups that were created by the current cluster.

* `linstor backup delete id myRemote my-rsc_back_20210824_072543`: This command deletes
a single backup from the given remote - namely the one with the given id, which consists
of the resource-name, the automatically generated snapshot-name (back_timestamp) and, if
set, the backup-suffix. The option `--prefix` lets you delete all backups starting with
the given id. The option `--cascade` deletes not only the specified backup, but all other
incremental backups depending on it.

* `linstor backup delete filter myRemote ...`: This command has a few different arguments
to specify a selection of backups to delete. `-t 20210914_120000` will delete all backups
made before 12 o'clock on the 14th of September, 2021. `-n myNode` will delete all backups
uploaded by the given node. `-r myRsc` will delete all backups with the given resource name.
These filters can be combined as needed. Finally, `--cascade` deletes not only the selected
backup(s), but all other incremental backups depending on any of the selected backups.

* `linstor backup delete s3key myRemote randomPictureInWrongBucket`: This command will find the
object with the given S3-key and delete it - without considering anything else. This should
only be used to either delete non-backup items from the remote, or to clean up a broken backup
that is no longer deleteable by other means. Using this command to delete a regular, working
backup will break that backup, so beware!

WARNING: All commands that have the `--cascade` option will NOT delete a backup that has
incremental backups depending on it unless you explicitly add that option.

TIP: All `linstor backup delete ...` commands have the `--dry-run` option, which will
give you a list of all the S3-objects that will be deleted. This can be used to ensure
nothing that should not be deleted is accidentally deleted.

Maybe the most important task after creating a backup is restoring it. To do so, only the remote
is needed - but it is also possible to restore into an existing resource definition with no existing
snapshots nor resources. There are two options for the command:

```bash
# linstor backup restore myRemote myNode targetRsc --resource sourceRsc
# linstor backup restore myRemote myNode targetRsc --id sourceRsc_back_20210824_072543
```

Either `--resource (-r)` or `--id` must be used, but you cannot use both of them together. `-r` is used to
restore the latest backup of the resource specified with this option, while `--id` restores the
exact backup specified by the given id, and can therefore be used to restore backups other than
the most recent.

If the backup to be restored includes a LUKS layer, the `--passphrase` argument is required. With
it, the passphrase of the original cluster of the backup needs to be set so that LINSTOR can decrypt
the volumes after download and re-encrypt them with the local passphrase.

The backup restore will download all the snapshots from the last full backup up to the specified
backup. Afterwards, it restores the snapshots into a new resource. If that last step should be skipped,
the `--download-only` option needs to be added to the command.

Backups can be downloaded from any cluster, not just the one that uploaded them, provided that the setup
is correct. Specifically, the target resource cannot have any existing resources or snapshots, and the
storage pool(s) used need to have the same storage providers. If the storage pool(s) on the target
node have the exact same names as on the cluster the backup was created on, no extra action is
necessary. Should they have different names, the option `--storpool-rename` needs to be used. It
expects at least one `oldname=newname` pair. For every storage pool of the original backup that
is not named in that list, it will be assumed that its name is exactly the same on the target node.

To find out exactly which storage pools need to be renamed, as well as how big the download and the
restored resource will be, the command `linstor backup info myRemote ...` can be used. Similar to the restore
command, either `-r` or `--id` need to be given, which add the same restrictions as with that command.
To see how much space will be left over in the local storage pools after a restore, the argument `-n myNode`
needs to be added. Just like with a restore, it assumes the storage pool names are exactly the same
on the given node as with the backup. Should that not be the case, again, just like with the restore
command, `--storpool-rename` should be used.


#### Scheduled Backup Shipping

Starting with LINSTOR Controller version 1.19.0 and working with LINSTOR client version 1.14.0
or above, you can configure scheduled backup shipping for deployed LINSTOR resources.

Scheduled backup shipping consists of three parts:

- A data set that consists of one or more deployed LINSTOR resources that you want to backup and
  ship

- A remote destination to ship backups to (another LINSTOR cluster or an S3 instance)

- A schedule that defines when the backups should ship

IMPORTANT: LINSTOR backup shipping only works for deployed LINSTOR resources that are backed by
LVM and ZFS storage pools, because these are the storage pool types with snapshot support in
LINSTOR.

##### Creating a Backup Shipping Schedule

You create a backup shipping schedule by using the LINSTOR client `schedule create` command and
defining the frequency of backup shipping using `cron` syntax. You also need to set options
that name the schedule and define various aspects of the backup shipping, such as on-failure
actions, the number of local and remote backup copies to keep, and whether to also schedule
incremental backup shipping.

At a minimum, the command needs a schedule name and a full backup cron schema to create a backup
shipping schedule. An example command would look like this:

```bash
# linstor schedule create \
  --incremental-cron '* * * * *' \ <1>
  --keep-local 5 \ <2>
  --keep-remote 4 \ <3>
  --on-failure RETRY \ <4>
  --max-retries 10 \ <5>
  <schedule_name> \ <6>
  '* * * * *' # full backup cron schema <7>
```

IMPORTANT: Enclose cron schemas within single or double quotation marks.

<1> If specified, the incremental cron schema describes how frequently to create and ship
incremental backups. New incremental backups are based on the most recent full backup.
[OPTIONAL]

<2> The `--keep-local` option allows you to specify how many snapshots that a full backup is
based upon should be kept at the local backup source. If unspecified, all snapshots will be
kept. [OPTIONAL]

<3> The `--keep-remote` option allows you to specify how many full backups should be kept at the
remote destination. This option only works with S3 remote backup destinations, because you would
not want to allow a cluster node to delete backups from a node in another cluster. All
incremental backups based on a deleted full backup will also be deleted at the remote
destination. If unspecified, the `--keep-remote` option defaults to "all". [OPTIONAL]

<4> Specifies whether to "RETRY" or "SKIP" the scheduled backup shipping if it fails. If "SKIP"
is specified, LINSTOR will ignore the failure and continue with the next scheduled backup
shipping. If "RETRY" is specified, LINSTOR will wait 60 seconds and then try the backup shipping
again. The LINSTOR `schedule create` command defaults to "SKIP" if no `--on-failure` option is
given. [OPTIONAL]

<5> The number of times to retry the backup shipping if a scheduled backup shipping fails and
the `--on-failure RETRY` option has been given. Without this option, the LINSTOR controller will
retry the scheduled backup shipping indefinitely, until it is successful. [OPTIONAL]

<6> The name that you give the backup schedule so that you can reference it later with the
schedule list, modify, delete, enable, or disable commands. [REQUIRED]

<7> This cron schema describes how frequently LINSTOR creates snapshots and ships full backups.
[REQUIRED]

IMPORTANT: If you specify an incremental cron schema that has overlap with the full cron schema
that you specify, at the times when both types of backup shipping would occur simultaneously,
LINSTOR will only make and ship a full backup. For example, if you specify that a full backup be
made every three hours, and an incremental backup be made every hour, then every third hour,
LINSTOR will only make and ship a full backup. For this reason, specifying the same cron schema
for both your incremental and full backup shipping schedules would be useless, because
incremental backups will never be made.

##### Modifying a Backup Shipping Schedule

You can modify a backup shipping schedule by using the LINSTOR client `schedule modify` command.
The syntax for the command is the same as that for the `schedule create` command. The name that
you specify with the `schedule modify` command must be an already existing backup schedule. Any
options to the command that you do not specify will retain their existing values. If you want to
set the `keep-local` or `keep-remote` options back to their default values, you can set them to
"all". If you want to set the `max-retries` option to its default value, you can set it to
"forever".

##### Configuring the Number of Local Snapshots and Remote Backups to Keep

Your physical storage is not infinite and your remote storage has a cost, so you will likely
want to set limits on the number of snapshots and backups you keep.

Both the `--keep-remote` and `--keep-local` options deserve special mention as they have
implications beyond what may be obvious. Using these options, you specify how many snapshots or
full backups should be kept, either on the local source or the remote destination.

###### Configuring the Keep-local Option

For example, if a `--keep-local=2` option is set, then the backup shipping schedule, on first
run, will make a snapshot for a full backup. On the next scheduled full backup shipping, it will
make a second snapshot for a full backup. On the next scheduled full backup shipping, it makes a
third snapshot for a full backup. This time, however, after successful completion, LINSTOR
deletes the first (oldest) full backup shipping snapshot. If snapshots were made for any
incremental backups based on this full snapshot, they will also be deleted from the local source
node. On the next successful full backup shipping, LINSTOR will delete the second full backup
snapshot and any incremental snapshots based upon it, and so on, with each successive backup
shipping.

NOTE: If there are local snapshots remaining from failed shipments, these will be deleted first,
even if they were created later.

If you have enabled a backup shipping schedule and then later manually delete a LINSTOR
snapshot, LINSTOR may not be able to delete everything it was supposed to. For example, if you
delete a full backup snapshot definition, on a later full backup scheduled shipping, there may
be incremental snapshots based on the manually deleted full backup snapshot that will not be
deleted.

###### Configuring the Keep-remote Option

As mentioned in the callouts for the example `linstor schedule create` command above, the
`keep-remote` option only works for S3 remote destinations. Here is an example of how the
option works. If a `--keep-remote=2` option is set, then the backup shipping schedule, on first
run, will make a snapshot for a full backup and ship it to the remote destination. On the next
scheduled full backup shipping, a second snapshot is made and a full backup shipped to the
remote destination. On the next scheduled full backup shipping, a third snapshot is made and a
full backup shipped to the remote destination. This time, additionally, after the third snapshot
successfully ships, the first full backup is deleted from the remote destination. If any
incremental backups were scheduled and made between the full backups, any that were made from
the first full backup would be deleted along with the full backup.

NOTE: This option only deletes backups at the remote destination. It does not delete snapshots
that the full backups were based upon at the local source node.

##### Listing a Backup Shipping Schedule

You can list your backup shipping schedules by using the `linstor schedule list` command.

For example:

```bash
# linstor schedule list
╭──────────────────────────────────────────────────────────────────────────────────────╮
┊ Name                ┊ Full        ┊ Incremental ┊ KeepLocal ┊ KeepRemote ┊ OnFailure ┊
╞══════════════════════════════════════════════════════════════════════════════════════╡
┊ my-bu-schedule      ┊ 2 * * * *   ┊             ┊ 3         ┊ 2          ┊ SKIP      ┊
╰──────────────────────────────────────────────────────────────────────────────────────╯
```

##### Deleting a Backup Shipping Schedule

The LINSTOR client `schedule delete` command completely deletes a backup shipping schedule
LINSTOR object. The command's only argument is the schedule name that you want to delete. If the
deleted schedule is currently creating or shipping a backup, the scheduled shipping process is
stopped. Depending on at which point the process stops, a snapshot, or a backup, or both, might
not be created and shipped.

This command does not affect previously created snapshots or successfully shipped backups. These
will be retained until they are manually deleted.

##### Enabling Scheduled Backup Shipping

You can use the LINSTOR client `backup schedule enable` command to enable a previously created
backup shipping schedule. The command has the following syntax:

```bash
# linstor backup schedule enable \
  [--node __source_node__] \ <1>
  [--rg __resource_group_name__ | --rd __resource_definition_name__] \ <2>
  __remote_name__ \ <3>
  __schedule_name__ <4>
```

<1> This is a special option that allows you to specify the controller node that will be used as
a source for scheduled backup shipments, if possible. If you omit this option from the command,
then LINSTOR will choose a source node at the time a scheduled shipping is made. [OPTIONAL]

<2> You can set here either the resource group or the resource definition (but not both) that
you want to enable the backup shipping schedule for. If you omit this option from the command,
then the command enables scheduled backup shipping for all deployed LINSTOR resources that can
make snapshots. [OPTIONAL]

<3> The name of the remote destination that you want to ship backups to. [REQUIRED]

<4> The name of a previously created backup shipping schedule. [REQUIRED]

##### Disabling a Backup Shipping Schedule

To disable a previously enabled backup shipping schedule, you use the LINSTOR client `backup
schedule disable` command. The command has the following syntax:

```bash
# linstor backup schedule disable \
  [--rg __resource_group_name__ | --rd __resource_definition_name__] \
  __remote_name__ \ <3>
  __schedule_name__ <4>
```

If you include the option specifying either a resource group or resource definition, as
described in the `backup schedule enable` command example above, then you disable the schedule
only for that resource group or resource definition.

For example, if you omitted specifying a resource group or resource definition in an earlier
`backup schedule enable` command, LINSTOR would schedule backup shipping for all its deployed
resources that can make snapshots. Your disable command would then only affect the resource
group or resource definition that you specify with the command. The backup shipping schedule
would still apply to any deployed LINSTOR resources besides the specified resource group or
resource definition.

The same as for the `backup schedule enable` command, if you specify neither a resource group
nor a resource definition, then LINSTOR disables the backup shipping schedule at the controller
level for all deployed LINSTOR resources.

##### Deleting Aspects of a Backup Shipping Schedule

You can use the `linstor backup schedule delete` command to granularly delete either a specified
resource definition or a resource group from a backup shipping schedule, without deleting the
schedule itself. This command has the same syntax and arguments as the `backup schedule disable`
command. If you specify neither a resource group nor a resource definition, the backup shipping
schedule you specify will be deleted at the controller level.

It may be helpful to think about the `backup schedule delete` command as a way that you can
*remove* a backup shipping schedule-remote pair from a specified LINSTOR object level, either a
resource definition, a resource group, or at the controller level if neither is specified.

The `backup schedule delete` command does not affect previously created snapshots or
successfully shipped backups. These will be retained until they are manually deleted, or until
they are removed by the effects of a still applicable keep-local or keep-remote option.

You might want to use this command when you have disabled a backup schedule for multiple LINSTOR
object levels and later want to affect a granular change, where a `backup schedule enable`
command might have unintended consequences.

For example, consider a scenario where you have a backup schedule-remote pair that you enabled
at a controller level. This controller has a resource group, _myresgroup_ that has several
resource definitions, _resdef1_ through _resdef9_, under it. For maintenance reasons perhaps,
you disable the schedule for two resource definitions, _resdef1_ and _resdef2_. You then realize
that further maintenance requires that you disable the backup shipping schedule at the resource
group level, for your _myresgroup_ resource group.

After completing some maintenance, you are able to enable the backup shipping schedule for
_resdef3_ through _resdef9_, but you are not yet ready to resume (enable) backup shipping for
_resdef1_ and _resdef2_. You can enable backup shipping for each resource definition
individually, _resdef3_ through _resdef9_, or you can use the `backup schedule delete` command
to delete the backup shipping schedule from the resource group, _myresgroup_. If you use the
`backup schedule delete` command, backups of _resdef3_ through _resdef9_ will ship again because
the backup shipping schedule is enabled at the controller level, but _resdef1_ and _resdef2_
will not ship because the backup shipping schedule is still disabled for them at the resource
definition level.

When you complete your maintenance and are again ready to ship backups for _resdef1_ and
_resdef2_, you can delete the backup shipping schedule for those two resource definitions to
return to your starting state: backup shipping scheduled for all LINSTOR deployed resources at
the controller level. To visualize this it may be helpful to refer to the decision tree diagram
for how LINSTOR decides whether or not to ship a backup in the
<<s-linstor-how-controller-determines-backup-shipping>> subsection.

NOTE: In the example scenario above, you might have enabled backup shipping on the resource
group, after completing some maintenance. In this case, backup shipping would resume for
resource definitions _resdef3_ through _resdef9_ but continue not to ship for resource
definitions _resdef1_ and _resdef2_ because backup shipping was still disabled for those
resource definitions. After you completed all maintenance, you could delete the backup shipping
schedule on _resdef1_ and _resdef2_. Then all of your resource definitions would be shipping
backups, as they were prior to your maintenance, because the schedule-remote pair was enabled at
the resource group level. However, this would remove your option to globally stop all scheduled
shipping at some later point in time at the controller level because the enabled schedule at the
resource group level would override any `schedule disable` command applied at the controller
level.

##### Listing Backup Shipping Schedules by Resource

You can list backup schedules by resource, using the LINSTOR client `schedule list-by-resource`
command. This command will show LINSTOR resources and how any backup shipping schedules apply
and to which remotes they are being shipped. If resources are not being shipped then the command
will show:

- Whether resources have no schedule-remote-pair entries (empty cells)

- Whether they have schedule-remote-pair entries but they are disabled ("disabled")

- Whether they have no resources, so no backup shipments can be made, regardless of whether any
  schedule-remote-pair entries are enabled or not ("undeployed")

If resources have schedule-remote-pairs and are being shipped, the command output will show when
the last backup was shipped and when the next backup is scheduled to ship. It will also show
whether the next and last backup shipments were full or incremental backups. Finally, the
command will show when the next planned incremental (if any) and full backup shipping will
occur.

You can use the `--active-only` flag with the `schedule list-by-resource` command to filter out all resources that are not being shipped.

##### How the LINSTOR Controller Determines Scheduled Backup Shipping

To determine if the LINSTOR Controller will ship a deployed LINSTOR resource with a certain
backup schedule for a given remote destination, the LINSTOR Controller uses the following logic:

image::images/linstor-controller-backup-schedule-decision-flowchart.svg[]

As the diagram shows, enabled or disabled backup shipping schedules have effect in the following order:

- Resource definition level

- Resource group level

- Controller level

A backup shipping schedule-remote pair that is enabled or disabled at a preceding level will override the enabled or disabled status for the same schedule-remote pair at a later level.

##### Determining How Scheduled Backup Shipping Affects a Resource

To determine how a LINSTOR resource will be affected by scheduled backup shipping, you can use
the LINSTOR client `schedule list-by-resource-details` command for a specified LINSTOR resource.

The command will output a table that shows on what LINSTOR object level a backup shipping
schedule is either not set (empty cell), enabled, or disabled.

By using this command, you can determine on which level you need to make a change to enable,
disable, or delete scheduled backup shipping for a resource.

Example output could look like this:

```bash
# linstor schedule list-by-resource-details __my_linstor_resource_name__
╭───────────────────────────────────────────────────────────────────────────╮
┊ Remote   ┊ Schedule   ┊ Resource-Definition ┊ Resource-Group ┊ Controller ┊
╞═══════════════════════════════════════════════════════════════════════════╡
┊ rem1     ┊ sch1       ┊ Disabled            ┊                ┊ Enabled    ┊
┊ rem1     ┊ sch2       ┊                     ┊ Enabled        ┊            ┊
┊ rem2     ┊ sch1       ┊ Enabled             ┊                ┊            ┊
┊ rem2     ┊ sch5       ┊                     ┊ Enabled        ┊            ┊
┊ rem3     ┊ sch4       ┊                     ┊ Disabled       ┊ Enabled    ┊
╰───────────────────────────────────────────────────────────────────────────╯
```
