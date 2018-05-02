# Design document

The Habitat operator manages Habitat services on Kubernetes using the operator
pattern. During the initial run it registers the [Custom Resource Definition
(CRD)][crd] for Habitat. The Habitat `CRD` is essentially the schema that
describes the contents of the manifests for deploying individual Habitat objects
using `StatefulSet`s.

Once the operator is running, it performs the following actions:

* Watches for new `Habitat` manifests and deploys corresponding sub-resources
* Watches for updates to existing manifests and changes corresponding properties
* Watches for deletes of the existing manifests and deletes corresponding
  objects
* Periodically checks running Habitat resources against the manifests and acts
  on the differences found

For instance, when the user creates a new custom object of type Habitat by
submitting a new manifest with `kubectl`, the operator fetches that object and
creates the corresponding Kubernetes structures (`StatefulSet`s, `Deployment`s,
`ConfigMap`s, `Secret`s, `PersistentVolume`s) according to its definition.

Another example is changing the Docker image inside a Habitat object. In this
case, the operator first goes to all `StatefulSet`s it manages and updates them
with the new Docker images; afterwards, all `Pod`s from each `StatefulSet` are
killed one by one ([rolling update][rolling]) and the replacements are spawned
automatically by each `StatefulSet` with the new Docker image.

## Service groups

In a non-Kubernetes Habitat setup, every service in a group after the first one
is [started][hab-sg] with the `--peer` flag, pointing to the first service's IP.

Since it's not possible in Kubernetes to launch `Pod`s in a
`Deployment`/`StatefulSet` with different arguments, the operator resorts to
using a `ConfigMap` instead.  `ConfigMap`s can be mounted as files inside
`Pod`s, and Habitat services can be started with the `--peer-watch-file` flag,
which expects a file.  This allows us to start *all* `Pod`s in the `StatefulSet`
with the same set of arguments.

The `ConfigMap` is maintained by the operator in the following ways:

* When it's empty, it populates it with the IP of one of the running `Pod`s
(which `Pod` gets chosen is irrelevant, the important thing is that
the `Pod` is in the `Running` state)
* Whenever a `Pod` dies, it makes sure that the IP in the `ConfigMap` is still
assigned to a running `Pod`, and if not, replaces it

## Sub-resources

As mentioned above, the operator creates several resources whenever it detects a
new `Habitat`. These sub-resources, all labeled with `habitat=true`, are to be
considered read-only; any modifications to them will be overwritten by the
operator.

The proper way to make changes is to **always** make them on the Habitat object,
and let the operator figure out the actual steps it needs to take.

## CRD Versioning

According to [Semantic Versioning][semver], backwards incompatible changes require
releasing a new major version. Unfortunately, it is [currently not
possible][crd-vers] to run multiple versions of a CRD side-by-side. The
workaround currently relies on two steps:

* Defining an additional field, `customVersion`
* Adding a specific key for each version of the spec (i.e. `spec.v1beta2`)

With these workarounds, the operator can support multiple "versions" of the CRD,
meaning that it can have multiple controllers each watching the Habitat CRD,
deciding which objects to operate on based on the `customVersion` field. In
reality though, there's only one version Kubernetes knows about, and we
multiplex our own custom versions in it.

The reason for separating specs by version is to make it possible to introduce
backwards-incompatible changes to existing fields (e.g. changing the type of a
field to pointer type), without affecting existing clusters.

### v1beta1 and v1beta2

The main difference introduced in the v1beta2 custom version is that Habitat
objects are now backed by `StatefulSet`s, rather than `Deployment`s, in order to
offer support for persistence.

Because this is a backwards-incompatible change, it was decided to implement it
under a different custom version.

### Deprecation

Any time a new release of the CRD is released, the previous one is to be
considered deprecated. Currently, we plan on supporting deprecated versions for
a timespan of 3 Kubernetes releases, after which they will be removed.

[crd]: https://kubernetes.io/docs/concepts/api-extension/custom-resources/#customresourcedefinitions
[rolling]: https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#rolling-updates
[crd-vers]: https://github.com/kubernetes/kubernetes/pull/60113/
[semver]: https://semver.org/
[hab-sg]: https://www.habitat.sh/docs/using-habitat/#service-groups
