package main

import (
	"fmt"

	serverSchema "github.com/hymatrix/hymx/server/schema"
	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func ollama() {
	//s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")
	var res *serverSchema.Response
	var err error

	res, err = s.SpawnAndWait(
		"LSjhdzBjyWuyUPe-g6PUzt8t1PUlw2FZ9SM3_hCh2Is",
		"eIgnDk4vSKPe0lYB6yhCHDV1dOw3JgYHGocfj7WGrjQ",
		[]goarSchema.Tag{
			{Name: "Module-Format", Value: "ollama"},
			// {Name: "RuntimeType", Value: "golua"},
		},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}
	fmt.Println("spawn success: ", res)

	target := res.Id

	res, err = s.SendMessageAndWait(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "Chat"},
			{Name: "Prompt", Value: "hello"},
			{Name: "Target", Value: target},
			{Name: "Module", Value: "0x84534"},
			{Name: "Block-Height", Value: "100000"},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("target1 load ok, ", res)
}
