package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	vmdockerSchema "github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	vmdockerUtils "github.com/cryptowizard0/vmdocker/vmdocker/utils"
	"github.com/hymatrix/hymx/schema"
	arSchema "github.com/permadao/goar/schema"
)

func genModule() {
	tags := buildModuleTags()
	itemId, err := s.SaveModule([]byte{}, schema.Module{
		Base:         schema.DefaultBaseModule,
		ModuleFormat: vmdockerSchema.ModuleFormat,
		Tags:         tags,
	})
	if err != nil {
		fmt.Println("generate and save module failed, ", "err", err)
		return
	}
	fmt.Println("generate and save module success, ", "id", itemId)
}

// buildModuleTags returns module tags for either build mode or pull mode.
//
// Build mode (triggered by setting VMDOCKER_BUILD_DOCKERFILE):
//
//	VMDOCKER_BUILD_DOCKERFILE  - path to the Dockerfile to embed
//	                             (e.g. ../../vmdocker_agent/Dockerfile.sandbox)
//	VMDOCKER_BUILD_CONTEXT_URL - optional remote build context URL (tarball or git URL)
//	VMDOCKER_BUILD_CONTEXT_DIR - optional local build context dir to embed in module tags
//	                             (default: directory containing VMDOCKER_BUILD_DOCKERFILE)
//	VMDOCKER_BUILD_TAG         - optional local image tag after build
//	                             (default: derived from embedded build content)
//
// Pull mode (default):
//
//	VMDOCKER_SANDBOX_IMAGE_NAME - Docker image name
//	VMDOCKER_SANDBOX_IMAGE_ID   - Docker image SHA256 digest
func buildModuleTags() []arSchema.Tag {
	base := []arSchema.Tag{
		{Name: "Runtime-Backend", Value: "sandbox"},
		{Name: "Sandbox-Agent", Value: "shell"},
		{Name: "Openclaw-Version", Value: "2026.3.1-beta.1"},
	}

	if dockerfilePath := os.Getenv("VMDOCKER_BUILD_DOCKERFILE"); dockerfilePath != "" {
		return buildModeModuleTags(base, dockerfilePath)
	}
	return pullModeModuleTags(base)
}

func buildModeModuleTags(base []arSchema.Tag, dockerfilePath string) []arSchema.Tag {
	dockerfilePath, err := filepath.Abs(dockerfilePath)
	if err != nil {
		fmt.Printf("failed to resolve Dockerfile path %s: %v\n", dockerfilePath, err)
		os.Exit(1)
	}
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		fmt.Printf("failed to read Dockerfile at %s: %v\n", dockerfilePath, err)
		os.Exit(1)
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	contextURL := os.Getenv("VMDOCKER_BUILD_CONTEXT_URL")
	contextArchive := ""
	if contextURL == "" {
		contextDir := GetEnvWith("VMDOCKER_BUILD_CONTEXT_DIR", filepath.Dir(dockerfilePath))
		contextDir, err = filepath.Abs(contextDir)
		if err != nil {
			fmt.Printf("failed to resolve build context path %s: %v\n", contextDir, err)
			os.Exit(1)
		}
		contextArchive, err = vmdockerUtils.CompressDirectory(contextDir)
		if err != nil {
			fmt.Printf("failed to compress build context at %s: %v\n", contextDir, err)
			os.Exit(1)
		}
	}
	buildArgs := buildArgsFromEnv()
	buildTag := GetEnvWith("VMDOCKER_BUILD_TAG", defaultBuildTag(encoded, contextURL, contextArchive, buildArgs))

	tags := append(base,
		arSchema.Tag{Name: "Build-Type", Value: "dockerfile"},
		arSchema.Tag{Name: "Build-Dockerfile-Content", Value: encoded},
		arSchema.Tag{Name: "Build-Tag", Value: buildTag},
	)
	if contextURL != "" {
		tags = append(tags, arSchema.Tag{Name: "Build-Context-URL", Value: contextURL})
	} else {
		tags = append(tags, arSchema.Tag{Name: vmdockerSchema.BuildContextTag, Value: contextArchive})
	}
	tags = append(tags, buildArgs...)
	fmt.Printf("build mode: Dockerfile=%s tag=%s context_url=%s embedded_context=%t\n", dockerfilePath, buildTag, contextURL, contextArchive != "")
	return tags
}

func pullModeModuleTags(base []arSchema.Tag) []arSchema.Tag {
	imageName := os.Getenv("VMDOCKER_SANDBOX_IMAGE_NAME")
	imageID := os.Getenv("VMDOCKER_SANDBOX_IMAGE_ID")
	if imageName == "" || imageID == "" {
		imageName = "chriswebber/docker-openclaw-sandbox:fix-test"
		imageID = "sha256:4daa6b51a12f41566bca09c2ca92a4982263db47f40d20d11c8f83f6ae85bc0e"
	}
	return append(base,
		arSchema.Tag{Name: "Image-Name", Value: imageName},
		arSchema.Tag{Name: "Image-ID", Value: imageID},
	)
}

func buildArgsFromEnv() []arSchema.Tag {
	const prefix = "VMDOCKER_BUILD_ARG_"
	keys := make([]string, 0)
	values := make(map[string]string)
	for _, env := range os.Environ() {
		key, value, ok := strings.Cut(env, "=")
		if !ok || !strings.HasPrefix(key, prefix) {
			continue
		}
		argName := strings.TrimPrefix(key, prefix)
		if argName == "" {
			continue
		}
		keys = append(keys, argName)
		values[argName] = value
	}
	sort.Strings(keys)

	tags := make([]arSchema.Tag, 0, len(keys))
	for _, key := range keys {
		tags = append(tags, arSchema.Tag{
			Name:  vmdockerSchema.BuildArgTagPrefix + key,
			Value: values[key],
		})
	}
	return tags
}

func defaultBuildTag(dockerfile, contextURL, contextArchive string, buildArgs []arSchema.Tag) string {
	sum := sha256.New()
	sum.Write([]byte(dockerfile))
	sum.Write([]byte{0})
	sum.Write([]byte(contextURL))
	sum.Write([]byte{0})
	sum.Write([]byte(contextArchive))
	sum.Write([]byte{0})
	for _, tag := range buildArgs {
		sum.Write([]byte(tag.Name))
		sum.Write([]byte{'='})
		sum.Write([]byte(tag.Value))
		sum.Write([]byte{0})
	}
	return "vmdocker-openclaw:" + hex.EncodeToString(sum.Sum(nil))[:12]
}
