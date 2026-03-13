package utils

import (
	"testing"

	vmdockerSchema "github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	goarSchema "github.com/permadao/goar/schema"
	"github.com/stretchr/testify/require"
)

func TestRuntimeSpecFromTags_DefaultsToSandbox(t *testing.T) {
	spec, err := RuntimeSpecFromTags(vmdockerSchema.ModuleFormat, []goarSchema.Tag{
		{Name: "Image-Name", Value: "chriswebber/docker-openclaw-sandbox:v0.0.1"},
		{Name: "Image-ID", Value: "sha256:sandbox-template"},
	})
	require.NoError(t, err)
	require.Equal(t, vmdockerSchema.BackendSandbox, spec.Backend)
	require.Equal(t, "shell", spec.Sandbox.Agent)
	require.Equal(t, "", spec.Sandbox.Workspace)
	require.Equal(t, "", spec.Sandbox.Network)
}

func TestRuntimeSpecFromTags_DockerRequiresImageID(t *testing.T) {
	_, err := RuntimeSpecFromTags(vmdockerSchema.ModuleFormat, []goarSchema.Tag{
		{Name: vmdockerSchema.RuntimeBackendTag, Value: vmdockerSchema.BackendDocker},
		{Name: "Image-Name", Value: "chriswebber/docker-openclaw:v0.0.4"},
	})
	require.EqualError(t, err, "Image-ID is empty")
}

func TestRuntimeSpecFromTags_SandboxRequiresImageID(t *testing.T) {
	_, err := RuntimeSpecFromTags(vmdockerSchema.ModuleFormat, []goarSchema.Tag{
		{Name: vmdockerSchema.RuntimeBackendTag, Value: vmdockerSchema.BackendSandbox},
		{Name: "Image-Name", Value: "chriswebber/docker-openclaw-sandbox:v0.0.1"},
	})
	require.EqualError(t, err, "Image-ID is empty")
}
