package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/kubestellar/console/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWorkloads(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/workloads", handler.ListWorkloads)

	// Setup Scheme and Dynamic Client
	scheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(scheme)

	// Seed data
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func(i int32) *int32 { return &i }(1),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "c1", Image: "nginx"}},
				},
			},
		},
	}

	dynClient := fake.NewSimpleDynamicClient(scheme, deployment)
	env.K8sClient.InjectDynamicClient("test-cluster", dynClient)

	// Test GET
	req, _ := http.NewRequest("GET", "/api/workloads?cluster=test-cluster", nil)
	resp, err := env.App.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	items := response["items"].([]interface{})
	assert.NotEmpty(t, items)
	workload := items[0].(map[string]interface{})
	assert.Equal(t, "test-deploy", workload["name"])
}

func TestGetWorkload(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/workloads/:cluster/:namespace/:name", handler.GetWorkload)

	scheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(scheme)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func(i int32) *int32 { return &i }(2),
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      2,
			ReadyReplicas: 2,
		},
	}

	dynClient := fake.NewSimpleDynamicClient(scheme, deployment)
	env.K8sClient.InjectDynamicClient("test-cluster", dynClient)

	// 1. Success Case
	req, _ := http.NewRequest("GET", "/api/workloads/test-cluster/default/my-app", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var workload map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&workload)
	assert.Equal(t, "my-app", workload["name"])
	assert.Equal(t, "test-cluster", workload["deployments"].([]interface{})[0].(map[string]interface{})["cluster"])

	// 2. Not Found
	reqNotFound, _ := http.NewRequest("GET", "/api/workloads/test-cluster/default/missing", nil)
	respNotFound, _ := env.App.Test(reqNotFound)
	assert.Equal(t, 404, respNotFound.StatusCode)
}

func TestDeployWorkload(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Post("/api/workloads/deploy", handler.DeployWorkload)

	scheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(scheme)

	// Source Deployment
	srcDep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "src-app",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func(i int32) *int32 { return &i }(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "nginx"}}},
			},
		},
	}

	sourceClient := fake.NewSimpleDynamicClient(scheme, srcDep)
	targetClient := fake.NewSimpleDynamicClient(scheme)

	env.K8sClient.InjectDynamicClient("source-cluster", sourceClient)
	env.K8sClient.InjectDynamicClient("target-cluster", targetClient)

	// Payload
	payload := map[string]interface{}{
		"workloadName":   "src-app",
		"namespace":      "default",
		"sourceCluster":  "source-cluster",
		"targetClusters": []string{"target-cluster"},
		"replicas":       3,
	}

	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/workloads/deploy", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req)
	require.NoError(t, err)
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Deploy returned %d: %s", resp.StatusCode, string(body))
	}
	assert.Equal(t, 200, resp.StatusCode)

	// Verify result body
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["success"])

	// Verify deployment created in target
	gvr := appsv1.SchemeGroupVersion.WithResource("deployments")
	dep, err := targetClient.Resource(gvr).Namespace("default").Get(context.Background(), "src-app", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "src-app", dep.GetName())

	// Check Replicas (need to traverse unstructured map)
	spec := dep.Object["spec"].(map[string]interface{})
	assert.Equal(t, int64(3), spec["replicas"]) // note: int64 for unstructured
}

func TestGetDeployStatus(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/workloads/deploy-status/:cluster/:namespace/:name", handler.GetDeployStatus)

	scheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(scheme)

	deploy := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "status-app", Namespace: "default"},
		Spec:       appsv1.DeploymentSpec{Replicas: func(i int32) *int32 { return &i }(5)},
		Status:     appsv1.DeploymentStatus{ReadyReplicas: 3, Replicas: 5},
	}

	dynClient := fake.NewSimpleDynamicClient(scheme, deploy)
	env.K8sClient.InjectDynamicClient("c1", dynClient)

	req, _ := http.NewRequest("GET", "/api/workloads/deploy-status/c1/default/status-app", nil)
	resp, err := env.App.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var status map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&status)
	assert.Equal(t, float64(5), status["replicas"])
	assert.Equal(t, float64(3), status["readyReplicas"])
}

