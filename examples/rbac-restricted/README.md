# Restricted RBAC policies

The policies defined in this directory are restricted, which limit the operator to have view of only one namespace.

## Workflow

Before deploying the Habitat operator inside your Kubernetes cluster the following roles must be created:

    kubectl apply -f examples/rbac-restricted/rbac-restricted.yml


If you're running the operator on minikube, the `minikube.yml` manifest sets up the required RBAC rules.

    kubectl apply -f examples/rbac-restricted/minikube.yml


Since the operator has less permissions, you(as cluster admin) need to create the CRD.

    kubectl apply -f examples/rbac-restricted/crd.yml


Once those roles were successfully created the Habitat operator can be deployed in the cluster:

    kubectl apply -f examples/rbac-restricted/habitat-operator.yml

