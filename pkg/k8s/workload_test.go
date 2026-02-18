package k8s

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestResolveWorkloadDependencies(t *testing.T) {
	// Setup fake dynamic client with a Deployment
	deployObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "dep1",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "c1",
								"image": "nginx",
								"env": []interface{}{
									map[string]interface{}{
										"name": "MY_ENV",
										"valueFrom": map[string]interface{}{
											"configMapKeyRef": map[string]interface{}{
												"name": "cm1",
												"key":  "foo",
											},
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

	cmObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "cm1",
				"namespace": "default",
			},
		},
	}

	scheme := runtime.NewScheme()

	gvrMap := map[schema.GroupVersionResource]string{
		{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}:                         "HorizontalPodAutoscalerList",
		{Group: "", Version: "v1", Resource: "services"}:                                                    "ServiceList",
		{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}:                                  "IngressList",
		{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}:                            "NetworkPolicyList",
		{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"}:                                  "PodDisruptionBudgetList",
		{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}:               "CustomResourceDefinitionList",
		{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}: "ValidatingWebhookConfigurationList",
		{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}:   "MutatingWebhookConfigurationList",
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"}:                              "RoleList",
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}:                       "RoleBindingList",
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}:                       "ClusterRoleList",
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}:                "ClusterRoleBindingList",
		{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}:                                      "PersistentVolumeClaimList",
		{Group: "apps", Version: "v1", Resource: "deployments"}:                                             "DeploymentList",
		{Group: "", Version: "v1", Resource: "configmaps"}:                                                  "ConfigMapList",
		{Group: "", Version: "v1", Resource: "secrets"}:                                                     "SecretList",
		{Group: "", Version: "v1", Resource: "serviceaccounts"}:                                             "ServiceAccountList",
	}

	fakeDyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrMap, deployObj, cmObj)

	// Reactor to return empty lists for everything
	fakeDyn.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gvr := action.GetResource()
		kind, ok := gvrMap[gvr]
		if !ok {
			kind = "List" // Fallback
		}
		return true, &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": kind, "apiVersion": gvr.GroupVersion().String()},
			Items:  []unstructured.Unstructured{},
		}, nil
	})

	m, _ := NewMultiClusterClient("")
	m.rawConfig = &api.Config{Contexts: map[string]*api.Context{"c1": {Cluster: "cluster1"}}}
	m.dynamicClients["c1"] = fakeDyn

	kind, bundle, err := m.ResolveWorkloadDependencies(context.Background(), "c1", "default", "dep1")
	if err != nil {
		t.Fatalf("ResolveWorkloadDependencies failed: %v", err)
	}
	if kind != "Deployment" {
		t.Errorf("Expected Deployment, got %s", kind)
	}
	if bundle.Workload.GetName() != "dep1" {
		t.Errorf("Expected workload name dep1")
	}

	// cm1 should be in dependencies
	found := false
	for _, d := range bundle.Dependencies {
		if d.Kind == DepConfigMap && d.Name == "cm1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ConfigMap cm1 in dependencies")
	}

	// Test NotFound
	_, _, err = m.ResolveWorkloadDependencies(context.Background(), "c1", "default", "missing")
	if err == nil {
		t.Error("Expected error for missing workload")
	}
}

