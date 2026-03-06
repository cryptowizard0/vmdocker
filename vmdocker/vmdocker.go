package vmdocker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	log.Debug("init ao process", "pid", v.pid)
	ctx := context.Background()

	runtimeManager, err := runtimemanager.GetRuntimeManager()
	if err != nil {
		log.Error("get runtime manager failed", "err", err)
		return err
	}

	runtimeSpec, err := utils.RuntimeSpecFromTags(v.Env.Module.ModuleFormat, v.Env.Module.Tags)
	if err != nil {
		return err
	}
	containerEnv := utils.ContainerEnvFromTags(tags)
	instanceInfo, err := runtimeManager.CreateInstance(ctx, v.pid, runtimeSpec, containerEnv)
	if err != nil {
		return err
	}
	v.instanceInfo = instanceInfo
	log.Debug("create runtime success", "pid", v.pid, "port", instanceInfo.Port, "runtimeId", instanceInfo.ID, "backend", instanceInfo.Backend)

	err = runtimeManager.StartInstance(ctx, v.pid)
	if err != nil {
		return err
	}

	err = v.waitForContainerReady(ctx, 120*time.Second)
	if err != nil {
		return fmt.Errorf("runtime not ready: %v", err)
	}

	// create ao process
	return v.spawn(vmdockerSchema.SpawnRequest{
		Pid:    v.pid,
		Owner:  v.Env.Meta.AccId,
		CuAddr: cuAddr,
		Data:   data,
		Tags:   tags,
		Evn:    v.Env,
	})
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
	return runtimeManager.RemoveInstance(context.Background(), v.pid)
}

// waitForContainerReady waits for the runtime to be ready by checking health endpoint.
func (v *VmDocker) waitForContainerReady(ctx context.Context, timeout time.Duration) error {
	if v.instanceInfo == nil {
		return fmt.Errorf("instanceInfo is nil")
	}

	startTime := time.Now()
	log.Debug("waiting for runtime to be ready", "pid", v.pid, "port", v.instanceInfo.Port)

	url := fmt.Sprintf("http://%s:%d/vmm/health", runtimeSchema.AllowHost, v.instanceInfo.Port)
	client := &http.Client{Timeout: 2 * time.Second}

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
			req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
			if err != nil {
				continue
			}

			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				elapsedTime := time.Since(startTime)
				log.Debug("runtime ready", "pid", v.pid, "elapsed_time", elapsedTime)
				return nil
			}
		}
	}
}

func (v *VmDocker) spawn(msg vmdockerSchema.SpawnRequest) error {
	log.Debug("spawn process", "pid", v.pid)

	// safe check
	if v.instanceInfo == nil {
		return fmt.Errorf("instanceInfo is nil, pid: %s", v.pid)
	}

	// POST /vmm/spawn
	url := fmt.Sprintf("http://%s:%d/vmm/spawn", runtimeSchema.AllowHost, v.instanceInfo.Port)

	// Convert request to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal request failed: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")

	// Send request
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (v *VmDocker) apply(msg vmdockerSchema.ApplyRequest) (outbox *vmmSchema.Result, err error) {
	// safe check
	if v.instanceInfo == nil {
		err = fmt.Errorf("instanceInfo is nil, pid: %s", v.pid)
		return
	}

	// POST /vmm/apply
	url := fmt.Sprintf("http://%s:%d/vmm/apply", runtimeSchema.AllowHost, v.instanceInfo.Port)
	// Convert request to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		err = fmt.Errorf("marshal request failed: %v", err)
		return
	}
	log.Debug("===> apply request", "pid", v.pid, "msg", string(jsonData))

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		err = fmt.Errorf("create request failed: %v", err)
		return
	}
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	// Send request
	resp, err := v.client.Do(req)
	if err != nil {
		err = fmt.Errorf("send request failed: %v", err)
		return
	}
	defer resp.Body.Close()
	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
		return
	}
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("read response body failed: %v", err)
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
