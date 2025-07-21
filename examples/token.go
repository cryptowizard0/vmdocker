package main

import (
	"fmt"
	"io"
	"os"

	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func sendMessage() {
	// s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")

	// spawn target1
	res, err := s.Spawn(
		"LSjhdzBjyWuyUPe-g6PUzt8t1PUlw2FZ9SM3_hCh2Is",
		"0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51",
		[]goarSchema.Tag{},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}
	target := res.Message
	fmt.Println("spawn target1: ", target)

	// Eval
	filePath := "token.lua"

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

	res, err = s.SendMessage(target, strCode,
		[]schema.Tag{
			{Name: "Action", Value: "Eval"},
			{Name: "Target", Value: target},
			{Name: "Module", Value: "0x84534"},
			{Name: "Block-Height", Value: "100000"},
			{Name: "Data", Value: strCode},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("Eval ok, ", res)

	// info
	res, err = s.SendMessage(target, "",

		[]schema.Tag{
			{Name: "Action", Value: "Info"},
			{Name: "Target", Value: target},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("Info ok, ", res)

	// balance
	res, err = s.SendMessage(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "Balance"},
			{Name: "Target", Value: target},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("Balance ok, ", res)

	// mint
	res, err = s.SendMessage(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "Mint"},
			{Name: "Target", Value: target},
			{Name: "Quantity", Value: "1000"},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("Mint ok, ", res)
}