func TestListWorkloads(t *testing.T) {
	deployObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":              "dep1",
				"namespace":         "default",
				"creationTimestamp": time.Now().UTC().Format(time.RFC3339),
				"labels":            map[string]interface{}{"app": "nginx"},
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "c1",
								"image": "nginx",
							},
						},
					},
				},
			},
			"status": map[string]interface{}{
				"readyReplicas":     int64(3),
				"availableReplicas": int64(3),
			},
		},
	}

	stsObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "StatefulSet",
			"metadata": map[string]interface{}{
				"name":              "sts1",
				"namespace":         "default",
				"creationTimestamp": time.Now().UTC().Format(time.RFC3339),
				"labels":            map[string]interface{}{"app": "db"},
			},
			"spec": map[string]interface{}{
				"replicas": int64(2),
			},
			"status": map[string]interface{}{
				"readyReplicas": int64(2),
			},
		},
	}

	dsObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "DaemonSet",
			"metadata": map[string]interface{}{
				"name":              "ds1",
				"namespace":         "default",
				"creationTimestamp": time.Now().UTC().Format(time.RFC3339),
				"labels":            map[string]interface{}{"app": "monitor"},
			},
			"status": map[string]interface{}{
				"desiredNumberScheduled": int64(5),
				"numberReady":            int64(5),
			},
		},
	}

	scheme := runtime.NewScheme()
	gvrMap := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}:  "DeploymentList",
		{Group: "apps", Version: "v1", Resource: "statefulsets"}: "StatefulSetList",
		{Group: "apps", Version: "v1", Resource: "daemonsets"}:   "DaemonSetList",
	}

	// We need a reactor that returns the object for LIST operations
	fakeDyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrMap, deployObj, stsObj, dsObj)
	fakeDyn.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gvr := action.GetResource()
		if gvr.Resource == "deployments" {
			return true, &unstructured.UnstructuredList{
				Object: map[string]interface{}{"kind": "DeploymentList", "apiVersion": "apps/v1"},
				Items:  []unstructured.Unstructured{*deployObj},
			}, nil
		}
		if gvr.Resource == "statefulsets" {
			return true, &unstructured.UnstructuredList{
				Object: map[string]interface{}{"kind": "StatefulSetList", "apiVersion": "apps/v1"},
				Items:  []unstructured.Unstructured{*stsObj},
			}, nil
		}
		if gvr.Resource == "daemonsets" {
			return true, &unstructured.UnstructuredList{
				Object: map[string]interface{}{"kind": "DaemonSetList", "apiVersion": "apps/v1"},
				Items:  []unstructured.Unstructured{*dsObj},
			}, nil
		}
		return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{}}, nil
	})

	m, _ := NewMultiClusterClient("")
	m.rawConfig = &api.Config{Contexts: map[string]*api.Context{"c1": {Cluster: "cluster1"}}}
	m.dynamicClients["c1"] = fakeDyn
	// Fake typed client for iteration
	m.clients["c1"] = nil

	// Test List all
	wls, err := m.ListWorkloads(context.Background(), "", "default", "")
	if err != nil {
		t.Fatalf("ListWorkloads failed: %v", err)
	}
	if wls.TotalCount != 3 {
		t.Errorf("Expected 3 workloads, got %d", wls.TotalCount)
	}

	// Sort logic validation (order might vary, so we check for existence)
	foundSts := false
	foundDs := false
	for _, w := range wls.Items {
		if w.Name == "sts1" && w.Type == "StatefulSet" {
			foundSts = true
		}
		if w.Name == "ds1" && w.Type == "DaemonSet" {
			foundDs = true
		}
	}
	if !foundSts {
		t.Error("Expected sts1 StatefulSet")
	}
	if !foundDs {
		t.Error("Expected ds1 DaemonSet")
	}

	// Test GetWorkload
	wl, err := m.GetWorkload(context.Background(), "c1", "default", "sts1")
	if err != nil {
		t.Fatalf("GetWorkload failed: %v", err)
	}
	if wl == nil {
		t.Fatal("Expected sts1 workload, got nil")
	}
	if wl.Name != "sts1" {
		t.Errorf("Expected sts1, got %s", wl.Name)
	}

	// Test filtering by cluster
	wls, err = m.ListWorkloads(context.Background(), "c1", "default", "")
	if err != nil {
		t.Fatalf("ListWorkloads specific cluster failed: %v", err)
	}
	if wls.TotalCount != 3 {
		t.Errorf("Expected 3 workloads, got %d", wls.TotalCount)
	}
}

func TestDeployWorkload(t *testing.T) {
	deployObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "dep1",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(1),
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "c1",
								"image": "nginx",
							},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	gvrMap := map[schema.GroupVersionResource]string{
		{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}:                         "HorizontalPodAutoscalerList",
		{Group: "", Version: "v1", Resource: "services"}:                                                    "ServiceList",
		{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}:                                  "IngressList",
		{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}:                            "NetworkPolicyList",
		{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"}:                                  "PodDisruptionBudgetList",
		{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}:               "CustomResourceDefinitionList",
		{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}: "ValidatingWebhookConfigurationList",
		{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}:   "MutatingWebhookConfigurationList",
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"}:                              "RoleList",
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}:                       "RoleBindingList",
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}:                       "ClusterRoleList",
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}:                "ClusterRoleBindingList",
		{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}:                                      "PersistentVolumeClaimList",
		{Group: "apps", Version: "v1", Resource: "deployments"}:                                             "DeploymentList",
		{Group: "apps", Version: "v1", Resource: "statefulsets"}:                                            "StatefulSetList",
		{Group: "apps", Version: "v1", Resource: "daemonsets"}:                                              "DaemonSetList",
		{Group: "", Version: "v1", Resource: "configmaps"}:                                                  "ConfigMapList",
		{Group: "", Version: "v1", Resource: "secrets"}:                                                     "SecretList",
		{Group: "", Version: "v1", Resource: "serviceaccounts"}:                                             "ServiceAccountList",
		{Group: "", Version: "v1", Resource: "namespaces"}:                                                  "NamespaceList",
	}

	// Fake client for src and target
	fakeDyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrMap, deployObj)
	fakeDyn.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		// Return valid list for deployments, empty for others
		gvr := action.GetResource()
		if gvr.Resource == "deployments" {
			return true, &unstructured.UnstructuredList{
				Object: map[string]interface{}{"kind": "DeploymentList", "apiVersion": "apps/v1"},
				Items:  []unstructured.Unstructured{*deployObj},
			}, nil
		}
		return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{}}, nil
	})
	// Allow Create of Namespace and Deployment on target
	fakeDyn.PrependReactor("create", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		return true, createAction.GetObject(), nil
	})

	m, _ := NewMultiClusterClient("")
	m.rawConfig = &api.Config{Contexts: map[string]*api.Context{
		"src": {Cluster: "source"},
		"tgt": {Cluster: "target"},
	}}
	m.dynamicClients["src"] = fakeDyn
	m.dynamicClients["tgt"] = fakeDyn

	opts := &DeployOptions{DeployedBy: "me"}
	resp, err := m.DeployWorkload(context.Background(), "src", "default", "dep1", []string{"tgt"}, 2, opts)
	if err != nil {
		t.Fatalf("DeployWorkload failed: %v", err)
	}
	if !resp.Success {
		t.Errorf("Expected success, fail msg: %v", resp.Message)
	}
	if len(resp.DeployedTo) != 1 || resp.DeployedTo[0] != "tgt" {
		t.Errorf("Expected deployed to tgt, got %v", resp.DeployedTo)
	}
}