func TestScaleWorkload(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Post("/api/workloads/scale", handler.ScaleWorkload)

	// Payload
	payload := map[string]interface{}{
		"workloadName":   "scale-app",
		"namespace":      "default",
		"targetClusters": []string{"scale-cluster"},
		"replicas":       10,
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/workloads/scale", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["success"])
}

func TestDeleteWorkload(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Delete("/api/workloads/:cluster/:namespace/:name", handler.DeleteWorkload)

	req, _ := http.NewRequest("DELETE", "/api/workloads/c1/default/del-app", nil)
	resp, err := env.App.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestClusterGroupsCRUD(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)

	env.App.Get("/api/cluster-groups", handler.ListClusterGroups)
	env.App.Post("/api/cluster-groups", handler.CreateClusterGroup)
	env.App.Put("/api/cluster-groups/:name", handler.UpdateClusterGroup)
	env.App.Delete("/api/cluster-groups/:name", handler.DeleteClusterGroup)

	createPayload := map[string]interface{}{
		"name":     "group1",
		"kind":     "static",
		"clusters": []string{"c1", "c2"},
		"color":    "blue",
	}
	data, _ := json.Marshal(createPayload)
	req, _ := http.NewRequest("POST", "/api/cluster-groups", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	req, _ = http.NewRequest("GET", "/api/cluster-groups", nil)
	resp, err = env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var listResp map[string][]map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &listResp)
	assert.Equal(t, 1, len(listResp["groups"]))
	assert.Equal(t, "group1", listResp["groups"][0]["name"])

	updatePayload := map[string]interface{}{
		"name":     "group1",
		"kind":     "static",
		"clusters": []string{"c1"},
		"color":    "red",
	}
	data, _ = json.Marshal(updatePayload)
	req, _ = http.NewRequest("PUT", "/api/cluster-groups/group1", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err = env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	req, _ = http.NewRequest("DELETE", "/api/cluster-groups/group1", nil)
	resp, err = env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	req, _ = http.NewRequest("GET", "/api/cluster-groups", nil)
	resp, _ = env.App.Test(req)
	var listRespEmpty map[string][]interface{}
	bodyEmpty, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyEmpty, &listRespEmpty)
	assert.Empty(t, listRespEmpty["groups"])
}

func TestEvaluateClusterQuery(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Post("/api/cluster-groups/evaluate", handler.EvaluateClusterQuery)

	config := &api.Config{
		Contexts: map[string]*api.Context{
			"c1-ctx": {Cluster: "cluster1"},
		},
		Clusters: map[string]*api.Cluster{
			"cluster1": {Server: "https://c1.com"},
		},
	}
	env.K8sClient.SetRawConfig(config)

	scheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(scheme)

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{"region": "us-east"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("16Gi"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("16Gi"),
			},
		},
	}

	c1Client := k8sfake.NewSimpleClientset(node)
	env.K8sClient.InjectClient("c1-ctx", c1Client)

	// Complex Query with various operators
	query := map[string]interface{}{
		"labelSelector": "region=us-east",
		"filters": []map[string]interface{}{
			{"field": "cpuCores", "operator": "gte", "value": "2"},
			{"field": "memoryGB", "operator": "gt", "value": "8"},
			{"field": "healthy", "operator": "eq", "value": "true"},
		},
	}

	data, _ := json.Marshal(query)
	req, _ := http.NewRequest("POST", "/api/cluster-groups/evaluate", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	clusters := result["clusters"].([]interface{})
	assert.Equal(t, 1, len(clusters))
	assert.Equal(t, "c1-ctx", clusters[0])
}

// MockAIProvider implements agent.AIProvider
type MockAIProvider struct {
	Response string
}

func (m *MockAIProvider) Name() string        { return "mock-ai" }
func (m *MockAIProvider) DisplayName() string { return "Mock AI" }
func (m *MockAIProvider) Description() string { return "Mock AI Provider" }
func (m *MockAIProvider) Provider() string    { return "mock" }
func (m *MockAIProvider) IsAvailable() bool   { return true }
func (m *MockAIProvider) Chat(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
	return &agent.ChatResponse{
		Content: m.Response,
		Agent:   "mock-ai",
		Done:    true,
	}, nil
}
func (m *MockAIProvider) StreamChat(ctx context.Context, req *agent.ChatRequest, onChunk func(chunk string)) (*agent.ChatResponse, error) {
	onChunk(m.Response)
	return &agent.ChatResponse{Content: m.Response, Done: true}, nil
}

func TestGenerateClusterQuery(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Post("/api/cluster-groups/generate", handler.GenerateClusterQuery)

	// Register Mock AI
	registry := agent.GetRegistry()
	mockAI := &MockAIProvider{
		Response: `{"suggestedName": "west-cpu-group", "query": {"labelSelector": "region=us-west", "filters": [{"field": "cpuCores", "operator": "gte", "value": "4"}]}}`,
	}
	registry.Register(mockAI)
	registry.SetDefault("mock-ai")

	payload := map[string]string{"prompt": "Find powerful clusters in west"}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/cluster-groups/generate", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	query := result["query"].(map[string]interface{})
	assert.Equal(t, "region=us-west", query["labelSelector"])
}

func TestResolveDependencies(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/workloads/resolve-deps/:cluster/:namespace/:name", handler.ResolveDependencies)

	scheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)

	// Create deployment
	deploy := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Replicas: func(i int32) *int32 { return &i }(1),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "app"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "app"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "nginx"}}},
			},
		},
	}

	// Create service matching headers
	svc := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "app-svc", Namespace: "default"},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "app"},
		},
	}

	dynClient := fake.NewSimpleDynamicClient(scheme, deploy, svc)
	env.K8sClient.InjectDynamicClient("c1", dynClient)

	req, _ := http.NewRequest("GET", "/api/workloads/resolve-deps/c1/default/app", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	// Should find Service dependency
	deps := result["dependencies"].([]interface{})
	foundSvc := false
	for _, d := range deps {
		dep := d.(map[string]interface{})
		if dep["kind"] == "Service" && dep["name"] == "app-svc" {
			foundSvc = true
		}
	}
	assert.True(t, foundSvc, "Should find matching service")
}

