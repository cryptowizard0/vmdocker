package utils

import (
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
