package main

import (
	"fmt"

	"github.com/cryptowizard0/vmdocker/vmdocker/utils"
	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

var (
	OpenclawModuleID = "SyNj7jn69zXD6y8nQjRYcnbJawAAPdd4khIiR_2w-6Q"
)

func spawnOpenclaw() string {
	res, err := s.SpawnAndWait(
		OpenclawModuleID,
		scheduler,
		[]goarSchema.Tag{
			{Name: "model", Value: "kimi-coding/k2p5"},
			{Name: "apiKey", Value: "sk-kimi-xxxx"},
			{Name: utils.ContainerEnvTagPrefix + "OPENCLAW_GATEWAY_TOKEN", Value: "openclaw-test-token"},
		},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return ""
	}

	target := res.Id
	fmt.Println("spawn ok, pid: ", target)

	return target
}

func chatOpenclaw(target string) {
	res, err := s.SendMessageAndWait(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "Chat"},
			{Name: "Command", Value: "你好"},
		})
	if err != nil {
		fmt.Println("Failed to apply: ", err)
		return
	}
	fmt.Println("res, ", res)

}

func telegramOpenclaw(target string) {
	res, err := s.SendMessageAndWait(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "ConfigureTelegram"},
			{Name: "botToken", Value: "xxxxx"},
			{Name: "defaultAccount", Value: "main"},
			{Name: "dmPolicy", Value: "pairing"},
		})
	if err != nil {
		fmt.Println("Failed to apply: ", err)
		return
	}
	fmt.Println("res, ", res)

}

func pairTgOpenclaw(target, pairCode string) {
	res, err := s.SendMessageAndWait(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "ApproveTelegramPairing"},
			{Name: "code", Value: pairCode},
			{Name: "channel", Value: "telegram"},
			{Name: "dmPolicy", Value: "pairing"},
		})
	if err != nil {
		fmt.Println("Failed to apply: ", err)
		return
	}
	fmt.Println("res, ", res)

}
