package vmdocker

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimeSchema "github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	goarSchema "github.com/permadao/goar/schema"
	goarUtils "github.com/permadao/goar/utils"
)

func TestEnsureModuleImageAvailableUsesLocalMatch(t *testing.T) {
	const (
		imageName = "example/image:test"
		imageID   = "sha256:expected"
	)

	fakeDocker, logPath, nameState, _, cleanup := installFakeDocker(t, imageName, imageID)
	defer cleanup()
	if err := os.WriteFile(nameState, []byte(""), 0o644); err != nil {
		t.Fatalf("write name state failed: %v", err)
	}

	originalLookPath := dockerLookPath
	dockerLookPath = func(file string) (string, error) {
		if file == "docker" {
			return fakeDocker, nil
		}
		return originalLookPath(file)
	}
	defer func() {
		dockerLookPath = originalLookPath
	}()

	if err := ensureModuleImageAvailable(context.Background(), "unused", runtimeSchema.ImageInfo{
		Name:          imageName,
		SHA:           imageID,
		Source:        runtimeSchema.ImageSourceModuleData,
		ArchiveFormat: runtimeSchema.ImageArchiveDockerSaveGZ,
	}); err != nil {
		t.Fatalf("ensureModuleImageAvailable failed: %v", err)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake docker log failed: %v", err)
	}
	if strings.Contains(string(raw), "image load") {
		t.Fatalf("expected local hit to skip docker image load, got log:\n%s", string(raw))
	}
}

func TestEnsureModuleImageAvailableLoadsFromModuleFileOnMiss(t *testing.T) {
	const (
		moduleID  = "module-1"
		imageName = "example/image:test"
		imageID   = "sha256:expected"
	)

	fakeDocker, logPath, nameState, _, cleanup := installFakeDocker(t, imageName, imageID)
	defer cleanup()

	originalLookPath := dockerLookPath
	dockerLookPath = func(file string) (string, error) {
		if file == "docker" {
			return fakeDocker, nil
		}
		return originalLookPath(file)
	}
	defer func() {
		dockerLookPath = originalLookPath
	}()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	if err := os.MkdirAll("mod", 0o755); err != nil {
		t.Fatalf("mkdir mod failed: %v", err)
	}
	if err := writeModulePayload(moduleID, []byte("tar-contents")); err != nil {
		t.Fatalf("write module payload failed: %v", err)
	}

	if err := ensureModuleImageAvailable(context.Background(), moduleID, runtimeSchema.ImageInfo{
		Name:          imageName,
		SHA:           imageID,
		Source:        runtimeSchema.ImageSourceModuleData,
		ArchiveFormat: runtimeSchema.ImageArchiveDockerSaveGZ,
	}); err != nil {
		t.Fatalf("ensureModuleImageAvailable failed: %v", err)
	}

	if _, err := os.Stat(nameState); err != nil {
		t.Fatalf("expected image tag state to exist after load: %v", err)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake docker log failed: %v", err)
	}
	log := string(raw)
	if !strings.Contains(log, "image load") {
		t.Fatalf("expected docker image load on cache miss, got log:\n%s", log)
	}
	if !strings.Contains(log, "image tag "+imageID+" "+imageName) {
		t.Fatalf("expected docker image tag after load, got log:\n%s", log)
	}
}

func installFakeDocker(t *testing.T, imageName, imageID string) (string, string, string, string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "docker.log")
	nameState := filepath.Join(tempDir, "name.state")
	idState := filepath.Join(tempDir, "id.state")
	fakeDocker := filepath.Join(tempDir, "docker")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >>%s
if [ "$1" = "image" ] && [ "$2" = "inspect" ]; then
  ref="$5"
  if [ "$ref" = %s ] && [ -f %s ]; then
    echo %s
    exit 0
  fi
  if [ "$ref" = %s ] && [ -f %s ]; then
    echo %s
    exit 0
  fi
  echo "missing image" >&2
  exit 1
fi
if [ "$1" = "image" ] && [ "$2" = "tag" ]; then
  if [ "$3" = %s ] && [ "$4" = %s ] && [ -f %s ]; then
    : > %s
    exit 0
  fi
  echo "cannot tag" >&2
  exit 1
fi
if [ "$1" = "image" ] && [ "$2" = "load" ]; then
  cat >/dev/null
  : > %s
  exit 0
fi
exit 0
`, shellEscapeForModuleTest(logPath), shellEscapeForModuleTest(imageName), shellEscapeForModuleTest(nameState), shellEscapeForModuleTest(imageID), shellEscapeForModuleTest(imageID), shellEscapeForModuleTest(idState), shellEscapeForModuleTest(imageID), shellEscapeForModuleTest(imageID), shellEscapeForModuleTest(imageName), shellEscapeForModuleTest(idState), shellEscapeForModuleTest(nameState), shellEscapeForModuleTest(idState))
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker failed: %v", err)
	}

	return fakeDocker, logPath, nameState, idState, func() {}
}

func writeModulePayload(moduleID string, payload []byte) error {
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	if _, err := gz.Write(payload); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}

	item := goarSchema.BundleItem{
		Data: goarUtils.Base64Encode(archive.Bytes()),
	}
	itemBin, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return os.WriteFile(moduleFilePath(moduleID), itemBin, 0o644)
}

func shellEscapeForModuleTest(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
