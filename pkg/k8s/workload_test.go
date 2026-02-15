package k8s

import (
	"context"
	"testing"

	"github.com/kubestellar/console/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestListWorkloadsForCluster(t *testing.T) {
	m, _ := NewMultiClusterClient("")

	myScheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(myScheme)
	_ = apiextensionsv1.AddToScheme(myScheme)

	// Create one of each: Deployment, StatefulSet, DaemonSet
	dep := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":              "my-dep",
				"namespace":         "default",
				"creationTimestamp": "2024-01-01T00:00:00Z",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
			},
			"status": map[string]interface{}{
				"readyReplicas":     int64(3),
				"availableReplicas": int64(3),
			},
		},
	}
	sts := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "StatefulSet",
			"metadata": map[string]interface{}{
				"name":      "my-sts",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(2),
			},
			"status": map[string]interface{}{
				"readyReplicas": int64(1), // Degraded
			},
		},
	}
	ds := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "DaemonSet",
			"metadata": map[string]interface{}{
				"name":      "my-ds",
				"namespace": "kube-system",
			},
			"status": map[string]interface{}{
				"desiredNumberScheduled": int64(5),
				"numberReady":            int64(5),
			},
		},
	}

	fakeDyn := fake.NewSimpleDynamicClient(myScheme, dep, sts, ds)
	m.SetDynamicClient("c1", fakeDyn)

	// Test listing ALL
	workloads, err := m.ListWorkloadsForCluster(context.Background(), "c1", "", "")
	if err != nil {
		t.Fatalf("ListWorkloadsForCluster failed: %v", err)
	}

	if len(workloads) != 3 {
		t.Errorf("Expected 3 workloads, got %d", len(workloads))
	}

	// Verify Deployment
	foundDep := false
	for _, w := range workloads {
		if w.Name == "my-dep" {
			foundDep = true
			if w.Type != v1alpha1.WorkloadTypeDeployment {
				t.Errorf("Expected Deployment type, got %s", w.Type)
			}
			if w.Status != v1alpha1.WorkloadStatusRunning {
				t.Errorf("Expected Running status for dep, got %s", w.Status)
			}
		} else if w.Name == "my-sts" {
			if w.Status != v1alpha1.WorkloadStatusDegraded {
				t.Errorf("Expected Degraded status for sts, got %s", w.Status)
			}
		}
	}
	if !foundDep {
		t.Error("Deployment my-dep not found")
	}

	// Test listing by Namespace
	workloads, err = m.ListWorkloadsForCluster(context.Background(), "c1", "default", "")
	if err != nil {
		t.Fatalf("ListWorkloadsForCluster (ns) failed: %v", err)
	}
	if len(workloads) != 2 { // dep, sts
		t.Errorf("Expected 2 workloads in default, got %d", len(workloads))
	}

	// Test listing by Type
	workloads, err = m.ListWorkloadsForCluster(context.Background(), "c1", "", "DaemonSet")
	if err != nil {
		t.Fatalf("ListWorkloadsForCluster (type) failed: %v", err)
	}
	if len(workloads) != 1 {
		t.Errorf("Expected 1 DaemonSet, got %d", len(workloads))
	}
	if workloads[0].Name != "my-ds" {
		t.Errorf("Expected my-ds, got %s", workloads[0].Name)
	}

	// Test GetWorkload
	w, err := m.GetWorkload(context.Background(), "c1", "default", "my-dep")
	if err != nil {
		t.Fatalf("GetWorkload failed: %v", err)
	}
	if w == nil {
		t.Error("GetWorkload returned nil")
	} else if w.Name != "my-dep" {
		t.Errorf("Expected my-dep, got %s", w.Name)
	}
}

