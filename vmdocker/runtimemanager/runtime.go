package runtimemanager

import (
	"context"
	"sync"

	"github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	"github.com/hymatrix/hymx/common"
)

var log = common.NewLog("vmdocker-runtime")

type IRuntimeManager interface {
	CreateInstance(ctx context.Context, pid string, runtimeSpec schema.RuntimeSpec, runtimeEnv []string) (*schema.InstanceInfo, error)
	GetInstance(pid string) (*schema.InstanceInfo, error)
	RemoveInstance(ctx context.Context, pid string) error
	StartInstance(ctx context.Context, pid string) error
	StopInstance(ctx context.Context, pid string) error
	ExecInstance(ctx context.Context, pid string, env []string, command string) (string, error)
	Checkpoint(ctx context.Context, pid, checkpointName string) (string, error)
	Restore(ctx context.Context, pid, checkpointName, snapshot string) error
}

var (
	runtimeOnce     sync.Once
	runtimeInstance IRuntimeManager
)

func GetRuntimeManager() (IRuntimeManager, error) {
	var initErr error
	runtimeOnce.Do(func() {
		runtimeInstance, initErr = newSandboxManager()
	})
	if initErr != nil {
		return nil, initErr
	}
	return runtimeInstance, nil
}

func GetDockerManager() (IRuntimeManager, error) {
	return newDockerManager()
}
