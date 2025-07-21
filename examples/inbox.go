package main

import (
	"fmt"
	"io"
	"os"

	"github.com/hymatrix/hymx/sdk"
	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func inbox() {
	// s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")

	// spawn target1
	res, err := s.SpawnAndWait(
		"LSjhdzBjyWuyUPe-g6PUzt8t1PUlw2FZ9SM3_hCh2Is",
		"eIgnDk4vSKPe0lYB6yhCHDV1dOw3JgYHGocfj7WGrjQ",
		[]goarSchema.Tag{},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}
	target1 := res.Id
	fmt.Println("spawn target1: ", target1)

	// spawn target2
	res, err = s.SpawnAndWait(
		"LSjhdzBjyWuyUPe-g6PUzt8t1PUlw2FZ9SM3_hCh2Is",
		"eIgnDk4vSKPe0lYB6yhCHDV1dOw3JgYHGocfj7WGrjQ",
		[]goarSchema.Tag{},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}
	target2 := res.Id
	fmt.Println("spawn target2: ", target2)

	inbox_step1(s, target1, target2, "inbox.lua")
	inbox_step2(s, target1, target2)
}

func inbox_step1(s *sdk.SDK, target1, target2, luaFile string) {
	// Eval
	filePath := luaFile

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
	res, err := s.SendMessageAndWait(target1, strCode,
		[]schema.Tag{
			{Name: "Action", Value: "Eval"},
			{Name: "Target", Value: target1},
			{Name: "Module", Value: "0x84534"},
			{Name: "Block-Height", Value: "100000"},
			{Name: "Data", Value: strCode},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("target1 load ok, ", res)

	res, err = s.SendMessageAndWait(target2, strCode,
		[]schema.Tag{
			{Name: "Action", Value: "Eval"},
			{Name: "Target", Value: target2},
			{Name: "Module", Value: "0x84534"},
			{Name: "Block-Height", Value: "100000"},
			{Name: "Data", Value: strCode},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("target2 load ok, ", res)
}

func inbox_step2(s *sdk.SDK, target1, target2 string) {
	res, err := s.SendMessageAndWait(target1, "",
		[]schema.Tag{
			{Name: "Action", Value: "SendMsg"},
			{Name: "Target", Value: target1},
			{Name: "SendTo", Value: target2},
		})
	if err != nil {
		fmt.Println("sendto error: ", err)
		return
	}
	fmt.Println("SendMsg ok, ", res)
}
