package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/kubestellar/console/pkg/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestListGateways(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewGatewayHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/gateway/gateways", handler.ListGateways)

	// Setup Scheme
	scheme := runtime.NewScheme()

	// Seed data
	gw := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      "my-gateway",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"gatewayClassName": "test-class",
			},
		},
	}

	gvr := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	gvrBeta := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gateways"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		gvr:     "GatewayList",
		gvrBeta: "GatewayList",
	})
	env.K8sClient.InjectDynamicClient("test-cluster", dynClient)
	env.K8sClient.InjectClient("test-cluster", k8sfake.NewSimpleClientset())

	// Inject success reactor
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": "GatewayList", "apiVersion": "gateway.networking.k8s.io/v1"},
			Items:  []unstructured.Unstructured{*gw},
		}, nil
	})

	// Case 1: List all (success)
	req, _ := http.NewRequest("GET", "/api/gateway/gateways", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var list v1alpha1.GatewayList
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &list)
	require.NoError(t, err)
	require.NotEmpty(t, list.Items)
	assert.Equal(t, "my-gateway", list.Items[0].Name)

	// Case 2: List specific cluster (success)
	req2, _ := http.NewRequest("GET", "/api/gateway/gateways?cluster=test-cluster", nil)
	resp2, err := env.App.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)

	// Case 3: List specific cluster (failure)
	// Inject failure at top of chain
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("simulated error")
	})

	req3, _ := http.NewRequest("GET", "/api/gateway/gateways?cluster=test-cluster", nil)
	resp3, err := env.App.Test(req3)
	require.NoError(t, err)
	assert.Equal(t, 200, resp3.StatusCode)
}

func TestGetGateway(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewGatewayHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/gateway/gateways/:cluster/:namespace/:name", handler.GetGateway)

	scheme := runtime.NewScheme()
	gw := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      "target-gw",
				"namespace": "default",
			},
		},
	}

	gvr := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	gvrBeta := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gateways"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		gvr:     "GatewayList",
		gvrBeta: "GatewayList",
	})
	env.K8sClient.InjectDynamicClient("c1", dynClient)
	env.K8sClient.InjectClient("c1", k8sfake.NewSimpleClientset())

	// Success reactor
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": "GatewayList", "apiVersion": "gateway.networking.k8s.io/v1"},
			Items:  []unstructured.Unstructured{*gw},
		}, nil
	})

	// Case 1: Found
	req, _ := http.NewRequest("GET", "/api/gateway/gateways/c1/default/target-gw", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Case 2: Not Found (simulate by returning empty list for different call or just check logic)
	// Currently reactor returns list containing "target-gw".
	// If we ask for "missing", logic filters it out.
	req2, _ := http.NewRequest("GET", "/api/gateway/gateways/c1/default/missing", nil)
	resp2, err := env.App.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 404, resp2.StatusCode)

	// Case 3: Client Error
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("list failure")
	})
	req3, _ := http.NewRequest("GET", "/api/gateway/gateways/c1/default/target-gw", nil)
	resp3, err := env.App.Test(req3)
	require.NoError(t, err)
	assert.Equal(t, 404, resp3.StatusCode)
}

func TestListHTTPRoutes(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewGatewayHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/gateway/httproutes", handler.ListHTTPRoutes)

	scheme := runtime.NewScheme()
	route := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "HTTPRoute",
			"metadata": map[string]interface{}{
				"name":      "my-route",
				"namespace": "default",
			},
		},
	}

	gvr := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
	gvrBeta := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "httproutes"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		gvr:     "HTTPRouteList",
		gvrBeta: "HTTPRouteList",
	})
	env.K8sClient.InjectDynamicClient("test-cluster", dynClient)
	env.K8sClient.InjectClient("test-cluster", k8sfake.NewSimpleClientset())

	// Success reactor
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": "HTTPRouteList", "apiVersion": "gateway.networking.k8s.io/v1"},
			Items:  []unstructured.Unstructured{*route},
		}, nil
	})

	// Case 1: List all
	req, _ := http.NewRequest("GET", "/api/gateway/httproutes", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var list v1alpha1.HTTPRouteList
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &list)
	require.NoError(t, err)
	assert.NotEmpty(t, list.Items)
	assert.Equal(t, "my-route", list.Items[0].Name)

	// Case 2: Specific cluster failure
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("route error")
	})
	req2, _ := http.NewRequest("GET", "/api/gateway/httproutes?cluster=test-cluster", nil)
	resp2, err := env.App.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)
}

func TestGetHTTPRoute(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewGatewayHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/gateway/httproutes/:cluster/:namespace/:name", handler.GetHTTPRoute)

	scheme := runtime.NewScheme()
	route := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "HTTPRoute",
			"metadata": map[string]interface{}{
				"name":      "target-route",
				"namespace": "default",
			},
		},
	}

	gvr := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
	gvrBeta := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "httproutes"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		gvr:     "HTTPRouteList",
		gvrBeta: "HTTPRouteList",
	})
	env.K8sClient.InjectDynamicClient("c1", dynClient)
	env.K8sClient.InjectClient("c1", k8sfake.NewSimpleClientset())

	// Success reactor
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": "HTTPRouteList", "apiVersion": "gateway.networking.k8s.io/v1"},
			Items:  []unstructured.Unstructured{*route},
		}, nil
	})

	// Case 1: Found
	req, _ := http.NewRequest("GET", "/api/gateway/httproutes/c1/default/target-route", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Case 2: 404
	req2, _ := http.NewRequest("GET", "/api/gateway/httproutes/c1/default/missing", nil)
	resp2, err := env.App.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 404, resp2.StatusCode)
}

func TestGetGatewayAPIStatus(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewGatewayHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/gateway/status", handler.GetGatewayAPIStatus)

	// The default setup has "test-cluster".
	// By default, fake dynamic client returns empty list (no error) for arbitrary resources,
	// unless CRD check logic in k8s client is specific.
	// k8s.IsGatewayAPIAvailable does a List(Limit=1).
	// Default fake reactor returns empty list, err=nil. So it should be "Available".

	gvr := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	gvrBeta := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gateways"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		gvr:     "GatewayList",
		gvrBeta: "GatewayList",
	})
	env.K8sClient.InjectDynamicClient("test-cluster", dynClient)
	env.K8sClient.InjectClient("test-cluster", k8sfake.NewSimpleClientset())

	req, _ := http.NewRequest("GET", "/api/gateway/status", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var res map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &res)
	clusters := res["clusters"].([]interface{})
	assert.NotEmpty(t, clusters)
}
