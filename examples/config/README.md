# Initial configuration

This example demostrates how initial configuration works with the Habitat operator. With the manifest file we deploy a `"Hello world."` Node.js Habitat service.
NOTE: Adding secret configuration to the `default.toml` is discouraged, as it will be uploaded as a docker image. Instead use the initial configuration `user.toml` file.

## Workflow

After the Habitat operator is up and running, execute the following command from the root of this repository:

`kubectl create -f /examples/nodejs`

This will create a [Kubernetes Secret](https://kubernetes.io/docs/concepts/configuration/secret/) with the configurations and a simple Node.js application that will display a msg. When running on minikube, it can be accessed under port `30001` of the minikube VM. `minikube ip` can be used to retrieve the IP.
Initially our app is configured to display the msg `"Hello world."`. Because we override this with the Secret we just created, our app will instead display `Hello from our Habitat-Operator!`.

## Deletion

The Habitat operator does not delete the Secret on ServiceGroup deletion, as it is not managed by the Habitat operator.
To manually delete the Secret simply run:

```
kubectl delete service user-toml-secret
```
