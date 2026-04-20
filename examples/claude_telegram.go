package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/utils"
	serverSchema "github.com/hymatrix/hymx/server/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func buildClaudeSpawnTags1(apiKey, baseURL, model, codeFlags, runtimeBackend, botToken string) []goarSchema.Tag {
	tags := make([]goarSchema.Tag, 0, 6)

	tags = append(tags,
		goarSchema.Tag{Name: utils.ContainerEnvTagPrefix + "RUNTIME_TYPE", Value: "telegramcustomer"},
		goarSchema.Tag{Name: utils.ContainerEnvTagPrefix + "ANTHROPIC_API_KEY", Value: apiKey},
	)

	if baseURL != "" {
		tags = append(tags, goarSchema.Tag{Name: utils.ContainerEnvTagPrefix + "ANTHROPIC_BASE_URL", Value: baseURL})
	}
	if model != "" {
		tags = append(tags, goarSchema.Tag{Name: utils.ContainerEnvTagPrefix + "ANTHROPIC_MODEL", Value: model})
	}
	if codeFlags != "" {
		tags = append(tags, goarSchema.Tag{Name: utils.ContainerEnvTagPrefix + "CLAUDE_CODE_FLAGS", Value: codeFlags})
	}
	if botToken != "" {
		tags = append(tags, goarSchema.Tag{Name: utils.ContainerEnvTagPrefix + "BOT_TOKEN", Value: botToken})
	}
	tags = append(tags, runtimeBackendTags(runtimeBackend)...)

	return tags
}

func spawnClaudeTg() string {
	apiKey := GetEnv("ANTHROPIC_API_KEY")
	baseURL := GetEnvWith("ANTHROPIC_BASE_URL", "")
	model := GetEnvWith("ANTHROPIC_MODEL", GetEnvWith("CLAUDE_MODEL", ""))
	codeFlags := GetEnvWith("CLAUDE_CODE_FLAGS", "")
	runtimeBackend := GetEnvWith("RUNTIME_BACKEND", "")
	botToken := GetEnvWith("BOT_TOKEN", "")
	timeout := openclawWaitTimeout("CLAUDE_SPAWN_WAIT_TIMEOUT", 10*time.Minute)

	totalStart := time.Now()
	fmt.Printf("[claude_spawn] start=%s module=%s backend=%s model=%s\n", totalStart.Format(time.RFC3339), module, runtimeBackend, model)

	requestStart := time.Now()
	resp, err := s.Spawn(
		module,
		scheduler,
		buildClaudeSpawnTags1(apiKey, baseURL, model, codeFlags, runtimeBackend, botToken),
	)
	if err != nil {
		fmt.Printf("[claude_spawn] spawn_request_failed elapsed=%s err=%v\n", time.Since(requestStart), err)
		return ""
	}
	requestElapsed := time.Since(requestStart)
	fmt.Printf("[claude_spawn] spawn_request_ok elapsed=%s msg_id=%s wait_timeout=%s\n", requestElapsed, resp.Id, timeout)

	waitStart := time.Now()
	res, pollCount, err := waitForResponseStats1(resp.Id, resp.Id, timeout)
	if err != nil {
		fmt.Printf("[claude_spawn] wait_failed elapsed=%s polls=%d err=%v total=%s\n", time.Since(waitStart), pollCount, err, time.Since(totalStart))
		return ""
	}
	waitElapsed := time.Since(waitStart)

	target := res.Id
	fmt.Printf("[claude_spawn] done=%s pid=%s request_elapsed=%s wait_elapsed=%s total_elapsed=%s polls=%d\n", time.Now().Format(time.RFC3339), target, requestElapsed, waitElapsed, time.Since(totalStart), pollCount)
	return target
}

func waitForResponseStats1(pid, msgid string, timeout time.Duration) (*serverSchema.Response, int, error) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	polls := 0
	for {
		select {
		case <-deadline:
			return nil, polls, fmt.Errorf("timeout waiting for result after %s", timeout)
		case <-ticker.C:
			polls++
			result, err := s.Client.GetResult(pid, msgid)
			if err != nil {
				return nil, polls, err
			}
			if result.ItemId == "" {
				continue
			}
			payload, err := json.Marshal(result)
			if err != nil {
				return nil, polls, err
			}
			return &serverSchema.Response{
				Id:      responseID(result, msgid),
				Message: string(payload),
			}, polls, nil
		}
	}
}
