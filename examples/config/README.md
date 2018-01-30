# Initial configuration

This example demostrates how initial configuration works with the Habitat operator. With the manifest file we deploy a Redis Habitat service.
NOTE: Adding secret configuration to the `default.toml` is discouraged, as it will be uploaded as a docker image. Instead use the initial configuration `user.toml` file.

## Workflow

After the Habitat operator is up and running, execute the following command from the root of this repository:

`kubectl create -f examples/config/habitat.yml`

This will create a [Kubernetes Secret](https://kubernetes.io/docs/concepts/configuration/secret/) with the configurations and a Redis database. When running on minikube, it can be accessed under port `30001` of the minikube VM. `minikube ip` can be used to retrieve the IP.
Initially the Redis database is configured to be in protected mode. Because we override this with the Secret we just created, our db will not be in this mode anymore.

## Deletion

The Habitat operator does not delete the Secret on Habitat deletion, as it is not managed by the Habitat operator.
To manually delete the Secret simply run:

```
kubectl delete service user-toml-secret
```
