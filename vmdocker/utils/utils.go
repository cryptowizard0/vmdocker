package utils

import (
	"errors"
	"os"

	"github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	hySchema "github.com/hymatrix/hymx/schema"
	"github.com/hymatrix/hymx/utils"
	goarSchema "github.com/permadao/goar/schema"
)

// todo: this function for test env, should remove when real data is ok
func BuildProcessTags(process hySchema.Process, nodeAddr string, tags []goarSchema.Tag) (hySchema.Process, error) {
	processTags, err := utils.ProcessToTags(process)
	if err != nil {
		return process, err
	}

	processTags = append(processTags, []goarSchema.Tag{
		{
			Name:  "Authority",
			Value: nodeAddr,
		},
		{
			Name:  "App-Name",
			Value: "hymatrix",
		},
		{
			Name:  "Name",
			Value: utils.GetTagsValueByDefault("Name", tags, "default"),
		},
		{
			Name:  "Content-Type",
			Value: "text/plain",
		},
		{
			Name:  "Reference",
			Value: utils.GetTagsValueByDefault("Reference", tags, "0"),
		},
	}...)

	fromProcess := utils.GetTagsValueByDefault("From-Process", tags, "")
	if fromProcess != "" {
		processTags = append(processTags, goarSchema.Tag{
			Name:  "From-Process",
			Value: fromProcess,
		})
	}
	process.Tags = processTags
	return process, nil
}

func RuntimeSpecFromTags(moduleFormat string, tags []goarSchema.Tag) (schema.RuntimeSpec, error) {
	if moduleFormat != schema.ModuleFormat {
		return schema.RuntimeSpec{}, errors.New("module format is not " + schema.ModuleFormat)
	}

	imageName := utils.GetTagsValueByDefault("Image-Name", tags, "")
	if imageName == "" {
		return schema.RuntimeSpec{}, errors.New("Image-Name is empty")
	}

	backend := utils.GetTagsValueByDefault(schema.RuntimeBackendTag, tags, schema.BackendSandbox)
	spec := schema.RuntimeSpec{
		Backend: backend,
		Image: schema.ImageInfo{
			Name: imageName,
			SHA:  utils.GetTagsValueByDefault("Image-ID", tags, ""),
		},
		Sandbox: schema.SandboxSpec{
			Agent:     utils.GetTagsValueByDefault(schema.SandboxAgentTag, tags, "openclaw"),
			Workspace: utils.GetTagsValueByDefault(schema.SandboxWorkspaceTag, tags, os.Getenv("PWD")),
			Network:   utils.GetTagsValueByDefault(schema.SandboxNetworkTag, tags, "restricted"),
			Name:      utils.GetTagsValueByDefault(schema.SandboxNameTag, tags, ""),
			Command:   utils.GetTagsValueByDefault(schema.SandboxCommandTag, tags, ""),
		},
	}

	switch spec.Backend {
	case schema.BackendDocker:
		if spec.Image.SHA == "" {
			return schema.RuntimeSpec{}, errors.New("Image-ID is empty")
		}
	case schema.BackendSandbox:
		if spec.Sandbox.Workspace == "" {
			return schema.RuntimeSpec{}, errors.New("Sandbox-Workspace is empty")
		}
	default:
		return schema.RuntimeSpec{}, errors.New("unsupported runtime backend: " + spec.Backend)
	}

	return spec, nil
}

// CheckModuleFormat validates the module configuration for the selected runtime backend.
func CheckModuleFormat(moduleFormat string, tags []goarSchema.Tag) error {
	_, err := RuntimeSpecFromTags(moduleFormat, tags)
	if err != nil {
		return err
	}

	return nil
}
