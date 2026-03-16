package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

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

	imageInfo, err := imageInfoFromTags(tags)
	if err != nil {
		return schema.RuntimeSpec{}, err
	}

	backend := utils.GetTagsValueByDefault(schema.RuntimeBackendTag, tags, schema.BackendSandbox)
	spec := schema.RuntimeSpec{
		Backend: backend,
		Image:   imageInfo,
		Sandbox: schema.SandboxSpec{
			Agent:     utils.GetTagsValueByDefault(schema.SandboxAgentTag, tags, "shell"),
			Workspace: utils.GetTagsValueByDefault(schema.SandboxWorkspaceTag, tags, ""),
			Network:   utils.GetTagsValueByDefault(schema.SandboxNetworkTag, tags, ""),
			Name:      utils.GetTagsValueByDefault(schema.SandboxNameTag, tags, ""),
			Command:   utils.GetTagsValueByDefault(schema.SandboxCommandTag, tags, ""),
		},
	}

	switch spec.Backend {
	case schema.BackendDocker, schema.BackendSandbox:
		if spec.Image.Build == nil && spec.Image.SHA == "" {
			return schema.RuntimeSpec{}, errors.New("Image-ID is empty")
		}
	default:
		return schema.RuntimeSpec{}, errors.New("unsupported runtime backend: " + spec.Backend)
	}

	return spec, nil
}

// imageInfoFromTags parses image configuration from module tags.
// When "Build-Type: dockerfile" is present it populates BuildSpec from the
// inline Dockerfile content tag; otherwise it falls back to the pull-mode
// Image-Name / Image-ID tags.
func imageInfoFromTags(tags []goarSchema.Tag) (schema.ImageInfo, error) {
	buildType := utils.GetTagsValueByDefault(schema.BuildTypeTag, tags, "")
	if buildType != "dockerfile" {
		imageName := utils.GetTagsValueByDefault("Image-Name", tags, "")
		if imageName == "" {
			return schema.ImageInfo{}, errors.New("Image-Name is empty")
		}
		return schema.ImageInfo{
			Name: imageName,
			SHA:  utils.GetTagsValueByDefault("Image-ID", tags, ""),
		}, nil
	}

	encoded := utils.GetTagsValueByDefault(schema.BuildDockerfileTag, tags, "")
	if encoded == "" {
		return schema.ImageInfo{}, errors.New("Build-Dockerfile-Content is empty")
	}
	dockerfileBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return schema.ImageInfo{}, fmt.Errorf("decode Build-Dockerfile-Content failed: %w", err)
	}

	buildTag := utils.GetTagsValueByDefault(schema.BuildTagTag, tags, "vmdocker-openclaw:local")
	return schema.ImageInfo{
		Name: buildTag,
		Build: &schema.BuildSpec{
			Dockerfile:     string(dockerfileBytes),
			ContextURL:     utils.GetTagsValueByDefault(schema.BuildContextURLTag, tags, ""),
			ContextArchive: utils.GetTagsValueByDefault(schema.BuildContextTag, tags, ""),
			Tag:            buildTag,
			Args:           buildArgsFromTags(tags),
		},
	}, nil
}

// buildArgsFromTags extracts docker --build-arg entries from tags prefixed with BuildArgTagPrefix.
func buildArgsFromTags(tags []goarSchema.Tag) map[string]string {
	args := make(map[string]string)
	for _, tag := range tags {
		if !strings.HasPrefix(tag.Name, schema.BuildArgTagPrefix) {
			continue
		}
		key := strings.TrimPrefix(tag.Name, schema.BuildArgTagPrefix)
		if key != "" {
			args[key] = tag.Value
		}
	}
	return args
}

// CheckModuleFormat validates the module configuration for the selected runtime backend.
func CheckModuleFormat(moduleFormat string, tags []goarSchema.Tag) error {
	_, err := RuntimeSpecFromTags(moduleFormat, tags)
	if err != nil {
		return err
	}

	return nil
}
