package vmdocker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager"
	runtimeSchema "github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	vmdockerSchema "github.com/cryptowizard0/vmdocker/vmdocker/schema"
	"github.com/cryptowizard0/vmdocker/vmdocker/utils"
	"github.com/hymatrix/hymx/common"
	vmmSchema "github.com/hymatrix/hymx/vmm/schema"
	goarSchema "github.com/permadao/goar/schema"
)

var log = common.NewLog("vmdocker")

const defaultRuntimeReadyTimeout = 10 * time.Minute

func Spawn(env vmmSchema.Env) (vm vmmSchema.Vm, err error) {
	vmd, err := New(env, env.Process.Scheduler, env.Process.Tags)
	if err != nil {
		return
	}

	err = vmd.Run(env.Process.Scheduler, []byte(env.Meta.Data), env.Process.Tags)
	if err != nil {
		return
	}
	log.Info("spawn process success", "pid", env.Meta.Pid, "from", env.Meta.AccId)
	return vmd, nil
}

type VmDocker struct {
	pid string
	Env vmmSchema.Env
	// runtime info
	instanceInfo *runtimeSchema.InstanceInfo
	// http client
	client *http.Client
	// close channel to signal container shutdown
	closeChan chan struct{}
}

// todo: add cpu, mem
func New(env vmmSchema.Env, nodeAddr string, tags []goarSchema.Tag) (*VmDocker, error) {
	var err error
	env.Process, err = utils.BuildProcessTags(env.Process, nodeAddr, tags)
	if err != nil {
		log.Error("BuildProcessTags failed", "err", err)
		return nil, err
	}
	v := &VmDocker{
		pid: env.Meta.ItemId,
		Env: env,
		client: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true, // No keep-alive
			},
			Timeout: 10 * 60 * time.Second,
		},
		closeChan: make(chan struct{}),
	}
	return v, nil
}

func (v *VmDocker) Run(cuAddr string, data []byte, tags []goarSchema.Tag) error {
	log.Info("starting vm runtime spawn flow", "pid", v.pid, "owner", v.Env.Meta.AccId, "module_format", v.Env.Module.ModuleFormat)
	ctx := context.Background()

	runtimeManager, err := runtimemanager.GetRuntimeManager()
	if err != nil {
		log.Error("get runtime manager failed", "err", err)
		return err
	}

	runtimeSpec, err := utils.RuntimeSpecFromTags(v.Env.Module.ModuleFormat, v.Env.Module.Tags)
	if err != nil {
		log.Error("build runtime spec failed", "pid", v.pid, "err", err)
		return err
	}
	log.Debug("runtime spec resolved", "pid", v.pid, "backend", runtimeSpec.Backend, "image", runtimeSpec.Image.Name, "sandbox_agent", runtimeSpec.Sandbox.Agent, "sandbox_workspace", runtimeSpec.Sandbox.Workspace)
	if err := ensureModuleImageAvailable(ctx, v.Env.Process.Module, runtimeSpec.Image); err != nil {
		log.Error("prepare module image failed", "pid", v.pid, "module", v.Env.Process.Module, "image", runtimeSpec.Image.Name, "err", err)
		return err
	}
	containerEnv := utils.ContainerEnvFromTags(tags)
	log.Debug("runtime env extracted", "pid", v.pid, "env_count", len(containerEnv), "tag_count", len(tags))
	instanceInfo, err := runtimeManager.CreateInstance(ctx, v.pid, runtimeSpec, containerEnv)
	if err != nil {
		log.Error("create runtime failed", "pid", v.pid, "backend", runtimeSpec.Backend, "image", runtimeSpec.Image.Name, "err", err)
		return err
	}
	v.instanceInfo = instanceInfo
	log.Info("runtime instance created", "pid", v.pid, "port", instanceInfo.Port, "runtime_id", instanceInfo.ID, "backend", instanceInfo.Backend)

	log.Debug("starting runtime instance", "pid", v.pid, "runtime_id", instanceInfo.ID)
	startRuntimeStart := time.Now()
	err = runtimeManager.StartInstance(ctx, v.pid)
	if err != nil {
		log.Error("start runtime failed", "pid", v.pid, "runtime_id", instanceInfo.ID, "backend", instanceInfo.Backend, "err", err)
		return err
	}
	log.Info("runtime instance start requested", "pid", v.pid, "runtime_id", instanceInfo.ID)
	log.Debug("runtime instance start elapsed", "pid", v.pid, "runtime_id", instanceInfo.ID, "elapsed", time.Since(startRuntimeStart))

	readyStart := time.Now()
	err = v.waitForContainerReady(ctx, defaultRuntimeReadyTimeout)
	if err != nil {
		log.Error("runtime readiness check failed", "pid", v.pid, "runtime_id", instanceInfo.ID, "backend", instanceInfo.Backend, "err", err)
		return fmt.Errorf("runtime not ready: %v", err)
	}
	log.Debug("runtime readiness confirmed", "pid", v.pid, "runtime_id", instanceInfo.ID, "elapsed", time.Since(readyStart))

	// create ao process
	log.Debug("sending spawn request to runtime", "pid", v.pid, "cu_addr", cuAddr)
	err = v.spawn(vmdockerSchema.SpawnRequest{
		Pid:    v.pid,
		Owner:  v.Env.Meta.AccId,
		CuAddr: cuAddr,
		Data:   data,
		Tags:   tags,
		Evn:    v.Env,
	})
	if err != nil {
		log.Error("runtime spawn request failed", "pid", v.pid, "runtime_id", instanceInfo.ID, "err", err)
		return err
	}
	log.Info("runtime spawn request completed", "pid", v.pid, "runtime_id", instanceInfo.ID)
	return nil
}

