package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/kubestellar/console/pkg/agent/protocol"
	"github.com/kubestellar/console/pkg/k8s"
	"github.com/kubestellar/console/pkg/settings"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestServer_HandleHealth(t *testing.T) {
	// 1. Setup mock kubectl proxy
	config := &api.Config{
		Contexts: map[string]*api.Context{
			"ctx-1": {Cluster: "c1"},
			"ctx-2": {Cluster: "c2"},
		},
	}
	mockProxy := &KubectlProxy{config: config}

	// 2. Setup server with mock dependencies
	server := &Server{
		kubectl:        mockProxy,
		allowedOrigins: []string{"http://allowed.com"},
		registry:       &Registry{providers: make(map[string]AIProvider)},
	}

	// 3. Create request
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "http://allowed.com") // Match allowed origin
	w := httptest.NewRecorder()

	// 4. Invoke handler
	server.handleHealth(w, req)

	// 5. Verify response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var payload protocol.HealthPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if payload.Status != "ok" {
		t.Errorf("Expected status 'ok', got %q", payload.Status)
	}
	if payload.Clusters != 2 {
		t.Errorf("Expected 2 clusters, got %d", payload.Clusters)
	}
}

func TestServer_HandleHealth_CORS(t *testing.T) {
	server := &Server{
		allowedOrigins: []string{"http://allowed.com"},
		registry:       &Registry{providers: make(map[string]AIProvider)},
		kubectl:        &KubectlProxy{config: &api.Config{}},
	}

	// Case 1: Allowed Origin
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "http://allowed.com")
	w := httptest.NewRecorder()
	server.handleHealth(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "http://allowed.com" {
		t.Error("CORS header missing for allowed origin")
	}

	// Case 2: Disallowed Origin
	req = httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "http://evil.com")
	w = httptest.NewRecorder()
	server.handleHealth(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS header present for disallowed origin")
	}
}

func TestServer_IsAllowedOrigin(t *testing.T) {
	server := &Server{
		allowedOrigins: []string{
			"http://localhost",
			"https://*.ibm.com",
		},
	}

	tests := []struct {
		origin string
		want   bool
	}{
		{"http://localhost", true},
		{"https://sub.ibm.com", true},
		{"https://deep.sub.ibm.com", true}, // Wildcard matches multiple levels? Assuming implementation uses robust matching
		{"http://ibm.com", false},          // Wrong scheme
		{"https://google.com", false},
		{"", false}, // Empty origin usually treated as allowed in checkOrigin logic, but isAllowedOrigin likely returns false map lookup
	}

	for _, tt := range tests {
		if got := server.isAllowedOrigin(tt.origin); got != tt.want {
			t.Errorf("isAllowedOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
		}
	}
}

func TestServer_HandleClustersHTTP(t *testing.T) {
	config := &api.Config{
		CurrentContext: "ctx-1",
		Contexts: map[string]*api.Context{
			"ctx-1": {Cluster: "c1", AuthInfo: "u1"},
		},
		Clusters: map[string]*api.Cluster{
			"c1": {Server: "https://c1.com"},
		},
	}
	mockProxy := &KubectlProxy{config: config}
	server := &Server{
		kubectl:        mockProxy,
		allowedOrigins: []string{"*"},
	}

	req := httptest.NewRequest("GET", "/clusters", nil)
	w := httptest.NewRecorder()

	server.handleClustersHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var payload protocol.ClustersPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("Failed to decode clusters payload: %v", err)
	}

	if len(payload.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(payload.Clusters))
	}
	if payload.Clusters[0].Name != "ctx-1" {
		t.Errorf("Expected cluster ctx-1, got %s", payload.Clusters[0].Name)
	}
}

