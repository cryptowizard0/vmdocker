package runtimemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
)

const (
	defaultSandboxAgent    = "shell"
	runtimeContainerPrefix = "runtime_"
)

type SandboxManager struct {
	instances     map[string]*schema.InstanceInfo
	launchSpecs   map[string]sandboxLaunchSpec
	mutex         sync.RWMutex
	portAllocator *portAllocator
	cliBin        string
}

type sandboxLaunchSpec struct {
	runtimeSpec schema.RuntimeSpec
	runtimeEnv  []string
}

func newSandboxManager() (*SandboxManager, error) {
	cliBin, err := exec.LookPath("docker")
	if err != nil {
		cliBin = "docker"
	}

	return &SandboxManager{
		instances:     make(map[string]*schema.InstanceInfo),
		launchSpecs:   make(map[string]sandboxLaunchSpec),
		portAllocator: newPortAllocator(10000, 20000),
		cliBin:        cliBin,
	}, nil
}

func (sm *SandboxManager) CreateInstance(ctx context.Context, pid string, runtimeSpec schema.RuntimeSpec, runtimeEnv []string) (*schema.InstanceInfo, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	log.Info("creating sandbox runtime instance", "pid", pid, "template_image", runtimeSpec.Image.Name, "env_count", len(runtimeEnv))
	if err := sm.ensureSandboxCLI(ctx); err != nil {
		return nil, err
	}
	if err := sm.ensureTemplateExists(ctx, runtimeSpec.Image); err != nil {
		return nil, err
	}

	port, err := sm.portAllocator.Allocate()
	if err != nil {
		return nil, err
	}

	sandboxName := runtimeSpec.Sandbox.Name
	if sandboxName == "" {
		sandboxName = defaultSandboxName(pid)
	}

	workspace := runtimeSpec.Sandbox.Workspace
	if workspace == "" {
		workspace, err = defaultSandboxWorkspace(pid)
		if err != nil {
			sm.portAllocator.Release(port)
			return nil, err
		}
	}
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		sm.portAllocator.Release(port)
		return nil, fmt.Errorf("create sandbox workspace failed: %w", err)
	}

	agent := runtimeSpec.Sandbox.Agent
	if agent == "" {
		agent = defaultSandboxAgent
	}

	args := []string{
		"sandbox", "create",
		"--name", sandboxName,
		"--pull-template", "missing",
		"-t", runtimeSpec.Image.Name,
		agent, workspace,
	}
	log.Debug("creating sandbox", "pid", pid, "sandbox_name", sandboxName, "agent", agent, "workspace", workspace, "template_image", runtimeSpec.Image.Name)

	if _, err := sm.runSandboxCommand(ctx, args...); err != nil {
		sm.portAllocator.Release(port)
		return nil, err
	}

	instance := &schema.InstanceInfo{
		ID:       sandboxName,
		Name:     pid,
		Port:     port,
		Status:   "created",
		CreateAt: time.Now(),
		Backend:  schema.BackendSandbox,
		Agent:    agent,
	}
	sm.instances[pid] = instance
	sm.launchSpecs[pid] = sandboxLaunchSpec{runtimeSpec: runtimeSpec, runtimeEnv: append([]string(nil), runtimeEnv...)}
	log.Info("sandbox runtime instance created", "pid", pid, "sandbox_name", sandboxName, "port", port)
	return instance, nil
}

func defaultSandboxWorkspace(pid string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, "sandbox_workspace", pid), nil
}

func defaultSandboxName(pid string) string {
	const maxPIDPrefixLen = 10
	if len(pid) > maxPIDPrefixLen {
		pid = pid[:maxPIDPrefixLen]
	}
	return ContainerNamePrefix + pid
}

func (sm *SandboxManager) GetInstance(pid string) (*schema.InstanceInfo, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	instance, exists := sm.instances[pid]
	if !exists {
		return nil, fmt.Errorf("instance not found")
	}
	return instance, nil
}

func (sm *SandboxManager) StartInstance(ctx context.Context, pid string) error {
	log.Info("starting sandbox runtime instance", "pid", pid)
	sm.mutex.RLock()
	launchSpec, exists := sm.launchSpecs[pid]
	sm.mutex.RUnlock()
	if !exists {
		return fmt.Errorf("sandbox launch spec not found: %s", pid)
	}

	return sm.startSandboxRuntime(ctx, pid, launchSpec.runtimeSpec, launchSpec.runtimeEnv)
}

func (sm *SandboxManager) startSandboxRuntime(ctx context.Context, pid string, runtimeSpec schema.RuntimeSpec, runtimeEnv []string) error {
	instance, err := sm.GetInstance(pid)
	if err != nil {
		return err
	}

	command := runtimeSpec.Sandbox.Command
	if command == "" {
		command = buildSandboxStartCommand()
	}
	log.Debug("executing sandbox runtime start command", "pid", pid, "sandbox_name", instance.ID, "command", command, "env_count", len(runtimeEnv))

	if _, err := sm.ExecInstance(ctx, pid, runtimeEnv, command); err != nil {
		return err
	}

	instance.Status = "running"
	log.Info("sandbox runtime instance started", "pid", pid, "sandbox_name", instance.ID)
	return nil
}

func (sm *SandboxManager) StopInstance(ctx context.Context, pid string) error {
	log.Info("stopping sandbox runtime instance", "pid", pid)
	instance, err := sm.GetInstance(pid)
	if err != nil {
		return err
	}

	if _, err := sm.runSandboxCommand(ctx, "sandbox", "stop", instance.ID); err != nil {
		return err
	}

	instance.Status = "stopped"
	log.Info("sandbox runtime instance stopped", "pid", pid, "sandbox_name", instance.ID)
	return nil
}