func (v *VmDocker) Apply(from string, meta vmmSchema.Meta) vmmSchema.Result {
	res, err := v.apply(vmdockerSchema.ApplyRequest{
		From:   from,
		Meta:   meta,
		Params: meta.Params,
	})

	if err != nil {
		return vmmSchema.Result{Error: err}
	}
	if res == nil {
		return vmmSchema.Result{Error: fmt.Errorf("apply returned nil result")}
	}
	return *res
}

func (v *VmDocker) Checkpoint() (string, error) {
	return "", runtimeSchema.ErrNotSupported
}

func (v *VmDocker) Restore(snapshot string) error {
	_ = snapshot
	return runtimeSchema.ErrNotSupported
}

func (v *VmDocker) Close() error {
	// Signal waitForContainerReady to exit immediately
	select {
	case v.closeChan <- struct{}{}:
	default:
		// Channel might be full or closed, ignore
	}

	runtimeManager, err := runtimemanager.GetRuntimeManager()
	if err != nil {
		log.Error("get runtime manager failed", "err", err)
		return err
	}
	log.Info("closing vm runtime", "pid", v.pid, "runtime_id", func() string {
		if v.instanceInfo == nil {
			return ""
		}
		return v.instanceInfo.ID
	}())
	return runtimeManager.RemoveInstance(context.Background(), v.pid)
}

// waitForContainerReady waits for the runtime to be ready by checking health endpoint.
func (v *VmDocker) waitForContainerReady(ctx context.Context, timeout time.Duration) error {
	if v.instanceInfo == nil {
		return fmt.Errorf("instanceInfo is nil")
	}

	startTime := time.Now()
	log.Debug("waiting for runtime to be ready", "pid", v.pid, "port", v.instanceInfo.Port)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			elapsedTime := time.Since(startTime)
			log.Debug("runtime health check timeout", "pid", v.pid, "elapsed_time", elapsedTime)
			return fmt.Errorf("timeout waiting for runtime to be ready")
		case <-v.closeChan:
			elapsedTime := time.Since(startTime)
			log.Debug("runtime closed during health check", "pid", v.pid, "elapsed_time", elapsedTime)
			return fmt.Errorf("runtime was closed")
		case <-ticker.C:
			statusCode, err := v.healthStatusCode(ctx)
			if err != nil {
				log.Debug("runtime health check failed", "pid", v.pid, "err", err)
				continue
			}
			log.Debug("runtime health check returned", "pid", v.pid, "status_code", statusCode)

			if statusCode == http.StatusOK {
				elapsedTime := time.Since(startTime)
				log.Debug("runtime ready", "pid", v.pid, "elapsed_time", elapsedTime)
				return nil
			}
		}
	}
}

func (v *VmDocker) spawn(msg vmdockerSchema.SpawnRequest) error {
	log.Debug("spawn process", "pid", v.pid, "owner", msg.Owner, "tag_count", len(msg.Tags))

	// safe check
	if v.instanceInfo == nil {
		return fmt.Errorf("instanceInfo is nil, pid: %s", v.pid)
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal request failed: %v", err)
	}

	statusCode, body, err := v.callRuntimeEndpoint(context.Background(), "/vmm/spawn", jsonData)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %d: %s", statusCode, string(body))
	}
	log.Debug("spawn request accepted", "pid", v.pid, "status_code", statusCode, "body", string(body))

	return nil
}

func (v *VmDocker) apply(msg vmdockerSchema.ApplyRequest) (outbox *vmmSchema.Result, err error) {
	// safe check
	if v.instanceInfo == nil {
		err = fmt.Errorf("instanceInfo is nil, pid: %s", v.pid)
		return
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		err = fmt.Errorf("marshal request failed: %v", err)
		return
	}
	log.Debug("===> apply request", "pid", v.pid, "msg", string(jsonData))

	statusCode, body, err := v.callRuntimeEndpoint(context.Background(), "/vmm/apply", jsonData)
	if err != nil {
		return
	}
	if statusCode != http.StatusOK {
		err = fmt.Errorf("request failed with status %d: %s", statusCode, string(body))
		return
	}

	var resOutbox vmdockerSchema.OutboxResponse
	err = json.Unmarshal(body, &resOutbox)
	if err != nil {
		log.Error("unmarshal response failed", "err", err)
		return
	}
	log.Debug("===> apply success", "result", resOutbox)

	outbox = &vmmSchema.Result{}
	if err = json.Unmarshal([]byte(resOutbox.Result), outbox); err != nil {
		log.Error("unmarshal response outbox failed", "err", err)
	}

	return
}

