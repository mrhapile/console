package agent

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"
)

// Mock configuration variables
var (
	mockStdout   string
	mockStderr   string
	mockExitCode int
)

// fakeExecCommand mimics exec.Command but calls a helper test function
func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{
		"GO_WANT_HELPER_PROCESS=1",
		fmt.Sprintf("MOCK_STDOUT=%s", mockStdout),
		fmt.Sprintf("MOCK_STDERR=%s", mockStderr),
		fmt.Sprintf("MOCK_EXIT_CODE=%d", mockExitCode),
		// Prevent coverage warning from polluting stderr
		"GOCOVERDIR=" + os.TempDir(),
	}
	return cmd
}

// TestHelperProcess is the function executed by the fake command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Write mock stdout
	fmt.Fprint(os.Stdout, os.Getenv("MOCK_STDOUT"))

	// Write mock stderr
	fmt.Fprint(os.Stderr, os.Getenv("MOCK_STDERR"))

	// Exit with mock code
	exitCode := 0
	if code := os.Getenv("MOCK_EXIT_CODE"); code != "" {
		fmt.Sscanf(code, "%d", &exitCode)
	}
	os.Exit(exitCode)
}

func TestKubectlProxy_Execute(t *testing.T) {
	// Restore original execCommand after tests
	defer func() { execCommand = exec.Command }()
	execCommand = fakeExecCommand

	tests := []struct {
		name          string
		args          []string
		mockStdout    string
		mockStderr    string
		mockExitCode  int
		wantOutput    string
		wantError     string
		wantExitCode  int
		expectBlocked bool
	}{
		{
			name:         "Successful get pods",
			args:         []string{"get", "pods"},
			mockStdout:   "pod-1\npod-2",
			mockExitCode: 0,
			wantOutput:   "pod-1\npod-2",
			wantExitCode: 0,
		},
		{
			name:         "Failed command",
			args:         []string{"get", "pods"},
			mockStderr:   "namespace not found",
			mockExitCode: 1,
			wantError:    "namespace not found",
			wantOutput:   "namespace not found", // Agent returns stderr as output if stdout is empty
			wantExitCode: 1,
		},
		{
			name:          "Blocked command (exec)",
			args:          []string{"exec", "-it", "pod", "--", "sh"},
			expectBlocked: true,
			wantError:     "Disallowed kubectl command",
			wantExitCode:  1,
		},
		{
			name:          "Blocked command (delete deployment)", // Only pods allowed
			args:          []string{"delete", "deployment", "foo"},
			expectBlocked: true,
			wantError:     "Disallowed kubectl command",
			wantExitCode:  1,
		},
		{
			name:         "Allowed delete pod",
			args:         []string{"delete", "pod", "foo"},
			mockStdout:   "pod deleted",
			mockExitCode: 0,
			wantOutput:   "pod deleted",
			wantExitCode: 0,
		},
	}

	proxy := &KubectlProxy{config: &api.Config{}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set mock expectations
			mockStdout = tt.mockStdout
			mockStderr = tt.mockStderr
			mockExitCode = tt.mockExitCode

			resp := proxy.Execute("default", "default", tt.args)

			if tt.expectBlocked {
				if resp.ExitCode == 0 {
					t.Errorf("Expected command to be blocked, but got success")
				}
				if resp.Error != "Disallowed kubectl command" {
					t.Errorf("Expected 'Disallowed kubectl command', got '%s'", resp.Error)
				}
				return
			}

			if resp.ExitCode != tt.wantExitCode {
				t.Errorf("ExitCode = %d, want %d", resp.ExitCode, tt.wantExitCode)
			}

			// The agent logic: if output is empty but stderr is not, output = stderr
			if resp.Output != tt.wantOutput {
				t.Errorf("Output = %q, want %q", resp.Output, tt.wantOutput)
			}

			if tt.wantError != "" && resp.Error != tt.wantError {
				t.Errorf("Error = %q, want %q", resp.Error, tt.wantError)
			}
		})
	}
}

func TestKubectlProxy_ValidateArgs(t *testing.T) {
	proxy := &KubectlProxy{}

	tests := []struct {
		args  []string
		valid bool
	}{
		{[]string{"get", "pods"}, true},
		{[]string{"get", "nodes"}, true},
		{[]string{"describe", "pod", "foo"}, true},
		{[]string{"scale", "deployment", "foo", "--replicas=3"}, true},
		{[]string{"scale", "sts/foo", "--replicas=3"}, true},
		{[]string{"delete", "pod", "foo"}, true},

		// Blocked cases
		{[]string{"apply", "-f", "file.yaml"}, false},
		{[]string{"exec", "pod", "--", "ls"}, false},
		{[]string{"delete", "node", "foo"}, false},
		{[]string{"get", "pods", ";", "rm", "-rf", "/"}, false},
		{[]string{"config", "view"}, true},
		{[]string{"config", "set-context", "foo"}, false}, // Mutation blocked
	}

	for _, tt := range tests {
		valid := proxy.validateArgs(tt.args)
		if valid != tt.valid {
			t.Errorf("validateArgs(%v) = %v, want %v", tt.args, valid, tt.valid)
		}
	}
}

