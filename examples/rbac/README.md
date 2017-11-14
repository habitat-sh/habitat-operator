# RBAC Habitat example

[RBAC](https://kubernetes.io/docs/admin/authorization/rbac/) stands for role-based access control, and in Kubernetes it's aimed at limiting the permissions within the cluster. The Habitat operator requires some access within the cluster, for example for creating and managing the `CRD` and all the other necessary resources. This is where the following example comes in. If the API server is started with the `--authorization-mode=RBAC` flag, then the following roles must be created for the Habitat operator to function properly.

## Workflow

Before deploying the Habitat operator inside your cluster the following roles must be created:

    kubectl apply -f examples/rbac/rbac.yml

Once those roles were successfully created the Habitat operator can be deployed in the cluster:

    kubectl apply -f examples/rbac/habitat.yml

