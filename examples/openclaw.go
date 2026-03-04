package main

import (
	"fmt"

	"github.com/cryptowizard0/vmdocker/vmdocker/utils"
	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

var (
	OpenclawModuleID = "S9UxD8ZmhowbDnCZv4IobxTIEU_OxsVHbk47Hl4Bg6o"
)

func spawnOpenclaw() string {
	res, err := s.SpawnAndWait(
		OpenclawModuleID,
		scheduler,
		[]goarSchema.Tag{
			{Name: "model", Value: "kimi-coding/k2p5"},
			{Name: "apiKey", Value: "sk-kimi-7p1NNBVQXasKKaBGD4WxWppcvewjUp9x3TEuhVcpi1p1Hqq49ZM3fvPgI6wj3jyB"},
			// {Name: utils.ContainerEnvTagPrefix + "RUNTIME_TYPE", Value: "openclaw"},
			// {Name: utils.ContainerEnvTagPrefix + "OPENCLAW_GATEWAY_URL", Value: "http://127.0.0.1:18789"},
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
			{Name: "botToken", Value: "8659441717:AAFjCdlu_o9D4xbSRshfqN5Kfku3VvyLol0"},
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
