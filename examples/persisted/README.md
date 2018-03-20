# Persistent Storage example

Habitat objects are translated by the operator into `StatefulSet` objects, which
provide optional support for persistent storage.

In order to enable persistent storage for your Habitat object, you need to:

* create a
[`StorageClass`](https://kubernetes.io/docs/concepts/storage/storage-classes/) object in your cluster
* add the `spec.persistentStorage` key to the Habitat object's manifest
* specify the `name` of the aforementioned `StorageClass` object under
`spec.persistence.storageClassName` in the Habitat object's manifest

## Workflow

Before deploying the example, create a `StorageClass` object, specifying the
type of volume your cluster is able to provision.

**NOTE**: If you're deploying the example on GKE, a standard
`StorageClass` for `GCEPersistentDisk` has already been defined, so you can skip
the above step

Once the `StorageClass` has been created, run the example:

    kubectl create -f examples/persisted/habitat.yml

When you want to delete the Habitat, run:

    kubectl delete -f examples/persisted/habitat.yml

**NOTE**: Any `PersistentVolume` created by the operator will **NOT** be
automatically removed. This is the default behaviour of Kubernetes and is
intended as a safeguard against accidental data deletion.

If you want to explicitly delete the `PersistentVolume`, run:

    kubectl delete pvc -l habitat-name=example-persistent-habitat
