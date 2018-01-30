# Initial configuration

This example demonstrates how initial configuration works with the Habitat operator. With the manifest file we deploy a Redis Habitat service.
NOTE: Adding secret configuration to the `default.toml` is discouraged, as it will be uploaded as a docker image. Instead use the initial configuration `user.toml` file.

## Workflow

After the Habitat operator is up and running, execute the following command from the root of this repository:

`kubectl create -f examples/config/habitat.yml`

This will create a [Kubernetes Secret](https://kubernetes.io/docs/concepts/configuration/secret/) with the configurations and a Redis database.
Initially the Redis database is configured to be in protected mode. Because we override this with the Secret we just created, our db will not be in this mode anymore.

By default, Redis listens on port 6379, but we change this to 6999 by mounting a
Secret as a file under `/hab/user/redis/config/user.toml` inside the Pod.

The web app is listening on port `30001`. When running on minikube, its IP can
be retrieved with `minikube ip`.

## Deletion

The Habitat operator does not delete the Secret on Habitat deletion, as it is not managed by the Habitat operator.
To manually delete the Secret simply run:

```
kubectl delete service user-toml-secret
```
