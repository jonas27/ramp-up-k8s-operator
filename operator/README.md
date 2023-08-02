# ramp-up-k8s-operator
Simple k8s operator with kubebuilder.

## tasks
* Define a new CRD to deploy the app from the “Deploy app” challenge.

* Create a simple k8s operator to deploy and remove the application created in chapter “Basics”
* The CR to be created should contain fields to configure the deployments:.
  * Configure port server listens on
  * Configure deployment replicas
  * Configure value returned by server
* If the deployment is deleted manually, the controller should recreate it.
  Note: Explore object ownership and finalizers.
* The operator should create or delete all dependent resources if the CR is deleted.
* Update the deployment when the CR is changed, e.g. return a different value from the server.
* Deploy the operator in a kubernetes cluster.

## Commands
```bash
kubebuilder init --domain joe.ionos.io --repo github.com/jonas27/ramp-up-k8s-operator/operator
kubebuilder create api --group ramp-up --version v1alpha1 --kind CharacterCounter

```