func TestKubectlProxy_ListContexts(t *testing.T) {
	// Setup mock config
	config := &api.Config{
		CurrentContext: "ctx-1",
		Contexts: map[string]*api.Context{
			"ctx-1": {Cluster: "cluster-1", AuthInfo: "user-1", Namespace: "ns-1"},
			"ctx-2": {Cluster: "cluster-2", AuthInfo: "user-2", Namespace: "default"},
		},
		Clusters: map[string]*api.Cluster{
			"cluster-1": {Server: "https://c1.example.com"},
			"cluster-2": {Server: "https://c2.example.com"},
		},
	}

	proxy := &KubectlProxy{config: config}

	clusters, current := proxy.ListContexts()

	if current != "ctx-1" {
		t.Errorf("Current context = %q, want %q", current, "ctx-1")
	}

	if len(clusters) != 2 {
		t.Errorf("Got %d clusters, want 2", len(clusters))
	}

	// Verify one of the clusters
	found := false
	for _, c := range clusters {
		if c.Name == "ctx-1" {
			found = true
			if c.Server != "https://c1.example.com" {
				t.Errorf("Cluster server = %q, want %q", c.Server, "https://c1.example.com")
			}
			if c.IsCurrent != true {
				t.Errorf("IsCurrent = %v, want true", c.IsCurrent)
			}
			if c.Namespace != "ns-1" {
				t.Errorf("Namespace = %q, want %q", c.Namespace, "ns-1")
			}
		}
	}
	if !found {
		t.Error("ctx-1 not found in result")
	}
}

func TestKubectlProxy_RenameContext(t *testing.T) {
	// Restore original execCommand after tests
	defer func() { execCommand = exec.Command }()
	execCommand = fakeExecCommand

	proxy := &KubectlProxy{
		kubeconfig: "/tmp/fake-config",
		config:     &api.Config{},
	}

	// 1. Successful rename
	mockExitCode = 0
	err := proxy.RenameContext("old-ctx", "new-ctx")
	if err != nil {
		t.Errorf("RenameContext failed: %v", err)
	}

	// 2. Failed rename
	mockExitCode = 1
	mockStderr = "error: context not found"
	err = proxy.RenameContext("missing-ctx", "new-ctx")
	if err == nil {
		t.Error("RenameContext should fail when kubectl fails")
	}
}

func TestKubectlProxy_Execute_Flags(t *testing.T) {
	// Restore original execCommand after tests
	defer func() { execCommand = exec.Command }()
	execCommand = fakeExecCommand

	proxy := &KubectlProxy{
		kubeconfig: "/tmp/config",
	}

	// Capture arguments passed to fakeExecCommand
	// Note: effectively checking "Execute" implementation details via side-effect on what fakeExecCommand would receive if we could inspect it easily.
	// Since fakeExecCommand runs a subprocess, we can't easily inspect args *inside* this test process unless we pass them back.
	// But we can check that it DOES NOT error on valid construction.

	// We'll rely on the fact that Execute builds args.
	// We can verify specific behaviors by mocking output.

	tests := []struct {
		name      string
		context   string
		namespace string
		args      []string
		wantErr   bool
	}{
		{"With context and namespace", "my-ctx", "my-ns", []string{"get", "pods"}, false},
		{"Empty context and namespace", "", "", []string{"get", "nodes"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExitCode = 0
			resp := proxy.Execute(tt.context, tt.namespace, tt.args)
			if tt.wantErr && resp.ExitCode == 0 {
				t.Error("Expected error, got success")
			}
			if !tt.wantErr && resp.ExitCode != 0 {
				t.Errorf("Expected success, got exit code %d", resp.ExitCode)
			}
		})
	}
}

func TestKubectlProxy_Helpers(t *testing.T) {
	proxy := &KubectlProxy{
		kubeconfig: "/tmp/config",
		config: &api.Config{
			CurrentContext: "my-ctx",
		},
	}

	if proxy.GetCurrentContext() != "my-ctx" {
		t.Errorf("GetCurrentContext() = %q, want %q", proxy.GetCurrentContext(), "my-ctx")
	}

	if proxy.GetKubeconfigPath() != "/tmp/config" {
		t.Errorf("GetKubeconfigPath() = %q, want %q", proxy.GetKubeconfigPath(), "/tmp/config")
	}
}

func TestNewKubectlProxy(t *testing.T) {
	// 1. With explicit path
	proxy, err := NewKubectlProxy("/tmp/missing-config")
	if err != nil {
		t.Fatalf("NewKubectlProxy failed: %v", err)
	}
	if proxy.GetKubeconfigPath() != "/tmp/missing-config" {
		t.Errorf("Path mismatch: %s", proxy.GetKubeconfigPath())
	}

	// 2. With KUBECONFIG env
	os.Setenv("KUBECONFIG", "/tmp/env-config")
	defer os.Unsetenv("KUBECONFIG")
	proxy2, _ := NewKubectlProxy("")
	if proxy2.GetKubeconfigPath() != "/tmp/env-config" {
		t.Errorf("Path mismatch from env: %s", proxy2.GetKubeconfigPath())
	}
}
