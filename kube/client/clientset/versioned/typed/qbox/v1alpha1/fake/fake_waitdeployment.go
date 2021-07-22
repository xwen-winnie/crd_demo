/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/xwen-winnie/crd_demo/kube/apis/qbox/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeWaitdeployments implements WaitdeploymentInterface
type FakeWaitdeployments struct {
	Fake *FakeQboxV1alpha1
	ns   string
}

var waitdeploymentsResource = schema.GroupVersionResource{Group: "qbox", Version: "v1alpha1", Resource: "waitdeployments"}

var waitdeploymentsKind = schema.GroupVersionKind{Group: "qbox", Version: "v1alpha1", Kind: "Waitdeployment"}

// Get takes name of the waitdeployment, and returns the corresponding waitdeployment object, and an error if there is any.
func (c *FakeWaitdeployments) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Waitdeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(waitdeploymentsResource, c.ns, name), &v1alpha1.Waitdeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Waitdeployment), err
}

// List takes label and field selectors, and returns the list of Waitdeployments that match those selectors.
func (c *FakeWaitdeployments) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.WaitdeploymentList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(waitdeploymentsResource, waitdeploymentsKind, c.ns, opts), &v1alpha1.WaitdeploymentList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.WaitdeploymentList{ListMeta: obj.(*v1alpha1.WaitdeploymentList).ListMeta}
	for _, item := range obj.(*v1alpha1.WaitdeploymentList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested waitdeployments.
func (c *FakeWaitdeployments) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(waitdeploymentsResource, c.ns, opts))

}

// Create takes the representation of a waitdeployment and creates it.  Returns the server's representation of the waitdeployment, and an error, if there is any.
func (c *FakeWaitdeployments) Create(ctx context.Context, waitdeployment *v1alpha1.Waitdeployment, opts v1.CreateOptions) (result *v1alpha1.Waitdeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(waitdeploymentsResource, c.ns, waitdeployment), &v1alpha1.Waitdeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Waitdeployment), err
}

// Update takes the representation of a waitdeployment and updates it. Returns the server's representation of the waitdeployment, and an error, if there is any.
func (c *FakeWaitdeployments) Update(ctx context.Context, waitdeployment *v1alpha1.Waitdeployment, opts v1.UpdateOptions) (result *v1alpha1.Waitdeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(waitdeploymentsResource, c.ns, waitdeployment), &v1alpha1.Waitdeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Waitdeployment), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeWaitdeployments) UpdateStatus(ctx context.Context, waitdeployment *v1alpha1.Waitdeployment, opts v1.UpdateOptions) (*v1alpha1.Waitdeployment, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(waitdeploymentsResource, "status", c.ns, waitdeployment), &v1alpha1.Waitdeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Waitdeployment), err
}

// Delete takes name of the waitdeployment and deletes it. Returns an error if one occurs.
func (c *FakeWaitdeployments) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(waitdeploymentsResource, c.ns, name), &v1alpha1.Waitdeployment{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeWaitdeployments) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(waitdeploymentsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.WaitdeploymentList{})
	return err
}

// Patch applies the patch and returns the patched waitdeployment.
func (c *FakeWaitdeployments) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Waitdeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(waitdeploymentsResource, c.ns, name, pt, data, subresources...), &v1alpha1.Waitdeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Waitdeployment), err
}
