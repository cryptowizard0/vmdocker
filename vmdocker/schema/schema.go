package schema

import (
	"context"
	"os"
	"time"

	vmmSchema "github.com/hymatrix/hymx/vmm/schema"
	goarSchema "github.com/permadao/goar/schema"
)

const (
	ModuleFormat = "web.vmdocker-golua-ao.v0.0.1"
	// example: web.vmdocker-golua-ao.v0.0.1
	// example: web.vmdocker-ollama.v0.0.1
)

// ImageInfo contains image name and verification information
type ImageInfo struct {
	Name string // Docker image name
	SHA  string // Image SHA256 digest for verification
}

var (
	DockerVersion = "1.47"
	ExprotPort    = "8080/tcp"
	AllowHost     = "127.0.0.1"             // Only host machine can access the container
	MaxMem        = 12 * 1024 * 1024 * 1024 // max 12GB memory
	CheckpointDir = "checkpoints"

	// use mount to share models
	UseMount    = false
	MountSource = os.ExpandEnv("$HOME/.ollama/models")
	MountTarget = "/app/models"
)

type ContainerInfo struct {
	ID       string
	Name     string
	Port     int
	Status   string
	CreateAt time.Time
}

// IDockerManager defines the interface for docker operations
type IDockerManager interface {
	// CreateContainer creates a new container with the given process id
	CreateContainer(ctx context.Context, pid string, imageInfo ImageInfo) (*ContainerInfo, error)

	// GetContainer returns container info by process id
	GetContainer(pid string) (*ContainerInfo, error)

	// RemoveContainer removes a container by process id
	RemoveContainer(ctx context.Context, pid string) error

	// StartContainer starts a container by process id
	StartContainer(ctx context.Context, pid string) error

	// StopContainer stops a container by process id
	StopContainer(ctx context.Context, pid string) error

	// Checkpoint creates a checkpoint for a container
	Checkpoint(ctx context.Context, pid, checkpointName string) (string, error)

	// Restore restores a container from checkpoint
	Restore(ctx context.Context, pid, checkpointName, snapshot string) error
}

type SpawnRequest struct {
	Pid    string           `json:"pid"`
	Owner  string           `json:"owner"`
	CuAddr string           `json:"cu_addr"`
	Data   []byte           `json:"data"`
	Tags   []goarSchema.Tag `json:"tags"`
	Evn    vmmSchema.Env    `json:"env"`
}

type ApplyRequest struct {
	Meta   vmmSchema.Meta    `json:"meta"`
	From   string            `json:"from"`
	Params map[string]string `json:"params"`
}

type OutboxResponse struct {
	Result string `json:"result"`
	Status string `json:"status"`
}