func (sm *SandboxManager) RemoveInstance(ctx context.Context, pid string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	log.Info("removing sandbox runtime instance", "pid", pid)
	instance, exists := sm.instances[pid]
	if !exists {
		return fmt.Errorf("instance not found: %s", pid)
	}

	if _, err := sm.runSandboxCommand(ctx, "sandbox", "stop", instance.ID); err != nil && !strings.Contains(err.Error(), "not running") {
		return err
	}

	if _, err := sm.runSandboxCommand(ctx, "sandbox", "rm", instance.ID); err != nil {
		return err
	}

	sm.portAllocator.Release(instance.Port)
	delete(sm.instances, pid)
	delete(sm.launchSpecs, pid)
	log.Info("sandbox runtime instance removed", "pid", pid, "sandbox_name", instance.ID)
	return nil
}

func (sm *SandboxManager) Checkpoint(context.Context, string, string) (string, error) {
	return "", schema.ErrNotSupported
}

func (sm *SandboxManager) Restore(context.Context, string, string, string) error {
	return schema.ErrNotSupported
}

func (sm *SandboxManager) ensureSandboxCLI(ctx context.Context) error {
	log.Debug("checking docker sandbox cli", "cli_bin", sm.cliBin)
	output, err := exec.CommandContext(ctx, sm.cliBin, "--help").CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker CLI is not available: %w", err)
	}
	if !strings.Contains(string(output), "sandbox") {
		return fmt.Errorf("docker sandbox CLI is not available on this machine")
	}
	return nil
}

type dockerImageInspectResult struct {
	ID          string   `json:"Id"`
	RepoDigests []string `json:"RepoDigests"`
}

func (sm *SandboxManager) ensureTemplateExists(ctx context.Context, imageInfo schema.ImageInfo) error {
	log.Debug("ensure sandbox template image exists", "image", imageInfo.Name, "expected_sha", imageInfo.SHA)
	if _, err := sm.inspectTemplateImage(ctx, imageInfo.Name); err == nil {
		log.Debug("sandbox template image already present locally", "image", imageInfo.Name)
		return sm.verifyTemplateSHA(ctx, imageInfo)
	}

	log.Info("pulling sandbox template image", "image", imageInfo.Name)
	if _, err := sm.runSandboxCommand(ctx, "pull", imageInfo.Name); err != nil {
		return fmt.Errorf("failed to pull template image %s: %v", imageInfo.Name, err)
	}

	return sm.verifyTemplateSHA(ctx, imageInfo)
}

func (sm *SandboxManager) verifyTemplateSHA(ctx context.Context, imageInfo schema.ImageInfo) error {
	log.Debug("verifying sandbox template image sha", "image", imageInfo.Name, "expected_sha", imageInfo.SHA)
	inspect, err := sm.inspectTemplateImage(ctx, imageInfo.Name)
	if err != nil {
		return fmt.Errorf("failed to inspect template image %s: %v", imageInfo.Name, err)
	}

	for _, digest := range inspect.RepoDigests {
		if strings.Contains(digest, imageInfo.SHA) {
			return nil
		}
	}
	if inspect.ID == imageInfo.SHA {
		log.Debug("sandbox template image sha matched local image id", "image", imageInfo.Name, "image_id", inspect.ID)
		return nil
	}

	return fmt.Errorf("template image SHA verification failed for %s: expected %s, got digests %v and ID %s",
		imageInfo.Name, imageInfo.SHA, inspect.RepoDigests, inspect.ID)
}

func (sm *SandboxManager) inspectTemplateImage(ctx context.Context, imageName string) (*dockerImageInspectResult, error) {
	log.Debug("inspecting sandbox template image", "image", imageName)
	output, err := sm.runSandboxCommand(ctx, "image", "inspect", imageName)
	if err != nil {
		return nil, err
	}

	var inspect []dockerImageInspectResult
	if err := json.Unmarshal([]byte(output), &inspect); err != nil {
		return nil, fmt.Errorf("parse template image inspect output failed: %w", err)
	}
	if len(inspect) == 0 {
		return nil, fmt.Errorf("template image inspect returned no result for %s", imageName)
	}
	return &inspect[0], nil
}

func (sm *SandboxManager) runSandboxCommand(ctx context.Context, args ...string) (string, error) {
	log.Debug("running sandbox command", "args", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, sm.cliBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		log.Error("sandbox command failed", "args", strings.Join(args, " "), "output", trimmed)
		if trimmed == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, trimmed)
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed != "" {
		log.Debug("sandbox command completed", "args", strings.Join(args, " "), "output", trimmed)
	}
	return trimmed, nil
}

func (sm *SandboxManager) ExecInstance(ctx context.Context, pid string, env []string, command string) (string, error) {
	instance, err := sm.GetInstance(pid)
	if err != nil {
		return "", err
	}

	args := []string{"sandbox", "exec"}
	for _, item := range env {
		if strings.TrimSpace(item) == "" {
			continue
		}
		args = append(args, "-e", item)
	}
	args = append(args, instance.ID, "sh", "-lc", command)
	log.Debug("exec into sandbox runtime", "pid", pid, "sandbox_name", instance.ID, "env_count", len(env), "command", command)
	return sm.runSandboxCommand(ctx, args...)
}

func buildSandboxStartCommand() string {
	return "start-vmdocker-agent.sh >/tmp/vmdocker-agent.log 2>&1 &"
}

func shellEscape(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