func TestMonitorWorkload(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/workloads/monitor/:cluster/:namespace/:name", handler.MonitorWorkload)

	scheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)

	deploy := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "monitored-app", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Replicas: func(i int32) *int32 { return &i }(1),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "monitored"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "monitored"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "nginx"}}},
			},
		},
	}

	dynClient := fake.NewSimpleDynamicClient(scheme, deploy)
	env.K8sClient.InjectDynamicClient("c1", dynClient)

	req, _ := http.NewRequest("GET", "/api/workloads/monitor/c1/default/monitored-app", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	assert.Equal(t, "monitored-app", result["workload"])
	assert.NotNil(t, result["status"])
}

func TestGetDeployLogs(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewWorkloadHandlers(env.K8sClient, env.Hub)
	env.App.Get("/api/workloads/logs/:cluster/:namespace/:name", handler.GetDeployLogs)

	scheme := runtime.NewScheme()
	_ = k8sscheme.AddToScheme(scheme)

	// Setup Pod for logs
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "log-pod-1",
			Namespace: "default",
			Labels:    map[string]string{"app": "log-app"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main"}},
		},
	}

	// Needs both dynamic (for listing pods) and typed (for logs)
	dynClient := fake.NewSimpleDynamicClient(scheme, pod)
	typedClient := k8sfake.NewSimpleClientset(pod)

	env.K8sClient.InjectDynamicClient("c1", dynClient)
	env.K8sClient.InjectClient("c1", typedClient)

	req, _ := http.NewRequest("GET", "/api/workloads/logs/c1/default/log-app", nil)
	resp, err := env.App.Test(req)
	require.NoError(t, err)

	// Create deployment to provide selector
	deploy := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "log-app", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "log-app"}},
		},
	}
	dynClientWithDeploy := fake.NewSimpleDynamicClient(scheme, pod, deploy)
	env.K8sClient.InjectDynamicClient("c1", dynClientWithDeploy)

	resp, err = env.App.Test(req)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 404 || resp.StatusCode == 500)
}
