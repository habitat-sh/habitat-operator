// Copyright (c) 2017 Chef Software Inc. and/or applicable contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package namespaced

import (
	"flag"
	"fmt"
	"os"
	"testing"

	habv1beta1 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta1"
	of "github.com/habitat-sh/habitat-operator/test/e2e/v1beta1/framework"

	"github.com/pkg/errors"
	aggregateerr "k8s.io/apimachinery/pkg/util/errors"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // Needed for GCP on Circle CI
)

const (
	TestNSNamespaced = "testing-namespaced"
)

var framework *of.Framework

func TestMain(m *testing.M) {
	var (
		err  error
		code int
	)

	image := flag.String("image", "", "habitat operator image, 'habitat/habitat-operator'")
	kubeconfig := flag.String("kubeconfig", "", "path to kube config file")
	externalIP := flag.String("ip", "", "external ip, eg. minikube ip")
	flag.Parse()

	if framework, err = of.Setup(*image, *kubeconfig, *externalIP, TestNSNamespaced); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := setupOperator(framework); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	code = m.Run()

	// write teardown code
	if err = cleanup(framework); err != nil {
		fmt.Printf("Error while cleanup: %v\n", err)
	}

	os.Exit(code)
}

// setupOperator takes care of installing the operator before tests
// are run against it.
func setupOperator(f *of.Framework) error {
	// Setup RBAC for operator
	if err := createRBAC(f); err != nil {
		return errors.Wrap(err, "create RBAC policies for namespaced tests failed")
	}

	// Create CRD
	crd, err := of.ConvertCRD("resources/operator/crd.yml")
	if err != nil {
		return errors.Wrap(err, "convert CRD from yml file failed")
	}

	if _, err = f.APIExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd); err != nil {
		return errors.Wrap(err, "create CRD failed")
	}

	// Get Habitat operator deployment from examples.
	d, err := of.ConvertDeployment("resources/operator/deployment.yml")
	if err != nil {
		return errors.Wrap(err, "convert Deployment from yml file failed")
	}
	// Override image with the one passed to the tests with one provided in cmd line arg
	d.Spec.Template.Spec.Containers[0].Image = f.Image
	// Create deployment for the Habitat operator.
	_, err = f.KubeClient.AppsV1beta1().Deployments(TestNSNamespaced).Create(d)
	if err != nil {
		return errors.Wrap(err, "create Deployment failed")
	}

	if err := f.WaitForResources("name", d.ObjectMeta.Name, 1); err != nil {
		return errors.Wrap(err, "waiting for the namespaced Operator deployment to be ready")
	}
	return nil
}

// createRBAC creates RBAC rules in the cluster necessary for the namespaced
// operator to run smoothly
func createRBAC(f *of.Framework) error {
	// Create ServiceAccount.
	sa, err := of.ConvertServiceAccount("resources/operator/service-account.yml")
	if err != nil {
		return errors.Wrap(err, "convert ServiceAccount from yml file failed")
	}
	_, err = f.KubeClient.CoreV1().ServiceAccounts(TestNSNamespaced).Create(sa)
	if err != nil {
		return errors.Wrap(err, "create ServiceAccount failed")
	}

	// Create Role
	role, err := of.ConvertRole("resources/operator/role.yml")
	if err != nil {
		return errors.Wrap(err, "convert Role from yml file failed")
	}
	if _, err = f.KubeClient.RbacV1().Roles(TestNSNamespaced).Create(role); err != nil {
		return errors.Wrap(err, "create Role failed")
	}

	// Create RoleBinding
	rb, err := of.ConvertRoleBinding("resources/operator/role-binding.yml")
	if err != nil {
		return errors.Wrap(err, "convert RoleBinding from yml file failed")
	}
	if _, err = f.KubeClient.RbacV1().RoleBindings(TestNSNamespaced).Create(rb); err != nil {
		return errors.Wrap(err, "create RoleBinding failed")
	}
	return nil
}

// cleanup deletes all the resources that were created for this test run
func cleanup(f *of.Framework) error {
	var errList []error

	// delete namespace, which will delete all the things created in that ns
	if err := f.DeleteNamespace(); err != nil {
		errList = append(errList, err)
	}

	// delete things that were created at a cluster scope
	// delete CRD
	name := habv1beta1.Kind(habv1beta1.HabitatResourcePlural)
	if err := f.DeleteCRD(name.String()); err != nil {
		errList = append(errList, err)
	}

	if len(errList) > 0 {
		errs := aggregateerr.NewAggregate(errList)
		return fmt.Errorf("%s", errs.Error())
	}
	return nil
}
