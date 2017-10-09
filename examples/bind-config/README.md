# Runtime binding + initial configuration

This demonstrates how to run two Habitat Services with a [binding](https://www.habitat.sh/docs/run-packages-binding/) between them, with initial configuration used to override the port of the PostgreSQL Habitat service. It also displays how different fields in the manifest file can be combined.

## Workflow

After the Habitat operator is up and running, execute the following command from the root of this repository:

```
kubectl create -f examples/bind+config/service_group.yml
```

This will deploy two `ServiceGroup`s, a simple HTTP server written in Go that will be bound to a PostgreSQL database. The Go server will display the database port number that was overriden by the initial configuration.

When running on minikube, it can be accessed under port `5555` of the minikube VM. `minikube ip` can be used to retrieve the IP.

