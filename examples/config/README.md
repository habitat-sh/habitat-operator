# User configuration

This examples demonstrates how to leverage the `user.toml` configuration
mechanism for Habitat Services within the operator.

When a `user.toml` file is located at `/hab/user/$servicename/config/`, it is
automatically loaded by the supervisor. We leverage Kubernetes Secrets to mount
a file in that path.

NOTE: Adding secret configuration to the `default.toml` is discouraged, as it will be uploaded as a docker image. Instead use the initial configuration `user.toml` file.

## Workflow

After the Habitat operator is up and running, execute the following command from the root of this repository:

`kubectl create -f examples/config/habitat.yml`

This will create a [Kubernetes Secret](https://kubernetes.io/docs/concepts/configuration/secret/) with the configurations and a Redis database.

By default, Redis listens on port 6379, but we change this to 6999 by mounting a
Secret as a file under `/hab/user/redis/config/user.toml` inside the Pod.

You can see this is the case by accessing the web app on port `30001`. When
running on minikube, its IP can be retrieved with `minikube ip`.

## Configuration updates

In Kubernetes, a file backed by a Secret is kept in sync with the Secret: when
the Secret changes, the file does as well (after a short delay).

You can try this yourself by editing the Secret:

    echo "port = 6160" | base64
    kubectl edit secret user-toml

As you type the last command, your default editor will be invoked. You should
then paste the new base64-encoded string under `data.user-toml`, and exit the
editor.

After a short delay, you should be able to see port 6160 being used by the Redis
service.

## Deletion

The Habitat operator does not delete the Secret on Habitat deletion, as it is not managed by the Habitat operator.
To manually delete the Secret simply run:

```
kubectl delete service user-toml-secret
```
