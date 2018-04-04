// Copyright (c) 2018 Chef Software Inc. and/or applicable contributors
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
package v1beta2

import (
	v1beta2 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta2"
	scheme "github.com/habitat-sh/habitat-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// HabitatsGetter has a method to return a HabitatInterface.
// A group's client should implement this interface.
type HabitatsGetter interface {
	Habitats(namespace string) HabitatInterface
}

// HabitatInterface has methods to work with Habitat resources.
type HabitatInterface interface {
	Create(*v1beta2.Habitat) (*v1beta2.Habitat, error)
	Update(*v1beta2.Habitat) (*v1beta2.Habitat, error)
	UpdateStatus(*v1beta2.Habitat) (*v1beta2.Habitat, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1beta2.Habitat, error)
	List(opts v1.ListOptions) (*v1beta2.HabitatList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta2.Habitat, err error)
	HabitatExpansion
}

// habitats implements HabitatInterface
type habitats struct {
	client rest.Interface
	ns     string
}

// newHabitats returns a Habitats
func newHabitats(c *HabitatV1beta2Client, namespace string) *habitats {
	return &habitats{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the habitat, and returns the corresponding habitat object, and an error if there is any.
func (c *habitats) Get(name string, options v1.GetOptions) (result *v1beta2.Habitat, err error) {
	result = &v1beta2.Habitat{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("habitats").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Habitats that match those selectors.
func (c *habitats) List(opts v1.ListOptions) (result *v1beta2.HabitatList, err error) {
	result = &v1beta2.HabitatList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("habitats").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested habitats.
func (c *habitats) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("habitats").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a habitat and creates it.  Returns the server's representation of the habitat, and an error, if there is any.
func (c *habitats) Create(habitat *v1beta2.Habitat) (result *v1beta2.Habitat, err error) {
	result = &v1beta2.Habitat{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("habitats").
		Body(habitat).
		Do().
		Into(result)
	return
}

// Update takes the representation of a habitat and updates it. Returns the server's representation of the habitat, and an error, if there is any.
func (c *habitats) Update(habitat *v1beta2.Habitat) (result *v1beta2.Habitat, err error) {
	result = &v1beta2.Habitat{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("habitats").
		Name(habitat.Name).
		Body(habitat).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *habitats) UpdateStatus(habitat *v1beta2.Habitat) (result *v1beta2.Habitat, err error) {
	result = &v1beta2.Habitat{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("habitats").
		Name(habitat.Name).
		SubResource("status").
		Body(habitat).
		Do().
		Into(result)
	return
}

// Delete takes name of the habitat and deletes it. Returns an error if one occurs.
func (c *habitats) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("habitats").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *habitats) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("habitats").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched habitat.
func (c *habitats) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta2.Habitat, err error) {
	result = &v1beta2.Habitat{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("habitats").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
