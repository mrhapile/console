package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestScaleWorkload(t *testing.T) {
	deployObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "dep1",
				"namespace": "default",
			},
		},
	}

	scheme := runtime.NewScheme()
	gvrMap := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
	}

	fakeDyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrMap, deployObj)

	m, _ := NewMultiClusterClient("")
	m.dynamicClients["c1"] = fakeDyn
	m.rawConfig = &api.Config{Contexts: map[string]*api.Context{"c1": {Cluster: "cluster1"}}}

	resp, err := m.ScaleWorkload(context.Background(), "default", "dep1", []string{"c1"}, 5)
	if err != nil {
		t.Fatalf("ScaleWorkload failed: %v", err)
	}
	if !resp.Success {
		t.Error("Expected success")
	}
}

func TestDeleteWorkload(t *testing.T) {
	scheme := runtime.NewScheme()
	gvrMap := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
	}

	fakeDyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrMap)

	m, _ := NewMultiClusterClient("")
	m.dynamicClients["c1"] = fakeDyn
	m.rawConfig = &api.Config{Contexts: map[string]*api.Context{"c1": {Cluster: "cluster1"}}}

	err := m.DeleteWorkload(context.Background(), "c1", "default", "dep1")
	if err != nil {
		t.Errorf("DeleteWorkload failed: %v", err)
	}
}

func TestGetClusterCapabilities(t *testing.T) {
	m, _ := NewMultiClusterClient("")

	// Setup fake client for cluster c1
	fakeClient := k8sfake.NewSimpleClientset()
	m.clients = map[string]kubernetes.Interface{
		"c1": fakeClient,
	}

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
			Labels: map[string]string{
				"gpu-type": "nvidia-a100",
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				"nvidia.com/gpu": resource.MustParse("1"),
			},
		},
	}

	fakeClient.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})

	m.rawConfig = &api.Config{Contexts: map[string]*api.Context{"c1": {Cluster: "cl1"}}}

	caps, err := m.GetClusterCapabilities(context.Background())
	if err != nil {
		t.Fatalf("GetClusterCapabilities failed: %v", err)
	}
	if caps == nil {
		t.Fatal("Expected capabilities")
	}
	if len(caps.Items) != 1 {
		t.Errorf("Expected 1 capability item, got %d", len(caps.Items))
	}
	if len(caps.Items) > 0 && caps.Items[0].NodeCount != 1 {
		t.Errorf("Expected 1 node, got %d", caps.Items[0].NodeCount)
	}
}

func TestLabelClusterNodes(t *testing.T) {
	scheme := runtime.NewScheme()
	gvrMap := map[schema.GroupVersionResource]string{
		{Group: "", Version: "v1", Resource: "nodes"}: "NodeList",
	}

	nodeObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Node",
			"metadata": map[string]interface{}{
				"name": "node1",
			},
		},
	}

	fakeDyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrMap, nodeObj)

	m, _ := NewMultiClusterClient("")
	m.dynamicClients["c1"] = fakeDyn
	m.rawConfig = &api.Config{Contexts: map[string]*api.Context{"c1": {Cluster: "cluster1"}}}

	err := m.LabelClusterNodes(context.Background(), "c1", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("LabelClusterNodes failed: %v", err)
	}
}

func TestRemoveClusterNodeLabels(t *testing.T) {
	scheme := runtime.NewScheme()
	gvrMap := map[schema.GroupVersionResource]string{
		{Group: "", Version: "v1", Resource: "nodes"}: "NodeList",
	}

	nodeObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Node",
			"metadata": map[string]interface{}{
				"name": "node1",
				"labels": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
	}

	fakeDyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrMap, nodeObj)

	m, _ := NewMultiClusterClient("")
	m.dynamicClients["c1"] = fakeDyn
	m.rawConfig = &api.Config{Contexts: map[string]*api.Context{"c1": {Cluster: "cluster1"}}}

	err := m.RemoveClusterNodeLabels(context.Background(), "c1", []string{"foo"})
	if err != nil {
		t.Fatalf("RemoveClusterNodeLabels failed: %v", err)
	}

	// Verify label removed
	updatedNode, err := fakeDyn.Resource(schema.GroupVersionResource{Version: "v1", Resource: "nodes"}).Get(context.Background(), "node1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	labels, _, _ := unstructured.NestedStringMap(updatedNode.Object, "metadata", "labels")
	if _, ok := labels["foo"]; ok {
		t.Error("Label foo should be removed")
	}
}

func TestListBindingPolicies(t *testing.T) {
	m, _ := NewMultiClusterClient("")
	bp, err := m.ListBindingPolicies(context.Background())
	if err != nil {
		t.Fatalf("ListBindingPolicies failed: %v", err)
	}
	if bp == nil {
		t.Fatal("Expected binding policies")
	}
	if len(bp.Items) != 0 {
		t.Error("Expected empty items")
	}
}
