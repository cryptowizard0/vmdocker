package utils

import (
	"errors"

	"github.com/cryptowizard0/vmdocker/vmdocker/schema"
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

// CheckModuleFormat validates that the module format:
// - starts with "web.vmdocker-" prefix
// - contains Image-Name and Image-ID tags
func CheckModuleFormat(moduleFormat string, tags []goarSchema.Tag) error {
	if moduleFormat != schema.ModuleFormat {
		return errors.New("module format is not " + schema.ModuleFormat)
	}

	imageName := utils.GetTagsValueByDefault("Image-Name", tags, "")
	if imageName == "" {
		return errors.New("Image-Name is empty")
	}

	imageID := utils.GetTagsValueByDefault("Image-ID", tags, "")
	if imageID == "" {
		return errors.New("Image-ID is empty")
	}

	return nil
}
