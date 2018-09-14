package rbac

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/renderutil"
)

func TestClusterRolesSync(t *testing.T) {

	exampleRules, err := extractRulesFromClusterRoles("examples/rbac/rbac.yml")
	require.NoError(t, err, "extracting Rules from ClusterRole failed for rbac in examples")

	testRules, err := extractRulesFromClusterRoles("test/e2e/v1beta1/clusterwide/resources/operator/cluster-role.yml")
	require.NoError(t, err, "extracting Rules from ClusterRole failed for rbac in tests")

	helmRules, err := extractRulesFromHelm("helm/habitat-operator/templates/clusterrole.yaml", true)
	require.NoError(t, err, "extracting Rules from Helm ClusterRole failed")

	require.Equal(t, exampleRules, testRules, "ClusterRole from 'example' and 'test' not equal")
	require.Equal(t, testRules, helmRules, "ClusterRole from 'test' and 'helm' not equal")

	t.Log("ClusterRoles are in sync")
}

func TestRolesSync(t *testing.T) {
	exampleRules, err := extractRulesFromRoles("examples/rbac-restricted/rbac-restricted.yml")
	require.NoError(t, err, "extracting Rules from Role failed for rbac in examples")

	testRules, err := extractRulesFromRoles("test/e2e/v1beta1/namespaced/resources/operator/role.yml")
	require.NoError(t, err, "extracting Rules from Role failed for rbac in tests")

	helmRules, err := extractRulesFromHelm("helm/habitat-operator/templates/role.yaml", false)
	require.NoError(t, err, "extracting Rules from Helm Role failed")

	require.Equal(t, exampleRules, testRules, "Role from 'example' and 'test' not equal")
	require.Equal(t, testRules, helmRules, "Role from 'test' and 'helm' not equal")

	t.Log("Roles are in sync")
}

func TestRolesAndClusterRolesSync(t *testing.T) {
	// Assuming all the Roles are in sync with each other and all the test/sync/
	// ClusterRoles are in sync with each other we can just see if
	// the Role and ClusterRole from the example are in sync with each other

	rolesRules, err := extractRulesFromRoles("examples/rbac-restricted/rbac-restricted.yml")
	require.NoError(t, err, "extracting Rules from Role failed for rbac in examples")

	clusterRolesRules, err := extractRulesFromClusterRoles("examples/rbac/rbac.yml")
	require.NoError(t, err, "extracting Rules from ClusterRole failed for rbac in examples")

	// Now we will just remove the two roles that this ClusterRole has extra and try to match
	//    rules:
	//    - apiGroups:
	//      - apiextensions.k8s.io
	//      resources:
	//      - customresourcedefinitions
	//      verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
	//    - apiGroups: [""]
	//      resources:
	//      - namespaces
	//      verbs: ["list"]
	//    - apiGroups:
	//      - habitat.sh
	//      resources:
	//      - habitats
	//      verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
	//
	// The first two rules for CRD and Namespaces are extra permissions that ClusterRole has
	// rest of the permissions are same for Role and ClusterRole so we just remove those two
	// and match if other roles match
	matchingClusterRoleRules := clusterRolesRules[2:]
	require.Equal(t, rolesRules, matchingClusterRoleRules, "Role and ClusterRole are not equal")
	t.Log("Roles and ClusterRoles are in sync")
}

// parse takes in relative path to a file and returns slice of byte slice
// if a yaml file is delimited by --- to contain multiple objects, then this
// separates those objects into multiple byte slices.
// So a yaml file like following
//
// a: b
// ---
// c: d
//
// will be divided into two files one containing `a: b` and other containing
// `c: d`.
func parse(relativePath string) ([][]byte, error) {
	path, err := filepath.Abs(getPathFor(relativePath))
	if err != nil {
		return nil, errors.Wrap(err, "finding absolute path failed")
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "file reading failed")
	}

	var manifests [][]byte
	apps := regexp.MustCompile("(^|\n)---\n").Split(string(data), -1)
	for _, app := range apps {
		if len(strings.TrimSpace(app)) > 0 {
			manifests = append(manifests, []byte(app))
		}
	}
	return manifests, nil
}

