# Persistent Storage example

Habitat objects are translated by the operator into `StatefulSet` objects, which
provide optional support for persistent storage.

In order to enable persistent storage for your Habitat object, you need to
configure [persistent
storage](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) on
your cluster. Detailed instructions on how to do that are beyond the scope of
this README, but what you should end up with is:

* a cluster configured for either static or dynamic provisioning,
* a `StorageClass` object,

Once you've done that, you can proceed to creating the `Habitat` object:

* populate the `spec.persistentStorage` struct in the `Habitat`'s manifest
* specify the `name` of the previously created `StorageClass` under
`spec.persistence.storageClassName` in the `Habitat`'s manifest

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