func TestServer_HandleRenameContextHTTP(t *testing.T) {
	// Mock executing kubectl
	// We need to swap execCommand package-level variable in agent package
	// But we are in agent package (same package test), so we can access it directly IF it's exported or same package
	// It is unexported 'execCommand'.
	// In kubectl.go: var execCommand = exec.Command
	// In kubectl_test.go: func fakeExecCommand(...)

	// Since we are in the same package 'agent', we can use fakeExecCommand from kubectl_test.go!
	// Important: We need to coordinate concurrent access if tests run in parallel.
	// We are not using t.Parallel(), so it's safeish, but defer restore is critical.

	defer func() { execCommand = exec.Command }()
	execCommand = fakeExecCommand

	// Setup proxy
	proxy := &KubectlProxy{
		kubeconfig: "/tmp/config",
		config:     &api.Config{},
	}

	server := &Server{
		kubectl:        proxy,
		allowedOrigins: []string{"*"},
	}

	// Case 1: Success
	mockExitCode = 0
	body1 := `{"oldName":"old", "newName":"new"}`
	req := httptest.NewRequest("POST", "/rename-context", strings.NewReader(body1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRenameContextHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Case 2: Invalid JSON
	req = httptest.NewRequest("POST", "/rename-context", strings.NewReader("bad-json"))
	w = httptest.NewRecorder()
	server.handleRenameContextHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for bad json, got %d", w.Code)
	}

	// Case 3: Failure
	mockExitCode = 1
	mockStderr = "rename failed"
	body3 := `{"oldName":"bad", "newName":"new"}`
	req = httptest.NewRequest("POST", "/rename-context", strings.NewReader(body3))
	w = httptest.NewRecorder()
	server.handleRenameContextHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for failure, got %d", w.Code)
	}
}

