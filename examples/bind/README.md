# Runtime binding

This demonstrates how to run two Habitat Services with a [binding](https://www.habitat.sh/docs/run-packages-binding/) between them.

After the Habitat Operator is up and running, execute the following command from the root of this repository:

```
kubectl create -f examples/bind/service_group.yml
```

This will deploy two `ServiceGroup`s, a simple HTTP server written in Go that will be bound to a PostgreSQL database. The Go server will display the port number the database listens on.

When running on minikube, it can be accessed under port `5555` of the minikube VM. `minikube ip` can be used to retrieve the IP.
