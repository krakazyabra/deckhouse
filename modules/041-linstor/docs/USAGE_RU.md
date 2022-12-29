---
title: "Модуль linstor: примеры конфигурации"
---

## Создание снапшотов

Этот модуль поддерживает создание снапшотов. Процедура создания снапшотов является общей и описана в [модуле snapshot-controller](../045-snapshot-controller/usage.html).

Однако у LINSTOR есть дополнительная функция, позволяющая создавать моментальные снимки на удаленном хранилище S3.
Подробная инструкция доступна в разделе [Расширенная настройка LINSTOR](advanced_usage.html#Backup-on-S3-Storage).

## Использование планировщика linstor

Планировщик `linstor` учитывает размещение данных в хранилище и старается размещать Pod в первую очередь на тех узлах, где данные доступны локально. Включается добавлением параметра `schedulerName: linstor` в описание Pod'а приложения.

Пример описания Pod'а, использующего планировщик `linstor`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: busybox
  namespace: default
spec:
  schedulerName: linstor # Использование планировщика linstor
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
