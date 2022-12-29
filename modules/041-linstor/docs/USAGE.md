---
title: "The linstor module: configuration examples"
---

## Creating snapshots

This module supports snapshots. The snapshots workflow is general and described in [snapshot-controller module](../045-snapshot-controller/usage.html).

However, LINSTOR has extra feature which allows you to create a snapshots on remote S3 storage.
The detailed instruction is available in [Advanced LINSTOR Configuration](advanced_usage.html#Backup-on-S3-Storage)

## Using the linstor scheduler

The linstor scheduler considers the placement of data in storage and tries to place Pods on nodes where data is available locally first.  

Specify the `schedulerName: linstor` parameter in the Pod description to use the `linstor` scheduler.

An example of such a Pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: busybox
  namespace: default
spec:
  schedulerName: linstor # Using the linstor scheduler
  containers:
  - name: busybox
    image: busybox
    command: ["tail", "-f", "/dev/null"]
    volumeMounts:
    - name: my-first-linstor-volume
      mountPath: /data
    ports:
    - containerPort: 80
  volumes:
  - name: my-first-linstor-volume
    persistentVolumeClaim:
      claimName: "test-volume"
```
