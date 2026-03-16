package utils

import (
	"encoding/base64"
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

func TestRuntimeSpecFromTags_BuildModeUsesEmbeddedContext(t *testing.T) {
	spec, err := RuntimeSpecFromTags(vmdockerSchema.ModuleFormat, []goarSchema.Tag{
		{Name: vmdockerSchema.RuntimeBackendTag, Value: vmdockerSchema.BackendSandbox},
		{Name: vmdockerSchema.BuildTypeTag, Value: "dockerfile"},
		{Name: vmdockerSchema.BuildDockerfileTag, Value: base64.StdEncoding.EncodeToString([]byte("FROM scratch"))},
		{Name: vmdockerSchema.BuildContextTag, Value: "encoded-context"},
		{Name: vmdockerSchema.BuildTagTag, Value: "vmdocker-openclaw:test"},
		{Name: vmdockerSchema.BuildArgTagPrefix + "FOO", Value: "bar"},
	})
	require.NoError(t, err)
	require.NotNil(t, spec.Image.Build)
	require.Equal(t, "vmdocker-openclaw:test", spec.Image.Name)
	require.Equal(t, "FROM scratch", spec.Image.Build.Dockerfile)
	require.Equal(t, "encoded-context", spec.Image.Build.ContextArchive)
	require.Equal(t, "bar", spec.Image.Build.Args["FOO"])
}
