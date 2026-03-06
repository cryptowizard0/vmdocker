package schema

import (
	"errors"
	"time"
)

const (
	ModuleFormat = "hymx.vmdocker.v0.0.1"
)

var ErrNotSupported = errors.New("not supported in stateless runtime mode")

const (
	BackendDocker  = "docker"
	BackendSandbox = "sandbox"
)

const (
	RuntimeBackendTag   = "Runtime-Backend"
	SandboxAgentTag     = "Sandbox-Agent"
	SandboxWorkspaceTag = "Sandbox-Workspace"
	SandboxNetworkTag   = "Sandbox-Network"
	SandboxNameTag      = "Sandbox-Name"
	SandboxCommandTag   = "Sandbox-Command"
)

// ImageInfo contains image name and verification information
type ImageInfo struct {
	Name string // Docker image name
	SHA  string // Image SHA256 digest for verification
}

type SandboxSpec struct {
	Agent     string
	Workspace string
	Network   string
	Name      string
	Command   string
}

type RuntimeSpec struct {
	Backend string
	Image   ImageInfo
	Sandbox SandboxSpec
}

type InstanceInfo struct {
	ID       string
	Name     string
	Port     int
	Status   string
	CreateAt time.Time
	Backend  string
	Agent    string
}
