package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hymatrix/hymx/schema"
)

func genModule() {
	item, _ := s.GenerateModule([]byte{}, schema.Module{
		Base:         schema.DefaultBaseModule,
		ModuleFormat: "golua",
	})
	bin, _ := json.Marshal(item)

	filename := fmt.Sprintf("mod-%s.json", item.Id)
	os.WriteFile(filename, bin, 0644)

}