func TestServer_ResourceHandlers(t *testing.T) {
	// Setup generic mock proxy
	defer func() { execCommand = exec.Command }()
	execCommand = fakeExecCommand

	config := &api.Config{
		CurrentContext: "ctx-1",
	}
	proxy := &KubectlProxy{
		config:     config,
		kubeconfig: "/tmp/config",
	}

	// Create mock k8s client
	k8sClient, _ := k8s.NewMultiClusterClient("")

	// Inject fake dynamic client for "ctx-1"
	scheme := runtime.NewScheme()
	fakeDyn := fake.NewSimpleDynamicClient(scheme)
	k8sClient.SetDynamicClient("ctx-1", fakeDyn)

	// Inject fake typed client for "ctx-1"
	fakeCS := fakek8s.NewSimpleClientset()
	k8sClient.SetClient("ctx-1", fakeCS)

	server := &Server{
		kubectl:        proxy,
		k8sClient:      k8sClient,
		allowedOrigins: []string{"*"},
	}

	tests := []struct {
		name    string
		path    string
		handler func(http.ResponseWriter, *http.Request)
		mockOut string
	}{
		{
			name:    "Namespaces",
			path:    "/namespaces?cluster=ctx-1",
			handler: server.handleNamespacesHTTP,
			mockOut: `{"namespaces":null,"source":"agent"}`,
		},
		{
			name:    "Nodes",
			path:    "/nodes?cluster=ctx-1",
			handler: server.handleNodesHTTP,
			mockOut: `{"nodes":null,"source":"agent"}`,
		},
		{
			name:    "Deployments",
			path:    "/deployments?namespace=default&cluster=ctx-1",
			handler: server.handleDeploymentsHTTP,
			mockOut: `{"deployments":null,"source":"agent"}`,
		},
		{
			name:    "Services",
			path:    "/services?namespace=kube-system&cluster=ctx-1",
			handler: server.handleServicesHTTP,
			mockOut: `{"services":null,"source":"agent"}`,
		},
		{
			name:    "StatefulSets",
			path:    "/statefulsets?namespace=default&cluster=ctx-1",
			handler: server.handleStatefulSetsHTTP,
			mockOut: `{"source":"agent","statefulsets":null}`,
		},
		{
			name:    "DaemonSets",
			path:    "/daemonsets?namespace=default&cluster=ctx-1",
			handler: server.handleDaemonSetsHTTP,
			mockOut: `{"daemonsets":null,"source":"agent"}`,
		},
		{
			name:    "ReplicaSets",
			path:    "/replicasets?namespace=default&cluster=ctx-1",
			handler: server.handleReplicaSetsHTTP,
			mockOut: `{"replicasets":null,"source":"agent"}`,
		},
		{
			name:    "CronJobs",
			path:    "/cronjobs?namespace=default&cluster=ctx-1",
			handler: server.handleCronJobsHTTP,
			mockOut: `{"cronjobs":null,"source":"agent"}`,
		},
		{
			name:    "Ingresses",
			path:    "/ingresses?namespace=default&cluster=ctx-1",
			handler: server.handleIngressesHTTP,
			mockOut: `{"ingresses":null,"source":"agent"}`,
		},
		{
			name:    "NetworkPolicies",
			path:    "/networkpolicies?namespace=default&cluster=ctx-1",
			handler: server.handleNetworkPoliciesHTTP,
			mockOut: `{"networkpolicies":null,"source":"agent"}`,
		},
		{
			name:    "ConfigMaps",
			path:    "/configmaps?namespace=default&cluster=ctx-1",
			handler: server.handleConfigMapsHTTP,
			mockOut: `{"configmaps":null,"source":"agent"}`,
		},
		{
			name:    "Secrets",
			path:    "/secrets?namespace=default&cluster=ctx-1",
			handler: server.handleSecretsHTTP,
			mockOut: `{"secrets":null,"source":"agent"}`,
		},
		{
			name:    "ServiceAccounts",
			path:    "/serviceaccounts?namespace=default&cluster=ctx-1",
			handler: server.handleServiceAccountsHTTP,
			mockOut: `{"serviceaccounts":null,"source":"agent"}`,
		},
		{
			name:    "Jobs",
			path:    "/jobs?namespace=default&cluster=ctx-1",
			handler: server.handleJobsHTTP,
			mockOut: `{"jobs":null,"source":"agent"}`,
		},
		{
			name:    "PVCs",
			path:    "/pvcs?namespace=default&cluster=ctx-1",
			handler: server.handlePVCsHTTP,
			mockOut: `{"pvcs":null,"source":"agent"}`,
		},
		{
			name:    "HPAs",
			path:    "/hpas?namespace=default&cluster=ctx-1",
			handler: server.handleHPAsHTTP,
			mockOut: `{"hpas":null,"source":"agent"}`,
		},
		{
			name:    "ClusterHealth",
			path:    "/health?cluster=ctx-1",
			handler: server.handleClusterHealthHTTP,
			mockOut: `{"cluster":"ctx-1","healthy":true`,
		},
		{
			name:    "Pods",
			path:    "/pods?namespace=default&cluster=ctx-1",
			handler: server.handlePodsHTTP,
			mockOut: `{"pods":null,"source":"agent"}`,
		},
		{
			name:    "GPUNodes",
			path:    "/gpu-nodes?cluster=ctx-1",
			handler: server.handleGPUNodesHTTP,
			mockOut: `{"nodes":null,"source":"agent"}`,
		},
		{
			name:    "Events",
			path:    "/events?cluster=ctx-1",
			handler: server.handleEventsHTTP,
			mockOut: `{"events":null,"source":"agent"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStdout = tt.mockOut
			mockExitCode = 0
			// Reset stderr for clean test
			mockStderr = ""

			req := httptest.NewRequest("GET", tt.path, nil)
			// Add query for namespace if present in path
			if strings.Contains(tt.path, "?") {
				parts := strings.Split(tt.path, "?")
				req.URL.RawQuery = parts[1]
			}

			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// We can't easily assert the output because execCommand is package-level and shared.
			// But we mock mockStdout
			// However, in our fakeExecCommand, we just write mockStdout to stdout.
			// The handler reads it.
			// So w.Body should contain mockStdout.
			// Note: strings.TrimSpace might be used by handler? Or JSON encoder?
			// Handlers usually do w.Write([]byte(output)).

			if !strings.Contains(w.Body.String(), tt.mockOut) {
				t.Errorf("Expected body to contain %q, got %q", tt.mockOut, w.Body.String())
			}
		})
	}
}

func TestServer_SettingsHandlers(t *testing.T) {
	// 1. Setup temporary config
	cm := GetConfigManager()
	oldPath := cm.GetConfigPath()
	tmpFile := "/tmp/agent-test-config.yaml"
	cm.SetConfigPath(tmpFile)
	defer func() {
		cm.SetConfigPath(oldPath)
		os.Remove(tmpFile)
	}()

	server := &Server{
		allowedOrigins:    []string{"*"},
		SkipKeyValidation: true,
	}

	// 2. Test handleSetKey
	reqBody := `{"provider":"openai", "apiKey":"test-key", "model":"gpt-4"}`
	req := httptest.NewRequest("POST", "/settings/keys", strings.NewReader(reqBody))
	w := httptest.NewRecorder()
	server.handleSettingsKeys(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleSetKey failed: %d - %s", w.Code, w.Body.String())
	}

	// Verify key was saved
	if cm.GetAPIKey("openai") != "test-key" {
		t.Error("API key not saved in config manager")
	}

	// 3. Test handleGetKeysStatus
	req = httptest.NewRequest("GET", "/settings/keys", nil)
	w = httptest.NewRecorder()
	server.handleSettingsKeys(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleGetKeysStatus failed: %d", w.Code)
	}

	var resp KeysStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode keys status: %v", err)
	}

	foundOpenAI := false
	for _, k := range resp.Keys {
		if k.Provider == "openai" {
			foundOpenAI = true
			if !k.Configured {
				t.Error("OpenAI should be configured")
			}
			if k.Valid == nil || !*k.Valid {
				t.Error("OpenAI key should be valid (skipped validation)")
			}
		}
	}
	if !foundOpenAI {
		t.Error("OpenAI provider not found in status response")
	}
}

// ServerMockProvider for testing handleChatMessage
type ServerMockProvider struct {
	name string
}

func (m *ServerMockProvider) Name() string        { return m.name }
func (m *ServerMockProvider) DisplayName() string { return m.name }
func (m *ServerMockProvider) Description() string { return m.name }
func (m *ServerMockProvider) Provider() string    { return "mock" }
func (m *ServerMockProvider) IsAvailable() bool   { return true }
func (m *ServerMockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{
		Content: "Mock response: " + req.Prompt,
		Agent:   m.name,
		TokenUsage: &ProviderTokenUsage{
			InputTokens:  1,
			OutputTokens: 2,
			TotalTokens:  3,
		},
		Done: true,
	}, nil
}
func (m *ServerMockProvider) StreamChat(ctx context.Context, req *ChatRequest, onChunk func(chunk string)) (*ChatResponse, error) {
	onChunk("Mock chunk")
	return m.Chat(ctx, req)
}

func TestServer_HandleChatMessage(t *testing.T) {
	registry := &Registry{
		providers:     map[string]AIProvider{"mock": &ServerMockProvider{name: "mock"}},
		selectedAgent: make(map[string]string),
	}
	server := &Server{
		registry: registry,
	}

	chatReq := protocol.ChatRequest{
		Prompt:    "Hello Test",
		SessionID: "session-1",
		Agent:     "mock",
	}

	msg := protocol.Message{
		ID:      "msg-1",
		Type:    protocol.TypeChat,
		Payload: chatReq,
	}

	respMsg := server.handleChatMessage(msg, "")

	if respMsg.Type != protocol.TypeResult {
		t.Errorf("Expected TypeResult, got %s", respMsg.Type)
	}

	payload, ok := respMsg.Payload.(protocol.ChatStreamPayload)
	if !ok {
		// handleChatMessage encodes payload as ChatStreamPayload
		// but since it's an interface, let's see how it's handled.
		// In go, the return from handleChatMessage has Payload as protocol.ChatStreamPayload
		t.Fatalf("Expected ChatStreamPayload, got %T", respMsg.Payload)
	}

	if payload.Content != "Mock response: Hello Test" {
		t.Errorf("Unexpected content: %s", payload.Content)
	}
}

func TestServer_SettingsAll(t *testing.T) {
	// Setup temporary settings paths
	sm := settings.GetSettingsManager()
	oldSettingsPath := sm.GetSettingsPath()
	tmpSettings := "/tmp/test-settings.json"
	tmpKey := "/tmp/test-keyfile"
	sm.SetSettingsPath(tmpSettings)
	sm.SetKeyPath(tmpKey)
	defer func() {
		sm.SetSettingsPath(oldSettingsPath)
		os.Remove(tmpSettings)
		os.Remove(tmpKey)
	}()

	server := &Server{
		allowedOrigins: []string{"*"},
	}

	// 1. Test GET /settings (initial default)
	req := httptest.NewRequest("GET", "/settings", nil)
	w := httptest.NewRecorder()
	server.handleSettingsAll(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /settings failed: %d", w.Code)
	}

	var all settings.AllSettings
	if err := json.Unmarshal(w.Body.Bytes(), &all); err != nil {
		t.Fatalf("Failed to unmarshal settings: %v", err)
	}

	// 2. Test PUT /settings
	all.Theme = "dark"
	all.APIKeys = map[string]settings.APIKeyEntry{
		"openai": {APIKey: "sk-test", Model: "gpt-4o"},
	}

	body, _ := json.Marshal(all)
	req = httptest.NewRequest("PUT", "/settings", strings.NewReader(string(body)))
	w = httptest.NewRecorder()
	server.handleSettingsAll(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PUT /settings failed: %d", w.Code)
	}

	// 3. Verify saved settings
	req = httptest.NewRequest("GET", "/settings", nil)
	w = httptest.NewRecorder()
	server.handleSettingsAll(w, req)

	var saved settings.AllSettings
	json.Unmarshal(w.Body.Bytes(), &saved)
	if saved.Theme != "dark" {
		t.Errorf("Expected theme dark, got %s", saved.Theme)
	}
	if saved.APIKeys["openai"].Model != "gpt-4o" {
		t.Errorf("Expected gpt-4o, got %s", saved.APIKeys["openai"].Model)
	}
}
