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
package fake

import (
	v1beta1 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeHabitats implements HabitatInterface
type FakeHabitats struct {
	Fake *FakeHabitatV1beta1
	ns   string
}

var habitatsResource = schema.GroupVersionResource{Group: "habitat.sh", Version: "v1beta1", Resource: "habitats"}

var habitatsKind = schema.GroupVersionKind{Group: "habitat.sh", Version: "v1beta1", Kind: "Habitat"}

// Get takes name of the habitat, and returns the corresponding habitat object, and an error if there is any.
func (c *FakeHabitats) Get(name string, options v1.GetOptions) (result *v1beta1.Habitat, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(habitatsResource, c.ns, name), &v1beta1.Habitat{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Habitat), err
}

// List takes label and field selectors, and returns the list of Habitats that match those selectors.
func (c *FakeHabitats) List(opts v1.ListOptions) (result *v1beta1.HabitatList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(habitatsResource, habitatsKind, c.ns, opts), &v1beta1.HabitatList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.HabitatList{}
	for _, item := range obj.(*v1beta1.HabitatList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested habitats.
func (c *FakeHabitats) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(habitatsResource, c.ns, opts))

}

// Create takes the representation of a habitat and creates it.  Returns the server's representation of the habitat, and an error, if there is any.
func (c *FakeHabitats) Create(habitat *v1beta1.Habitat) (result *v1beta1.Habitat, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(habitatsResource, c.ns, habitat), &v1beta1.Habitat{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Habitat), err
}

// Update takes the representation of a habitat and updates it. Returns the server's representation of the habitat, and an error, if there is any.
func (c *FakeHabitats) Update(habitat *v1beta1.Habitat) (result *v1beta1.Habitat, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(habitatsResource, c.ns, habitat), &v1beta1.Habitat{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Habitat), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeHabitats) UpdateStatus(habitat *v1beta1.Habitat) (*v1beta1.Habitat, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(habitatsResource, "status", c.ns, habitat), &v1beta1.Habitat{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Habitat), err
}

// Delete takes name of the habitat and deletes it. Returns an error if one occurs.
func (c *FakeHabitats) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(habitatsResource, c.ns, name), &v1beta1.Habitat{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeHabitats) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(habitatsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1beta1.HabitatList{})
	return err
}

// Patch applies the patch and returns the patched habitat.
func (c *FakeHabitats) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.Habitat, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(habitatsResource, c.ns, name, data, subresources...), &v1beta1.Habitat{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Habitat), err
}