func (v *VmDocker) healthStatusCode(ctx context.Context) (int, error) {
	statusCode, _, err := v.callRuntimeEndpoint(ctx, "/vmm/health", nil)
	return statusCode, err
}

func (v *VmDocker) callRuntimeEndpoint(ctx context.Context, path string, payload []byte) (int, []byte, error) {
	if v.instanceInfo == nil {
		return 0, nil, fmt.Errorf("instanceInfo is nil, pid: %s", v.pid)
	}

	if v.instanceInfo.Backend == runtimeSchema.BackendSandbox {
		return v.callSandboxRuntimeEndpoint(ctx, path, payload)
	}
	return v.callDockerRuntimeEndpoint(path, payload)
}

func (v *VmDocker) callDockerRuntimeEndpoint(path string, payload []byte) (int, []byte, error) {
	url := fmt.Sprintf("http://%s:%d%s", runtimeSchema.AllowHost, v.instanceInfo.Port, path)
	log.Debug("calling docker runtime endpoint", "pid", v.pid, "path", path, "url", url)

	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, fmt.Errorf("create request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")

	resp, err := v.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("send request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("read response body failed: %v", err)
	}
	log.Debug("docker runtime endpoint returned", "pid", v.pid, "path", path, "status_code", resp.StatusCode, "body", string(body))
	return resp.StatusCode, body, nil
}

func (v *VmDocker) callSandboxRuntimeEndpoint(ctx context.Context, path string, payload []byte) (int, []byte, error) {
	runtimeManager, err := runtimemanager.GetRuntimeManager()
	if err != nil {
		return 0, nil, fmt.Errorf("get runtime manager failed: %v", err)
	}

	command := ""
	cleanup := func() {}
	if len(payload) > 0 {
		payloadPath, err := v.writeSandboxPayloadFile(payload)
		if err != nil {
			return 0, nil, err
		}
		cleanup = func() {
			_ = os.Remove(payloadPath)
		}
		command = buildSandboxCurlCommandFromFile(path, payloadPath)
	} else {
		command = buildSandboxCurlCommand(path, payload)
	}
	defer cleanup()
	log.Debug("calling sandbox runtime endpoint", "pid", v.pid, "path", path, "command", command)
	output, err := runtimeManager.ExecInstance(ctx, v.pid, nil, command)
	if err != nil {
		return 0, nil, fmt.Errorf("sandbox exec failed: %v", err)
	}

	statusCode, body, err := parseSandboxCurlOutput(output)
	if err != nil {
		return 0, nil, err
	}
	log.Debug("sandbox runtime endpoint returned", "pid", v.pid, "path", path, "status_code", statusCode, "body", string(body))
	return statusCode, body, nil
}

func (v *VmDocker) writeSandboxPayloadFile(payload []byte) (string, error) {
	if v.instanceInfo == nil {
		return "", fmt.Errorf("instanceInfo is nil, pid: %s", v.pid)
	}
	if strings.TrimSpace(v.instanceInfo.Workspace) == "" {
		return "", fmt.Errorf("sandbox workspace is empty, pid: %s", v.pid)
	}
	payloadDir := filepath.Join(v.instanceInfo.Workspace, ".tmp")
	if err := os.MkdirAll(payloadDir, 0o755); err != nil {
		return "", fmt.Errorf("create sandbox payload dir failed: %w", err)
	}
	payloadPath := filepath.Join(payloadDir, fmt.Sprintf("runtime-request-%d.json", time.Now().UnixNano()))
	if err := os.WriteFile(payloadPath, payload, 0o600); err != nil {
		return "", fmt.Errorf("write sandbox payload failed: %w", err)
	}
	return payloadPath, nil
}

func buildSandboxCurlCommand(path string, payload []byte) string {
	url := "http://127.0.0.1:8080" + path
	body := ""
	if payload != nil {
		body = string(payload)
	}
	return fmt.Sprintf("curl -sS -X POST -H %s --data-raw %s %s -w '\\n__STATUS__:%%{http_code}'",
		shellEscape("Content-Type: application/json"),
		shellEscape(body),
		shellEscape(url),
	)
}

func buildSandboxCurlCommandFromFile(path, payloadPath string) string {
	url := "http://127.0.0.1:8080" + path
	return fmt.Sprintf("curl -sS -X POST -H %s --data-binary @%s %s -w '\\n__STATUS__:%%{http_code}'",
		shellEscape("Content-Type: application/json"),
		shellEscape(payloadPath),
		shellEscape(url),
	)
}

func parseSandboxCurlOutput(output string) (int, []byte, error) {
	idx := strings.LastIndex(output, "\n__STATUS__:")
	if idx == -1 {
		return 0, nil, fmt.Errorf("sandbox response missing status marker: %s", output)
	}
	statusText := strings.TrimSpace(output[idx+len("\n__STATUS__:"):])
	statusCode, err := strconv.Atoi(statusText)
	if err != nil {
		return 0, nil, fmt.Errorf("parse sandbox status failed: %w", err)
	}
	body := []byte(output[:idx])
	return statusCode, body, nil
}

func shellEscape(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