func TestListWorkloads(t *testing.T) {
	m, _ := NewMultiClusterClient("")
	myScheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(myScheme)
	_ = apiextensionsv1.AddToScheme(myScheme)

	// Cluster 1 has deployment
	dep1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata":   map[string]interface{}{"name": "dep1", "namespace": "default"},
		},
	}
	m.SetDynamicClient("c1", fake.NewSimpleDynamicClient(myScheme, dep1))

	// Cluster 2 has deployment
	dep2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata":   map[string]interface{}{"name": "dep2", "namespace": "default"},
		},
	}
	m.SetDynamicClient("c2", fake.NewSimpleDynamicClient(myScheme, dep2))

	// Setup RawConfig to simulate clusters
	config := &api.Config{
		Contexts: map[string]*api.Context{
			"c1": {Cluster: "c1"},
			"c2": {Cluster: "c2"},
		},
		Clusters: map[string]*api.Cluster{
			"c1": {Server: "https://c1"},
			"c2": {Server: "https://c2"},
		},
	}
	m.SetRawConfig(config)

	workloads, err := m.ListWorkloads(context.Background(), "", "default", "")
	if err != nil {
		t.Fatalf("ListWorkloads failed: %v", err)
	}

	if len(workloads.Items) != 2 {
		t.Errorf("Expected 2 workloads, got %d", len(workloads.Items))
	}
}

func TestDeployWorkload(t *testing.T) {
	// 1. Setup Client
	m, _ := NewMultiClusterClient("")

	myScheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(myScheme)
	_ = apiextensionsv1.AddToScheme(myScheme)

	// 2. Setup Source Cluster (cluster-1) with Deployment and dependency
	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "my-dep",
				"namespace": "default",
				"labels": map[string]interface{}{
					"app": "my-app",
				},
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "my-app",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "my-app",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "nginx",
								"image": "nginx:latest",
								"envFrom": []interface{}{
									map[string]interface{}{
										"configMapRef": map[string]interface{}{
											"name": "my-config",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	cm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "my-config",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"foo": "bar",
			},
		},
	}

	sourceClient := fake.NewSimpleDynamicClient(myScheme, deployment, cm)
	m.SetDynamicClient("cluster-1", sourceClient)

	// 3. Setup Target Cluster (cluster-2) empty
	targetClient := fake.NewSimpleDynamicClient(myScheme)
	m.SetDynamicClient("cluster-2", targetClient)

	// 4. Test DeployWorkload
	ctx := context.Background()
	opts := &DeployOptions{DeployedBy: "test-user"}

	resp, err := m.DeployWorkload(ctx, "cluster-1", "default", "my-dep", []string{"cluster-2"}, 5, opts)
	if err != nil {
		t.Fatalf("DeployWorkload failed: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success, got failure. Failed clusters: %v", resp.FailedClusters)
	}

	if len(resp.DeployedTo) != 1 || resp.DeployedTo[0] != "cluster-2" {
		t.Errorf("Expected deployed to [cluster-2], got %v", resp.DeployedTo)
	}

	// 5. Verify Deployment on Target
	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	targetDep, err := targetClient.Resource(gvr).Namespace("default").Get(ctx, "my-dep", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get deployment from target: %v", err)
	}

	// Verify replicas override
	replicas, _, _ := unstructured.NestedInt64(targetDep.Object, "spec", "replicas")
	if replicas != 5 {
		t.Errorf("Expected replicas 5, got %d", replicas)
	}

	// Verify labels
	labels := targetDep.GetLabels()
	if labels["kubestellar.io/deployed-by"] != "test-user" {
		t.Errorf("Expected deployed-by label, got %v", labels)
	}
}

