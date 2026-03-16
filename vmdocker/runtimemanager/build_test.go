package runtimemanager

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	vmdockerUtils "github.com/cryptowizard0/vmdocker/vmdocker/utils"
)

func TestBuildImageFromSpecRebuildsWhenCacheKeyDiffers(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "docker.log")
	fakeDocker := filepath.Join(tempDir, "docker")
	script := "#!/bin/sh\n" +
		"printf '%s\n' \"$*\" >>" + shellEscapeForTest(logPath) + "\n" +
		"if [ \"$1\" = \"image\" ] && [ \"$2\" = \"inspect\" ]; then\n" +
		"  if [ \"$3\" = \"--format\" ]; then\n" +
		"    echo stale-cache-key\n" +
		"    exit 0\n" +
		"  fi\n" +
		"fi\n" +
		"if [ \"$1\" = \"build\" ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker failed: %v", err)
	}

	spec := &schema.BuildSpec{
		Dockerfile:     "FROM scratch\nCOPY hello.txt /hello.txt\n",
		ContextArchive: compressContextForBuildTest(t, map[string]string{"hello.txt": "hello"}),
		Tag:            "vmdocker-openclaw:test",
	}

	if err := buildImageFromSpec(context.Background(), fakeDocker, spec); err != nil {
		t.Fatalf("buildImageFromSpec failed: %v", err)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log failed: %v", err)
	}
	log := string(raw)
	if !strings.Contains(log, "build -f") {
		t.Fatalf("expected docker build after cache-key mismatch, got:\n%s", log)
	}
	if !strings.Contains(log, "--label "+buildCacheLabel+"=") {
		t.Fatalf("expected build cache label in docker build command, got:\n%s", log)
	}
}

func TestBuildImageFromSpecUsesEmbeddedContextArchive(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "docker.log")
	fakeDocker := filepath.Join(tempDir, "docker")
	script := "#!/bin/sh\n" +
		"printf '%s\n' \"$*\" >>" + shellEscapeForTest(logPath) + "\n" +
		"if [ \"$1\" = \"image\" ] && [ \"$2\" = \"inspect\" ]; then\n" +
		"  exit 1\n" +
		"fi\n" +
		"if [ \"$1\" = \"build\" ]; then\n" +
		"  ctx=${@: -1}\n" +
		"  test -f \"$ctx/hello.txt\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker failed: %v", err)
	}

	spec := &schema.BuildSpec{
		Dockerfile:     "FROM scratch\nCOPY hello.txt /hello.txt\n",
		ContextArchive: compressContextForBuildTest(t, map[string]string{"hello.txt": "hello"}),
		Tag:            "vmdocker-openclaw:test",
	}

	if err := buildImageFromSpec(context.Background(), fakeDocker, spec); err != nil {
		t.Fatalf("buildImageFromSpec failed: %v", err)
	}
}

func TestBuildImageFromSpecSkipsWhenCacheKeyMatches(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "docker.log")
	fakeDocker := filepath.Join(tempDir, "docker")
	spec := &schema.BuildSpec{
		Dockerfile:     "FROM scratch\nCOPY hello.txt /hello.txt\n",
		ContextArchive: compressContextForBuildTest(t, map[string]string{"hello.txt": "hello"}),
		Tag:            "vmdocker-openclaw:test",
	}
	cacheKey := buildSpecCacheKey(spec)
	script := "#!/bin/sh\n" +
		"printf '%s\n' \"$*\" >>" + shellEscapeForTest(logPath) + "\n" +
		"if [ \"$1\" = \"image\" ] && [ \"$2\" = \"inspect\" ]; then\n" +
		"  if [ \"$3\" = \"--format\" ]; then\n" +
		"    echo " + shellEscapeForTest(cacheKey) + "\n" +
		"    exit 0\n" +
		"  fi\n" +
		"fi\n" +
		"if [ \"$1\" = \"build\" ]; then\n" +
		"  echo unexpected-build >&2\n" +
		"  exit 1\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker failed: %v", err)
	}

	if err := buildImageFromSpec(context.Background(), fakeDocker, spec); err != nil {
		t.Fatalf("buildImageFromSpec failed: %v", err)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log failed: %v", err)
	}
	log := string(raw)
	if strings.Contains(log, "build -f") {
		t.Fatalf("expected build to be skipped when cache key matches, got:\n%s", log)
	}
}

func compressContextForBuildTest(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, contents := range files {
		fullPath := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
	}
	archive, err := vmdockerUtils.CompressDirectory(dir)
	if err != nil {
		t.Fatalf("compress context failed: %v", err)
	}
	return archive
}
