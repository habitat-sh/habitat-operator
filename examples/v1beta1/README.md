# V1beta1 example

This example demonstrates how to create a Habitat object using the old v1beta1
type.

Since it's not currently possible to run multiple versions of a CRD at the same
time, we defined the `customVersion` field, and let controllers decide which
objects to act upon based on its value (e.g. the `v1beta2` controller will
ignore `Habitat` objects whose `customVersion` field is not `v1beta2`.

The absence of the field is interpreted as `v1beta`, for backwards-compatibility
with existing objects.
