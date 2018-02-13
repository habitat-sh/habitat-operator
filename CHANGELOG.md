# Habitat operator CHANGELOG

## [v0.5.0](https://github.com/kinvolk/habitat-operator/tree/v0.5.0) (1-2-2018)
[Full changelog](https://github.com/kinvolk/habitat-operator/compare/v0.4.0...v0.5.0)

### Breaking changes

- API downgrade from "v1" to "v1beta1" to better reflect the API instability [#167](https://github.com/kinvolk/habitat-operator/pull/167)
- Add Name field [#155](https://github.com/kinvolk/habitat-operator/pull/155)

Please refer to examples for how to adapt existing manifests.

### Features & Enhancements

- Updated mount path of user.toml config file as per [Habitat change](https://github.com/habitat-sh/habitat/pull/3814)
- Update user.toml path and example images [#172](https://github.com/kinvolk/habitat-operator/pull/172)
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

