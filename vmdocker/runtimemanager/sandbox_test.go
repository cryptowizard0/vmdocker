package runtimemanager

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
)

func TestSandboxManagerCreateAndStartInstanceUsesTemplateWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "docker.log")
	fakeDocker := filepath.Join(tempDir, "docker")
	script := "#!/bin/sh\nprintf '%s\n' \"$*\" >>" + shellEscapeForTest(logPath) + "\nif [ \"$1\" = \"--help\" ]; then\n  echo sandbox\n  exit 0\nfi\nif [ \"$1\" = \"image\" ] && [ \"$2\" = \"inspect\" ]; then\n  echo '[{\"Id\":\"sha256:template-id\",\"RepoDigests\":[\"chriswebber/docker-openclaw-sandbox@test-sha256:expected\"]}]'\n  exit 0\nfi\nexit 0\n"
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker failed: %v", err)
	}

	sm, err := newSandboxManager()
	if err != nil {
		t.Fatalf("newSandboxManager failed: %v", err)
	}
	sm.cliBin = fakeDocker

	spec := schema.RuntimeSpec{
		Backend: schema.BackendSandbox,
		Image: schema.ImageInfo{
			Name: "chriswebber/docker-openclaw-sandbox:test",
			SHA:  "sha256:expected",
		},
		Sandbox: schema.SandboxSpec{
			Agent:     "shell",
			Workspace: filepath.Join(tempDir, "workspace"),
			Name:      "runtime-pid-1",
		},
	}

	if _, err := sm.CreateInstance(context.Background(), "pid-1", spec, []string{"RUNTIME_TYPE=openclaw"}); err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	if err := sm.StartInstance(context.Background(), "pid-1"); err != nil {
		t.Fatalf("StartInstance failed: %v", err)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake docker log failed: %v", err)
	}
	log := string(raw)

	if !strings.Contains(log, "sandbox create --name runtime-pid-1 --pull-template missing -t chriswebber/docker-openclaw-sandbox:test shell "+filepath.Join(tempDir, "workspace")) {
		t.Fatalf("expected sandbox create command in log, got:\n%s", log)
	}
	if !strings.Contains(log, "image inspect chriswebber/docker-openclaw-sandbox:test") {
		t.Fatalf("expected template image inspect in log, got:\n%s", log)
	}
	if !strings.Contains(log, "sandbox exec -e RUNTIME_TYPE=openclaw runtime-pid-1 sh -lc") {
		t.Fatalf("expected sandbox exec with env in log, got:\n%s", log)
	}
	if !strings.Contains(log, "start-vmdocker-agent.sh") {
		t.Fatalf("expected start-vmdocker-agent.sh in log, got:\n%s", log)
	}
	if strings.Contains(log, "docker run") {
		t.Fatalf("unexpected nested docker run in log:\n%s", log)
	}
}

func TestSandboxManagerCreateInstancePullsAndVerifiesMissingTemplate(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "docker.log")
	statePath := filepath.Join(tempDir, "inspect-state")
	fakeDocker := filepath.Join(tempDir, "docker")
	script := "#!/bin/sh\nprintf '%s\n' \"$*\" >>" + shellEscapeForTest(logPath) + "\nif [ \"$1\" = \"--help\" ]; then\n  echo sandbox\n  exit 0\nfi\nif [ \"$1\" = \"image\" ] && [ \"$2\" = \"inspect\" ]; then\n  if [ ! -f " + shellEscapeForTest(statePath) + " ]; then\n    echo 'Error: No such image' >&2\n    exit 1\n  fi\n  echo '[{\"Id\":\"sha256:template-id\",\"RepoDigests\":[\"chriswebber/docker-openclaw-sandbox@test-sha256:expected\"]}]'\n  exit 0\nfi\nif [ \"$1\" = \"pull\" ]; then\n  : > " + shellEscapeForTest(statePath) + "\n  exit 0\nfi\nexit 0\n"
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker failed: %v", err)
	}

	sm, err := newSandboxManager()
	if err != nil {
		t.Fatalf("newSandboxManager failed: %v", err)
	}
	sm.cliBin = fakeDocker

	spec := schema.RuntimeSpec{
		Backend: schema.BackendSandbox,
		Image: schema.ImageInfo{
			Name: "chriswebber/docker-openclaw-sandbox:test",
			SHA:  "sha256:expected",
		},
		Sandbox: schema.SandboxSpec{
			Agent:     "shell",
			Workspace: filepath.Join(tempDir, "workspace"),
			Name:      "runtime-pid-2",
		},
	}

	if _, err := sm.CreateInstance(context.Background(), "pid-2", spec, nil); err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake docker log failed: %v", err)
	}
	log := string(raw)
	if !strings.Contains(log, "pull chriswebber/docker-openclaw-sandbox:test") {
		t.Fatalf("expected template image pull in log, got:\n%s", log)
	}
	if strings.Count(log, "image inspect chriswebber/docker-openclaw-sandbox:test") < 2 {
		t.Fatalf("expected template image inspect before and after pull, got:\n%s", log)
	}
}

