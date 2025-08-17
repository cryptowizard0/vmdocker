package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hymatrix/hymx/sdk"
	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func pingpong() {
	// s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")

	// spawn target1
	res, err := s.SpawnAndWait(
		module,
		scheduler,
		[]goarSchema.Tag{},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}
	target1 := res.Id
	fmt.Println("spawn target1: ", target1)

	// spawn target2
	s2 := sdk.NewFromBundler("http://127.0.0.1:8081", bundler)
	//s2 := s
	res, err = s2.SpawnAndWait(
		module,
		scheduler2,
		[]goarSchema.Tag{},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}
	target2 := res.Id
	fmt.Println("spawn target2: ", target2)

	time.Sleep(10000 * time.Millisecond)

	// load pingpong.lua
	pingpong_step1(s, target1, target2)

	// 等待用户输入后继续
	fmt.Println("Step1 完成，按回车键继续...")
	var input string
	fmt.Scanln(&input)

	// target1 send ping ===> target2
	// target2 resp pong ===> target1
	pingpong_step2(s, target1, target2)
}

func pingpong_step1(s *sdk.SDK, target1, target2 string) {
	// Eval
	filePath := "pingpong.lua"

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
			{Name: "Data", Value: strCode},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("target2 load ok, ", res)
}

func pingpong_step2(s *sdk.SDK, target1, target2 string) {
	// address := s.GetAddress()
	res, err := s.SendMessageAndWait(target1, "",
		[]schema.Tag{
			{Name: "Action", Value: "SendPing"},
			{Name: "Target", Value: target1},
			{Name: "SendTo", Value: target2},
		})
	if err != nil {
		fmt.Println("sendto error: ", err)
		return
	}
	fmt.Println("sendto ok, ", res)
}
