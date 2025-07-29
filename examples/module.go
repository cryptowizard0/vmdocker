package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hymatrix/hymx/schema"
	arSchema "github.com/permadao/goar/schema"
)

func genModule() {
	// ex ModuleFormat: "org.type.1.0.0"
	item, _ := s.GenerateModule([]byte{}, schema.Module{
		Base:         schema.DefaultBaseModule,
		ModuleFormat: "web.vmdocker-golua-ao.v0.0.1",
		Tags: []arSchema.Tag{
			{Name: "Image-Name", Value: "chriswebber/docker-golua:v0.0.2"},
			{Name: "Image-ID", Value: "sha256:a083ff79090cdd084ea428b6a9896e29eddfd767b5c5637a65a02ce1e073a88b"},
		},
	})
	bin, _ := json.Marshal(item)

	filename := fmt.Sprintf("mod-%s.json", item.Id)
	os.WriteFile(filename, bin, 0644)

}