func TestSandboxManagerCreateInstanceDefaultsWorkspacePerPid(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "docker.log")
	fakeDocker := filepath.Join(tempDir, "docker")
	script := "#!/bin/sh\nprintf '%s\n' \"$*\" >>" + shellEscapeForTest(logPath) + "\nif [ \"$1\" = \"--help\" ]; then\n  echo sandbox\n  exit 0\nfi\nif [ \"$1\" = \"image\" ] && [ \"$2\" = \"inspect\" ]; then\n  echo '[{\"Id\":\"sha256:template-id\",\"RepoDigests\":[\"chriswebber/docker-openclaw-sandbox@test-sha256:expected\"]}]'\n  exit 0\nfi\nexit 0\n"
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker failed: %v", err)
	}

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

	sm, err := newSandboxManager()
	if err != nil {
		t.Fatalf("newSandboxManager failed: %v", err)
	}
	sm.cliBin = fakeDocker

	spec := schema.RuntimeSpec{
		Backend: schema.BackendSandbox,
		Image: schema.ImageInfo{
			Name: "chriswebber/docker-openclaw-sandbox:test",
			SHA:  "sha256:expected",
		},
		Sandbox: schema.SandboxSpec{
			Agent: "shell",
			Name:  "runtime-pid-3",
		},
	}

	if _, err := sm.CreateInstance(context.Background(), "pid-3", spec, nil); err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	realTempDir, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		t.Fatalf("eval symlinks temp dir failed: %v", err)
	}
	expectedWorkspace := filepath.Join(realTempDir, "sandbox_workspace", "pid-3")
	if _, err := os.Stat(expectedWorkspace); err != nil {
		t.Fatalf("expected workspace directory to exist: %v", err)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake docker log failed: %v", err)
	}
	log := string(raw)
	if !strings.Contains(log, "sandbox create --name runtime-pid-3 --pull-template missing -t chriswebber/docker-openclaw-sandbox:test shell "+expectedWorkspace) {
		t.Fatalf("expected sandbox create command with default workspace in log, got:\n%s", log)
	}
}

func TestSandboxManagerCreateInstanceDefaultsSandboxNameToPidPrefix(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "docker.log")
	fakeDocker := filepath.Join(tempDir, "docker")
	script := "#!/bin/sh\nprintf '%s\n' \"$*\" >>" + shellEscapeForTest(logPath) + "\nif [ \"$1\" = \"--help\" ]; then\n  echo sandbox\n  exit 0\nfi\nif [ \"$1\" = \"image\" ] && [ \"$2\" = \"inspect\" ]; then\n  echo '[{\"Id\":\"sha256:template-id\",\"RepoDigests\":[\"chriswebber/docker-openclaw-sandbox@test-sha256:expected\"]}]'\n  exit 0\nfi\nexit 0\n"
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker failed: %v", err)
	}

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

	sm, err := newSandboxManager()
	if err != nil {
		t.Fatalf("newSandboxManager failed: %v", err)
	}
	sm.cliBin = fakeDocker

	pid := "4qIDQKNWm6kF4aDI-rpz_2VLkKWSLVxCJVw2RGJayoQ"
	spec := schema.RuntimeSpec{
		Backend: schema.BackendSandbox,
		Image: schema.ImageInfo{
			Name: "chriswebber/docker-openclaw-sandbox:test",
			SHA:  "sha256:expected",
		},
		Sandbox: schema.SandboxSpec{
			Agent: "shell",
		},
	}

	instance, err := sm.CreateInstance(context.Background(), pid, spec, nil)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	if instance.ID != "hymatrix_4qIDQKNWm6" {
		t.Fatalf("expected shortened sandbox name, got %q", instance.ID)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake docker log failed: %v", err)
	}
	log := string(raw)
	if !strings.Contains(log, "sandbox create --name hymatrix_4qIDQKNWm6 --pull-template missing -t chriswebber/docker-openclaw-sandbox:test shell ") {
		t.Fatalf("expected shortened sandbox name in log, got:\n%s", log)
	}
}

func shellEscapeForTest(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
