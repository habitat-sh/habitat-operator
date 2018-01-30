# Runtime binding + initial configuration

This demonstrates how to run two Habitat Services with a [binding](https://www.habitat.sh/docs/run-packages-binding/) between them, with initial configuration used to override the port of the Redis Habitat service. It also displays how different fields in the manifest file can be combined.

## Workflow

After the Habitat operator is up and running, execute the following command from the root of this repository:

```
kubectl create -f examples/bind-config/habitat.yml
```

This will deploy two `Habitat`s, a simple HTTP server written in Go that will be
bound to a Redis instance. By default, the Redis database instance would [listen
on port
6379](https://github.com/habitat-sh/core-plans/blob/7bc934c31e92c959aea0444671900c57c23d5265/redis/default.toml#L3),
but we change this with the configuration stored in the `user-toml`.

The Go web app displays the overridden database port number, and it can be
accessed under port `30001`. When running on minikube, its IP can be retrieved
with `minikube ip`.
