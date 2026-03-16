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

const (
	BuildTypeTag       = "Build-Type"
	BuildDockerfileTag = "Build-Dockerfile-Content"
	BuildContextURLTag = "Build-Context-URL"
	BuildContextTag    = "Build-Context-Archive"
	BuildTagTag        = "Build-Tag"
	BuildArgTagPrefix  = "Build-Arg-"
)

// BuildSpec holds the information needed to build a Docker image at spawn time.
// When non-nil, the image is built locally instead of pulled from a registry.
type BuildSpec struct {
	Dockerfile     string            // decoded Dockerfile content
	ContextURL     string            // optional URL for build context (tarball or git URL)
	ContextArchive string            // optional base64-encoded tar.gz build context
	Tag            string            // local image tag to apply after build
	Args           map[string]string // --build-arg values
}

// ImageInfo contains image name and verification information
type ImageInfo struct {
	Name  string     // Docker image name (or Build-Tag in build mode)
	SHA   string     // Image SHA256 digest for verification (pull mode only)
	Build *BuildSpec // non-nil triggers build mode; nil uses pull mode
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