// extractRulesFromClusterRoles takes in path to a file that has ClusterRole defined in it
// and returns Rules defined inside it
func extractRulesFromClusterRoles(path string) ([]rbacv1.PolicyRule, error) {
	manifests, err := parse(path)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing file %s failed", path)
	}

	for _, m := range manifests {
		rules, err := parseClusterRoles(m)
		if err != nil {
			continue
		}
		if len(rules) > 0 {
			return rules, nil
		}
	}
	return nil, fmt.Errorf("no ClusterRoles found")
}

// parseClusterRoles takes in a byte array of ClusterRole file and parses it into
// a ClusterRole object and returns Rules from that parsed object
// if the passed in type is not ClusterRole type then the returned Rules will
// be empty slice of PolicyRule.
func parseClusterRoles(d []byte) ([]rbacv1.PolicyRule, error) {
	cr := &rbacv1.ClusterRole{}

	if err := yaml.NewYAMLToJSONDecoder(bytes.NewReader(d)).Decode(cr); err != nil {
		return nil, errors.Wrap(err, "decoding ClusterRole failed")
	}
	return cr.Rules, nil
}

// extractRulesFromHelm takes in path to the Roles or ClusterRoles template and a boolean
// which when set to true it means that the 'path' is ClusterRole else it is Role. This
// function renders the Chart and returns Rules the appopriate file passed in 'path'.
func extractRulesFromHelm(path string, isItClusterRole bool) ([]rbacv1.PolicyRule, error) {
	// Paths are evaluated as follows
	// path: helm/habitat-operator/templates/clusterrole.yaml
	// chartPath: helm/habitat-operator
	// rolePath: habitat-operator/templates/clusterrole.yaml
	paths := strings.Split(path, "/")
	chartPath := strings.Join(paths[:2], "/")
	rolePath := strings.Join(paths[1:], "/")

	chart, err := chartutil.Load(getPathFor(chartPath))
	if err != nil {
		return nil, errors.Wrapf(err, "loading chart %s failed", path)
	}

	// If it is Role then we need to replace the value of `operatorNamespaced` to true
	if !isItClusterRole {
		chart.Values.Raw = strings.Replace(chart.Values.Raw, "operatorNamespaced: false", "operatorNamespaced: true", 1)
	}

	renderedFiles, err := renderutil.Render(chart, chart.Values, renderutil.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "rendering chart failed")
	}

	return parseClusterRoles([]byte(renderedFiles[rolePath]))
}

// extractRulesFromRoles takes in path to a file that has Role defined in it
// and returns Rules defined inside it
func extractRulesFromRoles(path string) ([]rbacv1.PolicyRule, error) {
	manifests, err := parse(path)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing file %s failed", path)
	}

	for _, m := range manifests {
		rules, err := parseRoles(m)
		if err != nil {
			continue
		}
		if len(rules) > 0 {
			return rules, nil
		}
	}
	return nil, fmt.Errorf("no Roles found")
}

// parseRoles takes in a byte array of Role file and parses it into
// a Role object and returns Rules from that parsed object
// if the passed in type is not Role type then the returned Rules will
// be empty slice of PolicyRule.
func parseRoles(d []byte) ([]rbacv1.PolicyRule, error) {
	role := &rbacv1.Role{}

	if err := yaml.NewYAMLToJSONDecoder(bytes.NewReader(d)).Decode(role); err != nil {
		return nil, errors.Wrap(err, "decoding Role failed")
	}
	return role.Rules, nil
}

// When a golang test is run it runs in the package it exist. This
// piece of code changes this and make sure that the path is root of
// the project. This is done to make sure that the resource files
// available in the project at various locations are easily accessible.
// We are here 'test/sync/rbac/' right now, this changes the path
// relative to root.
func getPathFor(path string) string {
	return fmt.Sprintf("../../../%s", path)
}
