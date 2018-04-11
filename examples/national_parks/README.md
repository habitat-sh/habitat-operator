# National Parks

National Parks demo app. A Java web app binding to a MongoDB instance.

## Workflow

After the Habitat operator is up and running, execute the following command from the root of this repository:

```
kubectl create -f examples/national_parks/habitat.yml
```

This will deploy two `Habitat`s, the national parks Java app and a MongoDB instance. By default, the MongoDB database instance would [listen
on port
27017](https://github.com/habitat-sh/core-plans/blob/master/mongodb/default.toml#L59),
but we change this with the configuration stored in the `user-toml`.

The Go web app displays the overridden database port number, and it can be
accessed under port `30001`. When running on minikube, its IP can be retrieved
with `minikube ip`.
