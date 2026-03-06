package runtimemanager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
)

const (
	defaultSandboxAgent    = "openclaw"
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

	if err := sm.ensureSandboxCLI(ctx); err != nil {
		return nil, err
	}

	port, err := sm.portAllocator.Allocate()
	if err != nil {
		return nil, err
	}

	sandboxName := runtimeSpec.Sandbox.Name
	if sandboxName == "" {
		sandboxName = ContainerNamePrefix + pid
	}

	workspace := runtimeSpec.Sandbox.Workspace
	if workspace == "" {
		workspace, err = os.Getwd()
		if err != nil {
			sm.portAllocator.Release(port)
			return nil, err
		}
	}

	agent := runtimeSpec.Sandbox.Agent
	if agent == "" {
		agent = defaultSandboxAgent
	}

	args := []string{"sandbox", "create", agent, workspace, "--name", sandboxName}
	if runtimeSpec.Sandbox.Network != "" {
		args = append(args, "--network", runtimeSpec.Sandbox.Network)
	}

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
	return instance, nil
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
		command = buildSandboxDockerRunCommand(pid, instance.Port, runtimeSpec.Image.Name, runtimeEnv)
	}

	if _, err := sm.runSandboxCommand(ctx, "sandbox", "exec", instance.ID, "sh", "-lc", command); err != nil {
		return err
	}

	instance.Status = "running"
	return nil
}

func (sm *SandboxManager) StopInstance(ctx context.Context, pid string) error {
	instance, err := sm.GetInstance(pid)
	if err != nil {
		return err
	}

	stopCommand := "docker rm -f " + shellEscape(runtimeContainerPrefix+pid)
	if _, err := sm.runSandboxCommand(ctx, "sandbox", "exec", instance.ID, "sh", "-lc", stopCommand); err != nil {
		return err
	}

	instance.Status = "stopped"
	return nil
}

func (sm *SandboxManager) RemoveInstance(ctx context.Context, pid string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	instance, exists := sm.instances[pid]
	if !exists {
		return fmt.Errorf("instance not found: %s", pid)
	}

	if _, err := sm.runSandboxCommand(ctx, "sandbox", "rm", "-f", instance.ID); err != nil {
		return err
	}

	sm.portAllocator.Release(instance.Port)
	delete(sm.instances, pid)
	delete(sm.launchSpecs, pid)
	return nil
}

func (sm *SandboxManager) Checkpoint(context.Context, string, string) (string, error) {
	return "", schema.ErrNotSupported
}

func (sm *SandboxManager) Restore(context.Context, string, string, string) error {
	return schema.ErrNotSupported
}

func (sm *SandboxManager) ensureSandboxCLI(ctx context.Context) error {
	output, err := exec.CommandContext(ctx, sm.cliBin, "--help").CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker CLI is not available: %w", err)
	}
	if !strings.Contains(string(output), "sandbox") {
		return fmt.Errorf("docker sandbox CLI is not available on this machine")
	}
	return nil
}

func (sm *SandboxManager) runSandboxCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, sm.cliBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, trimmed)
	}
	return strings.TrimSpace(string(output)), nil
}

func buildSandboxDockerRunCommand(pid string, port int, imageName string, runtimeEnv []string) string {
	args := []string{
		"docker", "run", "-d",
		"--name", runtimeContainerPrefix + pid,
		"-p", strconv.Itoa(port) + ":8080",
	}

	for _, env := range runtimeEnv {
		args = append(args, "-e", env)
	}

	args = append(args, imageName)
	for i, arg := range args {
		args[i] = shellEscape(arg)
	}
	return strings.Join(args, " ")
}

func shellEscape(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
