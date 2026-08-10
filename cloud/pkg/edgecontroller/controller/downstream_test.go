/*
Copyright 2024 The KubeEdge Authors.

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

package controller

import (
	"testing"

	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
)

func TestSyncRuntimeClassesToNode(t *testing.T) {
	// Create a fake kubernetes client and informer factory
	kubeClient := fake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
	runtimeClassInformer := informerFactory.Node().V1().RuntimeClasses()

	// Add test RuntimeClasses to the informer cache
	testRC1 := &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "kata",
			ResourceVersion: "1",
		},
		Handler: "kata",
	}
	testRC2 := &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "kata2",
			ResourceVersion: "2",
		},
		Handler: "kata2",
	}

	runtimeClassInformer.Informer().GetStore().Add(testRC1)
	runtimeClassInformer.Informer().GetStore().Add(testRC2)

	// Create the MockMessageLayer
	mockMessageLayer := &MockMessageLayer{
		ReceivedMessages: []model.Message{},
		ResponseMessages: []model.Message{},
		SendMessages:     []model.Message{},
	}

	// Initialize the DownstreamController
	dc := &DownstreamController{
		runtimeClassLister: runtimeClassInformer.Lister(),
		messageLayer:       mockMessageLayer,
	}

	nodeName := "edge-node"

	// Call the function
	err := dc.syncRuntimeClassesToNode(nodeName)
	if err != nil {
		t.Fatalf("syncRuntimeClassesToNode failed: %v", err)
	}

	// Verify messages were sent
	if len(mockMessageLayer.SendMessages) != 2 {
		t.Fatalf("expected 2 messages to be sent, got %d", len(mockMessageLayer.SendMessages))
	}

	// Verify the content of the messages
	expectedRCs := map[string]*nodev1.RuntimeClass{
		"kata":  testRC1,
		"kata2": testRC2,
	}

	for _, msg := range mockMessageLayer.SendMessages {
		if msg.GetOperation() != model.InsertOperation {
			t.Errorf("expected operation %s, got %s", model.InsertOperation, msg.GetOperation())
		}

		resType, err := messagelayer.GetResourceType(msg)
		if err != nil {
			t.Fatalf("failed to get resource type from message: %v", err)
		}

		if resType != model.ResourceTypeRuntimeClass {
			t.Errorf("expected resource type %s, got %s", model.ResourceTypeRuntimeClass, resType)
		}

		resName, err := messagelayer.GetResourceName(msg)
		if err != nil {
			t.Fatalf("failed to get resource name from message: %v", err)
		}

		expectedRC, ok := expectedRCs[resName]
		if !ok {
			t.Errorf("unexpected RuntimeClass in message: %s", resName)
			continue
		}

		// Ensure the body holds the expected RuntimeClass object
		rc, ok := msg.GetContent().(*nodev1.RuntimeClass)
		if !ok {
			t.Fatalf("expected message body to be *nodev1.RuntimeClass, got %T", msg.GetContent())
		}
		if rc.Name != expectedRC.Name {
			t.Errorf("expected RuntimeClass name %s, got %s", expectedRC.Name, rc.Name)
		}
	}
}
