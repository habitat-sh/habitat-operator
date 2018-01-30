# Runtime binding

This demonstrates how to run two Habitat Services with a [binding](https://www.habitat.sh/docs/run-packages-binding/) between them.

## Workflow

After the Habitat operator is up and running, execute the following command from the root of this repository:

```
kubectl create -f examples/bind/habitat.yml
```

This will deploy two `Habitat`s, a simple HTTP server written in Go that will be bound to a Redis database. The Go server will display the port number the database listens on.

When running on minikube, it can be accessed under port `30001` of the minikube VM. `minikube ip` can be used to retrieve the IP.