func TestResolveWorkloadDependencies(t *testing.T) {
	// Setup similar to above but test ResolveWorkloadDependencies wrapper
	m, _ := NewMultiClusterClient("")
	myScheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(myScheme)
	_ = apiextensionsv1.AddToScheme(myScheme)

	// Create Deployment
	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "my-dep",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{},
					},
				},
			},
		},
	}

	sourceClient := fake.NewSimpleDynamicClient(myScheme, deployment)
	m.SetDynamicClient("cluster-1", sourceClient)

	kind, bundle, err := m.ResolveWorkloadDependencies(context.Background(), "cluster-1", "default", "my-dep")
	if err != nil {
		t.Fatalf("ResolveWorkloadDependencies failed: %v", err)
	}

	if kind != "Deployment" {
		t.Errorf("Expected kind Deployment, got %s", kind)
	}

	if bundle == nil {
		t.Error("Expected bundle, got nil")
	}
}

func TestGetClusterCapabilities(t *testing.T) {
	m, _ := NewMultiClusterClient("")

	myScheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(myScheme)

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
		},
	}

	fakeCS := fakek8s.NewSimpleClientset(node)
	m.SetClient("c1", fakeCS)

	// We need to ensure ListClusters returns c1.
	config := &api.Config{
		Contexts: map[string]*api.Context{
			"c1": {Cluster: "c1"},
		},
		Clusters: map[string]*api.Cluster{
			"c1": {Server: "https://c1"},
		},
	}
	m.SetRawConfig(config)

	// Get capabilities
	caps, err := m.GetClusterCapabilities(context.Background())
	if err != nil {
		t.Fatalf("GetClusterCapabilities failed: %v", err)
	}

	if caps.TotalCount < 1 {
		t.Error("Expected at least 1 cluster capability")
	}
}

func TestNodeLabels(t *testing.T) {
	m, _ := NewMultiClusterClient("")
	myScheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(myScheme)

	node := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Node",
			"metadata": map[string]interface{}{
				"name": "node1",
				"labels": map[string]interface{}{
					"existing": "val",
					"toremove": "val",
				},
			},
		},
	}

	fakeDyn := fake.NewSimpleDynamicClient(myScheme, node)
	m.SetDynamicClient("c1", fakeDyn)

	// Test LabelClusterNodes
	err := m.LabelClusterNodes(context.Background(), "c1", map[string]string{"new": "label"})
	if err != nil {
		t.Fatalf("LabelClusterNodes failed: %v", err)
	}

	gvrNodes := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
	updatedNode, err := fakeDyn.Resource(gvrNodes).Get(context.Background(), "node1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	labels := updatedNode.GetLabels()
	if labels["new"] != "label" {
		t.Error("Label not added")
	}

	// Test RemoveClusterNodeLabels
	err = m.RemoveClusterNodeLabels(context.Background(), "c1", []string{"toremove", "nonexistent"})
	if err != nil {
		t.Fatalf("RemoveClusterNodeLabels failed: %v", err)
	}

	updatedNode, _ = fakeDyn.Resource(gvrNodes).Get(context.Background(), "node1", metav1.GetOptions{})
	labels = updatedNode.GetLabels()
	if _, ok := labels["toremove"]; ok {
		t.Error("Label not removed")
	}
	if labels["existing"] != "val" {
		t.Error("Existing label affected")
	}
}

func TestWorkloadOperations(t *testing.T) {
	m, _ := NewMultiClusterClient("")

	// Test Scale (Placeholder)
	resp, err := m.ScaleWorkload(context.Background(), "ns", "name", []string{"c1"}, 3)
	if err != nil {
		t.Fatalf("ScaleWorkload failed: %v", err)
	}
	if !resp.Success {
		t.Error("ScaleWorkload failed")
	}

	// Test Delete (Placeholder)
	err = m.DeleteWorkload(context.Background(), "c1", "ns", "name")
	if err != nil {
		t.Fatalf("DeleteWorkload failed: %v", err)
	}

	// Test ListBindingPolicies (Placeholder)
	bp, err := m.ListBindingPolicies(context.Background())
	if err != nil {
		t.Fatalf("ListBindingPolicies failed: %v", err)
	}
	if len(bp.Items) != 0 {
		t.Error("Expected empty binding policies")
	}
}
