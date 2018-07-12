# Setting up CircleCI and GCP for Habitat Operator e2e test
This document explains how to set up and configure Circle CI and Google Cloud for e2e tests to run successfully.

## Prerequisites
* A CircleCI 2.0 project.
* A Google account.
* A Google Cloud Platform project.

## Steps
### Create service account
To do this, you will need to create a [service account](https://developers.google.com/identity/protocols/OAuth2ServiceAccount).

* Open the [Service accounts](https://console.developers.google.com/iam-admin/serviceaccounts) page. If prompted, select a project.
* Click `Create service account`.
* In the `Create service account` window, type a name for the service account, and select `Furnish a new private key`. Then click `Save`.

Your new public/private key pair is generated and downloaded to your machine; it serves as the only copy of this key. **You are responsible for storing it securely**.

### Add Service account to CircleCI Environment
* Copy the contents of the JSON file you downloaded to the clipboard.
* In the CircleCI application, go to your project’s settings by clicking the gear icon on the top right.
* In the `Build Settings` section, click `Environment Variables`, then click the `Add Variable` button.
* Name the variable. For the Habitat Operator project, the variable is named `GCLOUD_SERVICE_KEY`.
* Paste the JSON file from the first step into the `Value` field.
* Click the `Add Variable` button.

Also, add this [environment variable](https://circleci.com/docs/2.0/env-vars/) to your project:
* GCLOUD_PROJECT_ID: the ID of your GCP project

### Add permissions on Google cloud
On Google Cloud IAM console, add the following roles to the service account you created in the first step:

* Service Account User
* Storage Admin
* CircleCI: This is a custom role (can be called anything else you prefer) which you should create with the following permissions:
    ```
    container.clusterRoleBindings.create
    container.clusterRoleBindings.get
    container.clusterRoleBindings.list
    container.clusterRoles.bind
    container.clusterRoles.create
    container.clusterRoles.get
    container.clusters.create
    container.clusters.delete
    container.clusters.get
    container.namespaces.create
    container.namespaces.get
    container.namespaces.list
    container.nodes.list
    container.operations.get
    container.pods.list
    container.replicationControllers.list
    container.serviceAccounts.create
    container.serviceAccounts.get
    container.serviceAccounts.list
    container.services.get
    container.services.list
    ```

## Issues encountered
### Error when creating Role Based Access Control
On the Circle CI config file, we granted the user the [ability to create roles](https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control) in Kubernetes by running the following command:
```
kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user $(gcloud config get-value account)
```
This was done to fix the following errors:
```
Error from server (NotFound): error when creating "examples/rbac/rbac.yml": clusterroles.rbac.authorization.k8s.io "habitat-operator" not found
Error from server (Forbidden): error when creating "examples/rbac/rbac.yml": clusterroles.rbac.authorization.k8s.io "habitat-operator" is forbidden: attempt to grant extra privileges: …
```

### No Auth Provider found for name "gcp"
k8s.io/client-go/plugin/pkg/client/auth/gcp package was added to fix the error
```
No Auth Provider found for name “gcp”
```

### Error with using outdated gcloud tools
Updating gcloud tools fixed this error when trying to configure docker to use gcloud to authenticate requests to Container Registry.
```
(gcloud.auth) Invalid choice: 'configure-docker'
```

### Issue with bind-config service type
Initially, when the e2e tests was run with minikube, the bind-config service type was `NodePort`. With that, CircleCI was unable to access the service on GKE. It was changed to type `LoadBalancer` to expose the service, and the ephemeral LoadBalancer IP was picked up after it had been generated on GKE.

## Existing issues
### Cancelling CI build
When a build gets cancelled on Circle CI, the images and clusters are not deleted on GCP. Pending when we figure out a way to automate cleaning up the resources created, you would need to manually do that yourself if you cancel a build.
