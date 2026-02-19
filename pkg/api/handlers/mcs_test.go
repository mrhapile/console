package handlers

import (
	"bytes"
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

func TestListServiceExports(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewMCSHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/mcs/exports", handler.ListServiceExports)

	scheme := runtime.NewScheme()
	export := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "multicluster.x-k8s.io/v1alpha1",
			"kind":       "ServiceExport",
			"metadata": map[string]interface{}{
				"name":      "my-svc",
				"namespace": "default",
			},
		},
	}

	gvr := schema.GroupVersionResource{Group: "multicluster.x-k8s.io", Version: "v1alpha1", Resource: "serviceexports"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		gvr: "ServiceExportList",
	})
	env.K8sClient.InjectDynamicClient("test-cluster", dynClient)
	env.K8sClient.InjectClient("test-cluster", k8sfake.NewSimpleClientset())

	// Success reactor
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": "ServiceExportList", "apiVersion": "multicluster.x-k8s.io/v1alpha1"},
			Items:  []unstructured.Unstructured{*export},
		}, nil
	})

	// Case 1: List all
	req, _ := http.NewRequest("GET", "/api/mcs/exports", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var list v1alpha1.ServiceExportList
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &list)
	require.NoError(t, err)
	require.NotEmpty(t, list.Items)
	assert.Equal(t, "my-svc", list.Items[0].Name)

	// Case 2: Specific cluster failure
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("export list error")
	})
	req2, _ := http.NewRequest("GET", "/api/mcs/exports?cluster=test-cluster", nil)
	resp2, err := env.App.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)
}

func TestGetServiceExport(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewMCSHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/mcs/exports/:cluster/:namespace/:name", handler.GetServiceExport)

	scheme := runtime.NewScheme()
	export := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "multicluster.x-k8s.io/v1alpha1",
			"kind":       "ServiceExport",
			"metadata": map[string]interface{}{
				"name":      "target-svc",
				"namespace": "default",
			},
		},
	}

	gvr := schema.GroupVersionResource{Group: "multicluster.x-k8s.io", Version: "v1alpha1", Resource: "serviceexports"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		gvr: "ServiceExportList",
	})
	env.K8sClient.InjectDynamicClient("c1", dynClient)
	env.K8sClient.InjectClient("c1", k8sfake.NewSimpleClientset())

	// Success reactor
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": "ServiceExportList", "apiVersion": "multicluster.x-k8s.io/v1alpha1"},
			Items:  []unstructured.Unstructured{*export},
		}, nil
	})

	// Found
	req, _ := http.NewRequest("GET", "/api/mcs/exports/c1/default/target-svc", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Client Error
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("fail")
	})
	req2, _ := http.NewRequest("GET", "/api/mcs/exports/c1/default/target-svc", nil)
	resp2, err := env.App.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 404, resp2.StatusCode)
}

func TestListServiceImports(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewMCSHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/mcs/imports", handler.ListServiceImports)

	scheme := runtime.NewScheme()
	imp := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "multicluster.x-k8s.io/v1alpha1",
			"kind":       "ServiceImport",
			"metadata": map[string]interface{}{
				"name":      "remote-svc",
				"namespace": "default",
			},
		},
	}

	gvr := schema.GroupVersionResource{Group: "multicluster.x-k8s.io", Version: "v1alpha1", Resource: "serviceimports"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		gvr: "ServiceImportList",
	})
	env.K8sClient.InjectDynamicClient("test-cluster", dynClient)
	env.K8sClient.InjectClient("test-cluster", k8sfake.NewSimpleClientset())

	// Success reactor
	dynClient.PrependReactor("list", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": "ServiceImportList", "apiVersion": "multicluster.x-k8s.io/v1alpha1"},
			Items:  []unstructured.Unstructured{*imp},
		}, nil
	})

	// List all
	req, _ := http.NewRequest("GET", "/api/mcs/imports", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var list v1alpha1.ServiceImportList
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &list)
	require.NoError(t, err)
	assert.NotEmpty(t, list.Items)
}

func TestCreateServiceExport(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewMCSHandlers(env.K8sClient, env.Hub)
	env.App.Post("/api/mcs/exports", handler.CreateServiceExport)

	// Case 1: Success
	gvr := schema.GroupVersionResource{Group: "multicluster.x-k8s.io", Version: "v1alpha1", Resource: "serviceexports"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		gvr: "ServiceExportList",
	})
	env.K8sClient.InjectDynamicClient("c1", dynClient)
	env.K8sClient.InjectClient("c1", k8sfake.NewSimpleClientset())

	// Success reactor
	dynClient.PrependReactor("create", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.Unstructured{}, nil
	})

	payload := map[string]string{
		"cluster":     "c1",
		"namespace":   "default",
		"serviceName": "my-svc",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/mcs/exports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// Verify creation via actions
	found := false
	for _, action := range dynClient.Actions() {
		if action.GetVerb() == "create" && action.GetResource().Resource == "serviceexports" {
			found = true
			break
		}
	}
	assert.True(t, found, "Create action not found")

	// Case 2: Validation Error
	payloadInvalid := map[string]string{
		"cluster":   "c1",
		"namespace": "default",
		// missing serviceName
	}
	bodyInvalid, _ := json.Marshal(payloadInvalid)
	reqInvalid, _ := http.NewRequest("POST", "/api/mcs/exports", bytes.NewReader(bodyInvalid))
	reqInvalid.Header.Set("Content-Type", "application/json")

	respInvalid, err := env.App.Test(reqInvalid)
	require.NoError(t, err)
	assert.Equal(t, 400, respInvalid.StatusCode)

	// Case 3: Client Error
	dynClient.PrependReactor("create", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("create failed")
	})
	reqFail, _ := http.NewRequest("POST", "/api/mcs/exports", bytes.NewReader(body))
	reqFail.Header.Set("Content-Type", "application/json")

	respFail, err := env.App.Test(reqFail)
	require.NoError(t, err)
	assert.Equal(t, 500, respFail.StatusCode)
}

func TestDeleteServiceExport(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewMCSHandlers(env.K8sClient, env.Hub)
	env.App.Delete("/api/mcs/exports/:cluster/:namespace/:name", handler.DeleteServiceExport)

	gvr := schema.GroupVersionResource{Group: "multicluster.x-k8s.io", Version: "v1alpha1", Resource: "serviceexports"}
	dynClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		gvr: "ServiceExportList",
	})
	env.K8sClient.InjectDynamicClient("c1", dynClient)
	env.K8sClient.InjectClient("c1", k8sfake.NewSimpleClientset())

	// Success reactor
	dynClient.PrependReactor("delete", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})

	// Case 1: Success
	req, _ := http.NewRequest("DELETE", "/api/mcs/exports/c1/default/svc1", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Case 2: Client Error
	dynClient.PrependReactor("delete", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("delete failed")
	})
	reqFail, _ := http.NewRequest("DELETE", "/api/mcs/exports/c1/default/svc1", nil)
	respFail, err := env.App.Test(reqFail)
	require.NoError(t, err)
	assert.Equal(t, 500, respFail.StatusCode)
}
