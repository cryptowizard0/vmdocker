package utils

import (
	"testing"

	vmdockerSchema "github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	goarSchema "github.com/permadao/goar/schema"
	"github.com/stretchr/testify/require"
)

func TestRuntimeSpecFromTags_DefaultsToSandbox(t *testing.T) {
	spec, err := RuntimeSpecFromTags(vmdockerSchema.ModuleFormat, []goarSchema.Tag{
		{Name: vmdockerSchema.ImageNameTag, Value: "chriswebber/docker-openclaw-sandbox:v0.0.1"},
		{Name: vmdockerSchema.ImageIDTag, Value: "sha256:sandbox-template"},
		{Name: vmdockerSchema.ImageSourceTag, Value: vmdockerSchema.ImageSourceModuleData},
		{Name: vmdockerSchema.ImageArchiveTag, Value: vmdockerSchema.ImageArchiveDockerSaveGZ},
	})
	require.NoError(t, err)
	require.Equal(t, vmdockerSchema.BackendSandbox, spec.Backend)
	require.Equal(t, "shell", spec.Sandbox.Agent)
	require.Equal(t, "", spec.Sandbox.Workspace)
	require.Equal(t, "", spec.Sandbox.Network)
	require.Equal(t, vmdockerSchema.ImageSourceModuleData, spec.Image.Source)
	require.Equal(t, vmdockerSchema.ImageArchiveDockerSaveGZ, spec.Image.ArchiveFormat)
}

func TestRuntimeSpecFromTags_DockerRequiresImageID(t *testing.T) {
	_, err := RuntimeSpecFromTags(vmdockerSchema.ModuleFormat, []goarSchema.Tag{
		{Name: vmdockerSchema.RuntimeBackendTag, Value: vmdockerSchema.BackendDocker},
		{Name: vmdockerSchema.ImageNameTag, Value: "chriswebber/docker-openclaw:v0.0.4"},
		{Name: vmdockerSchema.ImageSourceTag, Value: vmdockerSchema.ImageSourceModuleData},
		{Name: vmdockerSchema.ImageArchiveTag, Value: vmdockerSchema.ImageArchiveDockerSaveGZ},
	})
	require.EqualError(t, err, vmdockerSchema.ImageIDTag+" is empty")
}

func TestRuntimeSpecFromTags_SandboxRequiresImageID(t *testing.T) {
	_, err := RuntimeSpecFromTags(vmdockerSchema.ModuleFormat, []goarSchema.Tag{
		{Name: vmdockerSchema.RuntimeBackendTag, Value: vmdockerSchema.BackendSandbox},
		{Name: vmdockerSchema.ImageNameTag, Value: "chriswebber/docker-openclaw-sandbox:v0.0.1"},
		{Name: vmdockerSchema.ImageSourceTag, Value: vmdockerSchema.ImageSourceModuleData},
		{Name: vmdockerSchema.ImageArchiveTag, Value: vmdockerSchema.ImageArchiveDockerSaveGZ},
	})
	require.EqualError(t, err, vmdockerSchema.ImageIDTag+" is empty")
}

func TestRuntimeSpecFromTags_RejectsLegacyBuildModules(t *testing.T) {
	_, err := RuntimeSpecFromTags(vmdockerSchema.ModuleFormat, []goarSchema.Tag{
		{Name: vmdockerSchema.RuntimeBackendTag, Value: vmdockerSchema.BackendSandbox},
		{Name: "Build-Type", Value: "dockerfile"},
		{Name: vmdockerSchema.ImageNameTag, Value: "chriswebber/docker-openclaw-sandbox:v0.0.1"},
		{Name: vmdockerSchema.ImageIDTag, Value: "sha256:sandbox-template"},
	})
	require.EqualError(t, err, "Build-Type modules are no longer supported")
}

func TestRuntimeSpecFromTags_RequiresModuleDataSource(t *testing.T) {
	_, err := RuntimeSpecFromTags(vmdockerSchema.ModuleFormat, []goarSchema.Tag{
		{Name: vmdockerSchema.ImageNameTag, Value: "chriswebber/docker-openclaw-sandbox:v0.0.1"},
		{Name: vmdockerSchema.ImageIDTag, Value: "sha256:sandbox-template"},
	})
	require.EqualError(t, err, "Image-Source must be module-data")
}
