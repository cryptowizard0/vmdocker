package main

import (
	"fmt"
	"time"

	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func result() {
	// s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")

	// spawn target1
	res, err := s.Spawn(
		"LSjhdzBjyWuyUPe-g6PUzt8t1PUlw2FZ9SM3_hCh2Is",
		"eIgnDk4vSKPe0lYB6yhCHDV1dOw3JgYHGocfj7WGrjQ",
		[]goarSchema.Tag{},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}
	target := res.Message

	code := `
		print("hello eval")
		-- local json = require("json")
		-- local data = json.encode(ao.env)
		print("ao.id:" .. ao.id)
	`

	code = "require('.process')._version"

	// Record start time
	startTime := time.Now()
	res, err = s.SendMessage(target, code,
		[]schema.Tag{
			{Name: "Action", Value: "Eval"},
			{Name: "Target", Value: target},
			{Name: "Module", Value: "0x84534"},
			{Name: "Block-Height", Value: "100000"},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Printf("result from return (took %v): %s\n", time.Since(startTime), res.Message)

	// Record Result call time
	startTime = time.Now()
	result, err := s.Client.GetResult(res.Id)
	if err != nil {
		fmt.Println("Failed to get result: ", err)
		return
	}
	fmt.Printf("result from Result (took %v): %+v\n", time.Since(startTime), result)

	// Record GetResults call time
	startTime = time.Now()
	results, err := s.Client.GetResults(target, 5)
	if err != nil {
		fmt.Println("Failed to get results: ", err)
		return
	}
	fmt.Printf("results from GetResults (took %v): %+v\n", time.Since(startTime), results)
}
