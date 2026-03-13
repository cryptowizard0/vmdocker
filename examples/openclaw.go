package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/utils"
	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

var (
	OpenclawModuleID = os.Getenv("OPENCLAW_MODULE_ID")
)

func spawnOpenclaw() string {
	if OpenclawModuleID == "" {
		OpenclawModuleID = GetEnvWith("OPENCLAW_MODULE_ID", "AzbJ2MZ7hz5gJnbhWFbQ9JJRRJe4VdVtDAePKUjC6zM")
	}

	openclawModel := GetEnvWith("OPENCLAW_MODEL", "kimi-coding/k2p5")
	openclawAPIKey := os.Getenv("OPENCLAW_API_KEY")
	sandboxWorkspace := GetEnvWith("OPENCLAW_SANDBOX_WORKSPACE", ".")
	openclawGatewayToken := GetEnvWith("OPENCLAW_GATEWAY_TOKEN", "openclaw-test-token")

	start := time.Now()
	fmt.Printf("[openclaw_spawn] start=%s module=%s\n", start.Format(time.RFC3339), OpenclawModuleID)
	res, err := s.SpawnAndWait(
		OpenclawModuleID,
		scheduler,
		[]goarSchema.Tag{
			{Name: "model", Value: openclawModel},
			{Name: "apiKey", Value: openclawAPIKey},
			{Name: "Sandbox-Workspace", Value: sandboxWorkspace},
			{Name: utils.ContainerEnvTagPrefix + "OPENCLAW_GATEWAY_TOKEN", Value: openclawGatewayToken},
		},
	)
	if err != nil {
		fmt.Printf("[openclaw_spawn] failed after=%s err=%v\n", time.Since(start), err)
		return ""
	}

	target := res.Id
	fmt.Printf("[openclaw_spawn] done=%s elapsed=%s pid=%s\n", time.Now().Format(time.RFC3339), time.Since(start), target)

	return target
}

func chatOpenclaw(target string) {
	start := time.Now()
	fmt.Printf("[openclaw_chat] start=%s target=%s\n", start.Format(time.RFC3339), target)
	res, err := s.SendMessageAndWait(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "Chat"},
			{Name: "Command", Value: "你好"},
		})
	if err != nil {
		fmt.Printf("[openclaw_chat] failed after=%s target=%s err=%v\n", time.Since(start), target, err)
		return
	}
	fmt.Printf("[openclaw_chat] done=%s elapsed=%s target=%s\n", time.Now().Format(time.RFC3339), time.Since(start), target)
	fmt.Println("res, ", res)

}

func telegramOpenclaw(target string) {
	botToken := GetEnv("OPENCLAW_TELEGRAM_BOT_TOKEN")
	defaultAccount := GetEnvWith("OPENCLAW_TELEGRAM_DEFAULT_ACCOUNT", "main")
	dmPolicy := GetEnvWith("OPENCLAW_TELEGRAM_DM_POLICY", "pairing")

	start := time.Now()
	fmt.Printf("[openclaw_tg] start=%s target=%s\n", start.Format(time.RFC3339), target)
	res, err := s.SendMessageAndWait(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "ConfigureTelegram"},
			{Name: "botToken", Value: botToken},
			{Name: "defaultAccount", Value: defaultAccount},
			{Name: "dmPolicy", Value: dmPolicy},
		})
	if err != nil {
		fmt.Printf("[openclaw_tg] failed after=%s target=%s err=%v\n", time.Since(start), target, err)
		return
	}
	fmt.Printf("[openclaw_tg] done=%s elapsed=%s target=%s\n", time.Now().Format(time.RFC3339), time.Since(start), target)
	fmt.Println("res, ", res)

}

func pairTgOpenclaw(target, pairCode string) {
	start := time.Now()
	fmt.Printf("[openclaw_pair] start=%s target=%s\n", start.Format(time.RFC3339), target)
	res, err := s.SendMessageAndWait(target, "",
		[]schema.Tag{
			{Name: "Action", Value: "ApproveTelegramPairing"},
			{Name: "code", Value: pairCode},
			{Name: "channel", Value: "telegram"},
			{Name: "dmPolicy", Value: "pairing"},
		})
	if err != nil {
		fmt.Printf("[openclaw_pair] failed after=%s target=%s err=%v\n", time.Since(start), target, err)
		return
	}
	fmt.Printf("[openclaw_pair] done=%s elapsed=%s target=%s\n", time.Now().Format(time.RFC3339), time.Since(start), target)
	fmt.Println("res, ", res)

}
