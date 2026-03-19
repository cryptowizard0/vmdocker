package runtimemanager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRuntimeWorkspaceDefaultsPerPID(t *testing.T) {
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	workspace, err := resolveRuntimeWorkspace("pid-1", "")
	if err != nil {
		t.Fatalf("resolveRuntimeWorkspace failed: %v", err)
	}

	realTempDir, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		t.Fatalf("eval symlinks temp dir failed: %v", err)
	}
	expected := filepath.Join(realTempDir, runtimeWorkspaceDir, "pid-1")
	if workspace != expected {
		t.Fatalf("workspace = %q, want %q", workspace, expected)
	}
}

func TestResolveRuntimeWorkspaceUsesAbsoluteRoot(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "workspace-root")

	workspace, err := resolveRuntimeWorkspace("pid-2", root)
	if err != nil {
		t.Fatalf("resolveRuntimeWorkspace failed: %v", err)
	}

	expectedRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("abs root failed: %v", err)
	}
	expected := filepath.Join(expectedRoot, runtimeWorkspaceDir, "pid-2")
	if workspace != expected {
		t.Fatalf("workspace = %q, want %q", workspace, expected)
	}
}

func TestEnsureRuntimeWorkspaceCreatesFullLayout(t *testing.T) {
	tempDir := t.TempDir()
	workspace, err := ensureRuntimeWorkspace("pid-3", filepath.Join(tempDir, "workspace-root"))
	if err != nil {
		t.Fatalf("ensureRuntimeWorkspace failed: %v", err)
	}
	for _, dir := range runtimeWorkspaceLayoutDirs(workspace) {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected runtime dir %s to exist: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", dir)
		}
	}
}

func TestEnsureRuntimeWorkspaceRootCreatesOnlyWorkspaceRoot(t *testing.T) {
	tempDir := t.TempDir()
	workspace, err := ensureRuntimeWorkspaceRoot("pid-4", filepath.Join(tempDir, "workspace-root"))
	if err != nil {
		t.Fatalf("ensureRuntimeWorkspaceRoot failed: %v", err)
	}

	info, err := os.Stat(workspace)
	if err != nil {
		t.Fatalf("expected workspace root %s to exist: %v", workspace, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", workspace)
	}

	for _, dir := range runtimeWorkspaceLayoutDirs(workspace)[1:] {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("expected runtime dir %s to be absent, got err=%v", dir, err)
		}
	}
}

func TestDockerAndSandboxShareRuntimeEnvironmentContract(t *testing.T) {
	workspace := "/tmp/workspace/sandbox_workspace/pid-1"
	env := appendRuntimePersistenceEnv([]string{"RUNTIME_TYPE=openclaw"}, workspace)
	full := strings.Join(env, "\n")

	for _, item := range []string{
		"RUNTIME_TYPE=openclaw",
		"OPENCLAW_STATE_DIR=/tmp/workspace/sandbox_workspace/pid-1/.openclaw",
		"OPENCLAW_HOME=/tmp/workspace/sandbox_workspace/pid-1",
		"OPENCLAW_CONFIG_PATH=/tmp/workspace/sandbox_workspace/pid-1/.openclaw/openclaw.json",
		"OPENCLAW_AGENT_WORKSPACE=/tmp/workspace/sandbox_workspace/pid-1/.openclaw/workspace",
		"HOME=/tmp/workspace/sandbox_workspace/pid-1/.home",
		"TMPDIR=/tmp/workspace/sandbox_workspace/pid-1/.tmp",
		"XDG_CONFIG_HOME=/tmp/workspace/sandbox_workspace/pid-1/.xdg/config",
		"XDG_CACHE_HOME=/tmp/workspace/sandbox_workspace/pid-1/.xdg/cache",
		"XDG_STATE_HOME=/tmp/workspace/sandbox_workspace/pid-1/.xdg/state",
	} {
		if !strings.Contains(full, item) {
			t.Fatalf("expected env %q in %v", item, env)
		}
	}
}
