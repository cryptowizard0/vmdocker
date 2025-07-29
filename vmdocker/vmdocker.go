package vmdocker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/schema"
	"github.com/cryptowizard0/vmdocker/vmdocker/utils"
	"github.com/hymatrix/hymx/common"
	hymxUtils "github.com/hymatrix/hymx/utils"
	vmmSchema "github.com/hymatrix/hymx/vmm/schema"
	goarSchema "github.com/permadao/goar/schema"
)

var log = common.NewLog("vmdocker")

func SpawnVmDocker(env vmmSchema.Env) (vm vmmSchema.Vm, err error) {
	vmd, err := New(env, env.Process.Scheduler, env.Process.Tags)
	if err != nil {
		return
	}

	err = vmd.Run(env.Process.Scheduler, []byte(env.Meta.Data), env.Process.Tags)
	if err != nil {
		return
	}
	log.Info("spawn process success", "pid", env.Id, "from", env.AccId)
	return vmd, nil
}

type VmDocker struct {
	//  info
	pid string
	Env vmmSchema.Env
	// container info
	containerInfo *schema.ContainerInfo
	// http client
	client *http.Client
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
		pid: env.Id,
		Env: env,
		client: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true, // No keep-alive
			},
			Timeout: 10 * 60 * time.Second,
		},
	}
	return v, nil
}

func NewFromCheckpoint(env vmmSchema.Env, checkpointName string, tags []goarSchema.Tag) (*VmDocker, error) {
	v := &VmDocker{
		pid: env.Id,
		Env: env,
		client: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true, // No keep-alive
			},
			Timeout: 60 * time.Second,
		},
	}

	err := v.initProcessFromCheckpoint(checkpointName)
	if err != nil {
		log.Error("init process from checkpoint failed", "err", err)
		return nil, err
	}
	log.Debug("create process from checkpoint success", "pid", v.pid)
	return v, nil
}

func (v *VmDocker) initProcessFromCheckpoint(checkpointName string) error {
	// log.Debug("init ao process from checkpoint", "pid", v.pid)
	// return v.Restore([]byte(checkpointName))
	return nil
}

func (v *VmDocker) Run(cuAddr string, data []byte, tags []goarSchema.Tag) error {
	log.Debug("init ao process", "pid", v.pid)
	ctx := context.Background()

	dm, err := GetDockerManager()
	if err != nil {
		log.Error("get docker manager failed", "err", err)
		return err
	}

	err = utils.CheckModuleFormat(v.Env.Module.ModuleFormat, v.Env.Module.Tags)
	if err != nil {
		return err
	}
	imageInfo := schema.ImageInfo{
		Name: hymxUtils.GetTagsValueByDefault("Image-Name", v.Env.Module.Tags, ""),
		SHA:  hymxUtils.GetTagsValueByDefault("Image-ID", v.Env.Module.Tags, ""),
	}
	// start docker container
	containerInfo, err := dm.CreateContainer(ctx, v.pid, imageInfo)
	if err != nil {
		return err
	}
	v.containerInfo = containerInfo
	log.Debug("create container success", "pid", v.pid, "port", containerInfo.Port, "containerId", containerInfo.ID)

	// start container
	err = dm.StartContainer(ctx, v.pid)
	if err != nil {
		return err
	}
	time.Sleep(10 * time.Second) // Wait for container to fully start

	// create ao process
	return v.spawn(schema.SpawnRequest{
		Pid:    v.pid,
		Owner:  v.Env.AccId,
		CuAddr: cuAddr,
		Data:   data,
		Tags:   tags,
		Evn:    v.Env,
	})
}

func (v *VmDocker) Apply(from string, meta vmmSchema.Meta) (*vmmSchema.Result, error) { // TODO implement me
	return v.apply(schema.ApplyRequest{
		From:   from,
		Meta:   meta,
		Params: meta.Params,
	})
}

func (v *VmDocker) spawn(msg schema.SpawnRequest) error {
	log.Debug("spawn ao process", "pid", v.pid)

	// safe check
	if v.containerInfo == nil {
		return fmt.Errorf("containerInfo is nil, pid: %s", v.pid)
	}

	// POST /vmm/spawn
	url := fmt.Sprintf("http://%s:%d/vmm/spawn", schema.AllowHost, v.containerInfo.Port)

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

func (v *VmDocker) apply(msg schema.ApplyRequest) (outbox *vmmSchema.Result, err error) {
	// safe check
	if v.containerInfo == nil {
		err = fmt.Errorf("containerInfo is nil, pid: %s", v.pid)
		return
	}

	// POST /vmm/apply
	url := fmt.Sprintf("http://%s:%d/vmm/apply", schema.AllowHost, v.containerInfo.Port)
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

	var resOutbox schema.OutboxResponse
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

func (v *VmDocker) Checkpoint() (string, error) {
	dm, err := GetDockerManager()
	if err != nil {
		log.Error("get docker manager failed", "err", err)
		return "", err
	}
	checkpointName := fmt.Sprintf("checkpoint-%s-%d", v.pid, v.Env.Nonce)
	data, err := dm.Checkpoint(context.Background(), v.pid, checkpointName)
	if err != nil {
		log.Error("create checkpoint failed", "err", err, "pid", v.pid)
		return "", err
	}
	log.Info("create checkpoint success", "pid", v.pid, "checkpointName", checkpointName)
	return data, nil
}

func (v *VmDocker) Restore(snapshot string) error {
	checkpointName := fmt.Sprintf("checkpoint-%s-%d", v.pid, time.Now().Unix())
	log.Debug("restore process", "pid", v.pid, "name", checkpointName)

	// stop container first
	dm, err := GetDockerManager()
	if err != nil {
		log.Error("get docker manager failed", "err", err)
		return err
	}
	err = dm.StopContainer(context.Background(), v.pid)
	if err != nil {
		log.Error("stop container failed", "err", err)
		return err
	}

	// start container from checkpoint
	err = dm.Restore(context.Background(), v.pid, checkpointName, snapshot)
	if err != nil {
		log.Error("restore failed", "err", err, "pid", v.pid)
		return err
	}
	return nil
}

func (v *VmDocker) Close() error {
	dm, err := GetDockerManager()
	if err != nil {
		log.Error("get docker manager failed", "err", err)
		return err
	}
	return dm.RemoveContainer(context.Background(), v.pid)
}
