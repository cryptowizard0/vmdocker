package vmdocker

import (
	"strings"
	"testing"
)

func TestBuildSandboxCurlCommandFromFile(t *testing.T) {
	command := buildSandboxCurlCommandFromFile("/vmm/spawn", "/tmp/request.json")
	if !strings.Contains(command, "--data-binary @'/tmp/request.json'") {
		t.Fatalf("expected curl command to read payload from file, got: %s", command)
	}
	if strings.Contains(command, "--data-raw") {
		t.Fatalf("expected file-based curl command to avoid --data-raw, got: %s", command)
	}
}
