# Habitat operator CHANGELOG

## [v0.8.1](https://github.com/habitat-sh/habitat-operator/tree/v0.8.1) (14-9-2018)
[Full changelog](https://github.com/habitat-sh/habitat-operator/compare/v0.8.0...v0.8.1)

### Bug fixes

- Revert Helm Chart value to deploy operator clusterwide [#360](https://github.com/habitat-sh/habitat-operator/pull/360)


## [v0.8.0](https://github.com/habitat-sh/habitat-operator/tree/v0.8.0) (11-9-2018)
[Full changelog](https://github.com/habitat-sh/habitat-operator/compare/v0.7.2...v0.8.0)

### Bug fixes

- RBAC: Remove permission for Deployments with removal of Habitat spec v1beta1 [#339](https://github.com/habitat-sh/habitat-operator/pull/339)


### Features & Enhancements

- Added new topology label `operator.habitat.sh/topology` (see deprecation section for information about the older label) [#332](https://github.com/habitat-sh/habitat-operator/pull/332)
- controller: change generated StatefulSet version from v1beta1 to v1 [#334](https://github.com/habitat-sh/habitat-operator/pull/334)
- Add support for Kubernetes 1.11 [#340](https://github.com/habitat-sh/habitat-operator/pull/340)
- RBAC: Harden RBAC policies for the operator enabling operator to run in two modes, clusterwide and namespaced [#346](https://github.com/habitat-sh/habitat-operator/pull/346)
- RBAC version: update all artifacts to rbac.authorization.k8s.io/v1 [#353](https://github.com/habitat-sh/habitat-operator/pull/353)
- Display version when booting operator [#350](https://github.com/habitat-sh/habitat-operator/pull/350)


### Deprecations

- Habitat CRD spec: drop support for v1beta1 [#331](https://github.com/habitat-sh/habitat-operator/pull/331)
- Topology label `topology` is deprecated and will be removed in two releases [#332](https://github.com/habitat-sh/habitat-operator/pull/332)
- Drop support of Kubernetes 1.8 [#344](https://github.com/habitat-sh/habitat-operator/pull/344)


## [v0.7.2](https://github.com/habitat-sh/habitat-operator/tree/v0.7.2) (12-7-2018)
[Full changelog](https://github.com/habitat-sh/habitat-operator/compare/v0.7.1...v0.7.2)

### Bug fixes

- Fix outdated helm charts [#317](https://github.com/habitat-sh/habitat-operator/pull/317)

### Features & Enhancements

- Remove restriction to deploy at least 3 instances in leader topology [#299](https://github.com/habitat-sh/habitat-operator/pull/299)
- Check if RBAC rules are in sync during the build process [#319](https://github.com/habitat-sh/habitat-operator/pull/319)

### Docs

- Add document for CI setup with GCP [#310](https://github.com/habitat-sh/habitat-operator/pull/310)

### Deprecations

- 0.7.x is the last version to support v1beta1 custom version of the Habitat CRD.

## [v0.7.1](https://github.com/habitat-sh/habitat-operator/tree/v0.7.1) (10-7-2018)
[Full changelog](https://github.com/habitat-sh/habitat-operator/compare/v0.7.0...v0.7.1)

### Bug fixes

- Fix usage of LabelSelector for deleting pods on updates [#314](https://github.com/habitat-sh/habitat-operator/pull/314)

### Deprecations

- 0.7.x is the last version to support v1beta1 custom version of the Habitat CRD.

## [v0.7.0](https://github.com/habitat-sh/habitat-operator/tree/v0.7.0) (10-7-2018)
[Full changelog](https://github.com/habitat-sh/habitat-operator/compare/v0.6.1...v0.7.0)

### Bug fixes

- Mark items as done after validity check [#274](https://github.com/habitat-sh/habitat-operator/pull/274)
- Print custom version string instead of pointer to make error messages useful [#275](https://github.com/habitat-sh/habitat-operator/pull/275)

### Features & Enhancements

- Add "Channel" to Habitat CRD for Habitat packages [#259](https://github.com/habitat-sh/habitat-operator/pull/259)
- Add support for Kubernetes 1.10 [#258](https://github.com/habitat-sh/habitat-operator/pull/258)
- Broadcast events when Habitat objects are modified or fail validation [#267](https://github.com/habitat-sh/habitat-operator/pull/267)
- Remove dependency on pflag library [#271](https://github.com/habitat-sh/habitat-operator/pull/271)
- Compare Resource Versions on update [#272](https://github.com/habitat-sh/habitat-operator/pull/272)
- Shutdown gracefully through double signal handling [#285](https://github.com/habitat-sh/habitat-operator/pull/285)
- Run E2E tests on GCE [#302](https://github.com/habitat-sh/habitat-operator/pull/302)
- Enable test for persistent storage [#303](https://github.com/habitat-sh/habitat-operator/pull/303)
- Run E2E tests on multiple versions of kubernetes [#306](https://github.com/habitat-sh/habitat-operator/pull/306)
- Delete pods if StatefulSet object is updated [#307](https://github.com/habitat-sh/habitat-operator/pull/307)

### Docs

- Add instructions to update & verify scripts [#255](https://github.com/habitat-sh/habitat-operator/pull/255)
- Add design document [#253](https://github.com/habitat-sh/habitat-operator/pull/253)

### Deprecations

- This is the last version to support v1beta1 custom version of the Habitat CRD.

### Breaking changes

- Support for Kubernetes 1.7 has been dropped [#258](https://github.com/habitat-sh/habitat-operator/pull/258)

## [v0.6.1](https://github.com/kinvolk/habitat-operator/tree/v0.6.1) (23-4-2018)
[Full changelog](https://github.com/kinvolk/habitat-operator/compare/v0.6.0...v0.6.1)

### Bug fixes

- Fix RBAC rules used by helm charts [#249](https://github.com/habitat-sh/habitat-operator/pull/249)

## [v0.6.0](https://github.com/kinvolk/habitat-operator/tree/v0.6.0) (20-4-2018)
[Full changelog](https://github.com/kinvolk/habitat-operator/compare/v0.5.1...v0.6.0)

### Features & Enhancements

- New "version" of the habitat CRD that uses StatefulSets, has persistence functionality. The previous version that uses Deployments is still supported but is frozen [#201](https://github.com/habitat-sh/habitat-operator/pull/201) [#240](https://github.com/habitat-sh/habitat-operator/pull/240)
- Oldest supported Habitat Supervisor is 0.52 [#190](https://github.com/habitat-sh/habitat-operator/pull/190)

Please compare the `v1beta1` and `v1beta2` manifests of the standalone example in `examples/v1beta1/habitat.yml` and `examples/standalone/habitat.yml`, respectively, to compare the immediate differences between them. Please refer to `examples/persisted/habitat.yml` for an example of the persistence functionality.

### Deprecations

* `Habitat` Manifests that do not specify a `customVersion`, or that specify a
`customVersion = v1beta1` are deprecated, and support for them will be removed
when Kubernetes 1.11 is released. Please upgrade your manifests to the latest
`customVersion`.

## [v0.5.1](https://github.com/kinvolk/habitat-operator/tree/v0.5.1) (14-2-2018)
[Full changelog](https://github.com/kinvolk/habitat-operator/compare/v0.5.0...v0.5.1)

- Fix versions in example files [#188](https://github.com/kinvolk/habitat-operator/pull/188)

## [v0.5.0](https://github.com/kinvolk/habitat-operator/tree/v0.5.0) (13-2-2018)
[Full changelog](https://github.com/kinvolk/habitat-operator/compare/v0.4.0...v0.5.0)

### Breaking changes

- API downgrade from "v1" to "v1beta1" to better reflect the API instability [#167](https://github.com/kinvolk/habitat-operator/pull/167)
- Add Name field [#155](https://github.com/kinvolk/habitat-operator/pull/155)

Please refer to examples for how to adapt existing manifests.

### Features & Enhancements

- Add support for passing environment variables to the supervisor
[#184](https://github.com/kinvolk/habitat-operator/pull/184)
- Update mount path of `user.toml` config file as per [Habitat change](https://github.com/habitat-sh/habitat/pull/3814)
- Update `user.toml` path and example images [#172](https://github.com/kinvolk/habitat-operator/pull/172)
- Use cache for ConfigMaps [#157](https://github.com/kinvolk/habitat-operator/pull/157)
- Add helm chart for the operator [#161](https://github.com/kinvolk/habitat-operator/pull/161)

## [v0.4.0](https://github.com/kinvolk/habitat-operator/tree/v0.4.0) (5-1-2018)
[Full changelog](https://github.com/kinvolk/habitat-operator/compare/v0.3.0...v0.4.0)

### Features & Enhancements

- Update to client-go v6.0.0. to make operator work with Kubernetes 1.9.x [#146](https://github.com/kinvolk/habitat-operator/pull/146)
- Conform to upstream controllers directory structure [#147](https://github.com/kinvolk/habitat-operator/pull/147)

## [v0.3.0](https://github.com/kinvolk/habitat-operator/tree/v0.3.0) (19-12-2017)
[Full changelog](https://github.com/kinvolk/habitat-operator/compare/v0.2.0...v0.3.0)

### Features & Enhancements

- Start multiple parallel queue workers [#136](https://github.com/kinvolk/habitat-operator/pull/136)
- Wait for caches to be synced before starting workers [#134](https://github.com/kinvolk/habitat-operator/pull/134)
- Explicitly stop workers when the controller stops [#135](https://github.com/kinvolk/habitat-operator/pull/135)

### Bug fixes

- Prevent panic due to enqueueing `nil` objects [#133](https://github.com/kinvolk/habitat-operator/pull/133)

## [v0.2.0](https://github.com/kinvolk/habitat-operator/tree/v0.2.0) (20-11-2017)
[Full changelog](https://github.com/kinvolk/habitat-operator/compare/v0.1.0...v0.2.0)

### Features & Enhancements

- React to all events involved with a Habitat object for faster and more consistent reconciliation with the desired state. [#113](https://github.com/kinvolk/habitat-operator/pull/113)
- Update Deployment when Habitat object is updated [#124](https://github.com/kinvolk/habitat-operator/pull/124)

### Bug fixes

- Fix Habitat removal [#125](https://github.com/kinvolk/habitat-operator/pull/125)
