package main

import (
	"fmt"
	"io"
	"os"

	"github.com/hymatrix/hymx/sdk"
	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func spawnChild() {
	res, err := s.SpawnAndWait(
		"GjkXoqJuVmrmgwfekxP5ykrlmfSV3ESgh4rb0E-jZfE",
		"0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51",
		[]goarSchema.Tag{},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}
	target := res.Id
	fmt.Println("Spawned: ", res)

	spawn_step1(s, target)
	spawn_step2(s, target)
	// spawn_step3(s, target, "yF1THHwLjrTBYu6crhvvsCGF9256GgALGhlXSE5YuRU")
}

func spawn_step1(s *sdk.SDK, target string) {
	// Eval
	filePath := "spawn.lua"

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Failed to open file: ", err)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Failed to read file: ", err)
		return
	}
	strCode := string(content)
	res, err := s.SendMessageAndWait(target, strCode,
		[]schema.Tag{
			{Name: "Action", Value: "Eval"},
			{Name: "Target", Value: target},
			{Name: "Module", Value: "GjkXoqJuVmrmgwfekxP5ykrlmfSV3ESgh4rb0E-jZfE"},
			{Name: "Block-Height", Value: "100000"},
			{Name: "Data", Value: strCode},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("target load ok, ", res)
}

func spawn_step2(s *sdk.SDK, target string) {
	res, err := s.SendMessageAndWait(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "New"},
			{Name: "Target", Value: target},
		})
	if err != nil {
		fmt.Println("sendto error: ", err)
		return
	}
	fmt.Println("sendto ok, ", res)
}

func spawn_step3(s *sdk.SDK, target, target2 string) {
	childProcess := target2

	code := `
		print("hello eval")
		local json = require("json")
		-- local data = json.encode(ao.env)
		-- print(data)
		print(Owner)
		print(ao.id)
	`
	res, err := s.SendMessage(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "SendEval"},
			{Name: "Target", Value: target},
			{Name: "SendTo", Value: childProcess},
			{Name: "Data", Value: code},
		})
	if err != nil {
		fmt.Println("sendto error: ", err)
		return
	}
	fmt.Println("sendto ok, ", res)
}
