# Habitat operator CHANGELOG

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

