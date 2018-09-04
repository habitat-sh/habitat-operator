# Runtime binding

This demonstrates how to provide the contents of the Habitat Service files directory via a Kubernetes Secret.


## Workflow

After the Habitat operator is up and running, execute the following command from the root of this repository:

```
kubectl create -f examples/files-volume/habitat.yml
```

This will deploy redis and also put a file containing a password into `hab/svc/redis/files/pwfile`.

