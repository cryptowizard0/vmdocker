package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// buildModuleTags returns module tags for artifact-assisted build mode, build mode, or pull mode.
//
// Artifact mode (triggered by setting VMDOCKER_ARTIFACT_SOURCE_DIR):
//
//	VMDOCKER_ARTIFACT_SOURCE_DIR - local path to the vmdocker_agent source tree
//	VMDOCKER_ARTIFACT_GOOS       - target GOOS for compiled binaries (default: linux)
//	VMDOCKER_ARTIFACT_GOARCH     - target GOARCH for compiled binaries (default: arm64)
//	VMDOCKER_BUILD_TAG           - optional final image tag after spawn-time build
//
// Build mode (triggered by setting VMDOCKER_BUILD_DOCKERFILE or
// VMDOCKER_BUILD_DOCKERFILE_PATH):
//
//	VMDOCKER_BUILD_DOCKERFILE  - optional local path to the Dockerfile to embed
//	                             (e.g. ../../vmdocker_agent/Dockerfile.sandbox)
//	VMDOCKER_BUILD_DOCKERFILE_PATH
//	                           - optional Dockerfile path inside the remote
//	                             build context repo (e.g. Dockerfile.sandbox)
//	VMDOCKER_BUILD_CONTEXT_URL - optional remote build context URL (git URL
//	                             preferred when using VMDOCKER_BUILD_DOCKERFILE_PATH)
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

	if os.Getenv("VMDOCKER_ARTIFACT_SOURCE_DIR") != "" {
		return artifactModeModuleTags(base)
	}
	if os.Getenv("VMDOCKER_BUILD_DOCKERFILE") != "" || os.Getenv("VMDOCKER_BUILD_DOCKERFILE_PATH") != "" {
		return buildModeModuleTags(base)
	}
	return pullModeModuleTags(base)
}

func artifactModeModuleTags(base []arSchema.Tag) []arSchema.Tag {
	sourceDir := os.Getenv("VMDOCKER_ARTIFACT_SOURCE_DIR")
	sourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		fmt.Printf("failed to resolve artifact source path %s: %v\n", sourceDir, err)
		os.Exit(1)
	}

	contextDir, cleanup, err := buildArtifactContext(sourceDir)
	if err != nil {
		fmt.Printf("failed to build artifact context: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	archive, err := vmdockerUtils.CompressDirectory(contextDir)
	if err != nil {
		fmt.Printf("failed to compress artifact context at %s: %v\n", contextDir, err)
		os.Exit(1)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(artifactRuntimeDockerfile()))
	buildArgs := buildArgsFromEnv()
	buildTag := GetEnvWith("VMDOCKER_BUILD_TAG", defaultBuildTag(encoded, "", archive, buildArgs))

	tags := append(base,
		arSchema.Tag{Name: vmdockerSchema.BuildTypeTag, Value: "dockerfile"},
		arSchema.Tag{Name: vmdockerSchema.BuildDockerfileTag, Value: encoded},
		arSchema.Tag{Name: vmdockerSchema.BuildContextTag, Value: archive},
		arSchema.Tag{Name: vmdockerSchema.BuildTagTag, Value: buildTag},
	)
	tags = append(tags, buildArgs...)
	fmt.Printf("artifact mode: source=%s tag=%s embedded_context=%t\n", sourceDir, buildTag, true)
	return tags
}

func buildModeModuleTags(base []arSchema.Tag) []arSchema.Tag {
	content, dockerfileSource, localDockerfilePath, err := resolveDockerfileContent()
	if err != nil {
		fmt.Printf("failed to resolve Dockerfile content: %v\n", err)
		os.Exit(1)
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	contextURL := os.Getenv("VMDOCKER_BUILD_CONTEXT_URL")
	contextArchive := ""
	if contextURL == "" {
		if localDockerfilePath == "" {
			fmt.Println("VMDOCKER_BUILD_CONTEXT_URL is required when using VMDOCKER_BUILD_DOCKERFILE_PATH without VMDOCKER_BUILD_DOCKERFILE")
			os.Exit(1)
		}
		contextDir := GetEnvWith("VMDOCKER_BUILD_CONTEXT_DIR", filepath.Dir(localDockerfilePath))
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
	fmt.Printf("build mode: Dockerfile=%s tag=%s context_url=%s embedded_context=%t\n", dockerfileSource, buildTag, contextURL, contextArchive != "")
	return tags
}

func resolveDockerfileContent() ([]byte, string, string, error) {
	if dockerfilePath := os.Getenv("VMDOCKER_BUILD_DOCKERFILE"); dockerfilePath != "" {
		absPath, err := filepath.Abs(dockerfilePath)
		if err != nil {
			return nil, "", "", fmt.Errorf("resolve local Dockerfile path %s: %w", dockerfilePath, err)
		}
		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, "", "", fmt.Errorf("read local Dockerfile %s: %w", absPath, err)
		}
		return content, absPath, absPath, nil
	}

	dockerfilePath := os.Getenv("VMDOCKER_BUILD_DOCKERFILE_PATH")
	if dockerfilePath == "" {
		return nil, "", "", fmt.Errorf("set VMDOCKER_BUILD_DOCKERFILE or VMDOCKER_BUILD_DOCKERFILE_PATH")
	}
	contextURL := os.Getenv("VMDOCKER_BUILD_CONTEXT_URL")
	if contextURL == "" {
		return nil, "", "", fmt.Errorf("VMDOCKER_BUILD_CONTEXT_URL is required when using VMDOCKER_BUILD_DOCKERFILE_PATH")
	}

	rawURL, err := githubRawURLFromContextURL(contextURL, dockerfilePath)
	if err != nil {
		return nil, "", "", err
	}
	content, err := fetchURL(rawURL)
	if err != nil {
		return nil, "", "", fmt.Errorf("fetch remote Dockerfile %s: %w", rawURL, err)
	}
	return content, rawURL, "", nil
}

func fetchURL(rawURL string) ([]byte, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

var githubArchivePattern = regexp.MustCompile(`^/([^/]+)/([^/]+)/archive/(.+)\.(?:tar\.gz|zip)$`)

func githubRawURLFromContextURL(contextURL, dockerfilePath string) (string, error) {
	u, err := neturl.Parse(contextURL)
	if err != nil {
		return "", fmt.Errorf("parse VMDOCKER_BUILD_CONTEXT_URL %q: %w", contextURL, err)
	}
	if !strings.EqualFold(u.Host, "github.com") {
		return "", fmt.Errorf("remote Dockerfile fetch currently supports github.com context URLs only")
	}
	ref := u.Fragment
	path := strings.TrimSuffix(u.Path, "/")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "https://github.com/")

	switch {
	case strings.HasSuffix(path, ".git"):
		path = strings.TrimSuffix(path, ".git")
		if ref == "" {
			ref = "HEAD"
		}
	case githubArchivePattern.MatchString(u.Path):
		matches := githubArchivePattern.FindStringSubmatch(u.Path)
		path = matches[1] + "/" + matches[2]
		if ref == "" {
			ref = strings.TrimPrefix(matches[3], "refs/heads/")
			ref = strings.TrimPrefix(ref, "refs/tags/")
		}
	default:
		return "", fmt.Errorf("unsupported github context URL format %q", contextURL)
	}

	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("unsupported github repo path %q", path)
	}
	dockerfilePath = strings.TrimPrefix(dockerfilePath, "/")
	if dockerfilePath == "" {
		return "", fmt.Errorf("VMDOCKER_BUILD_DOCKERFILE_PATH cannot be empty")
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", parts[0], parts[1], ref, dockerfilePath), nil
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

func buildArtifactContext(sourceDir string) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "vmdocker-artifact-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	appDir := filepath.Join(tempDir, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		cleanup()
		return "", nil, err
	}

	goos := GetEnvWith("VMDOCKER_ARTIFACT_GOOS", "linux")
	goarch := GetEnvWith("VMDOCKER_ARTIFACT_GOARCH", "arm64")
	if err := goBuildArtifact(sourceDir, filepath.Join(appDir, "main"), ".", goos, goarch); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := goBuildArtifact(sourceDir, filepath.Join(appDir, "bootstrap"), "./cmd/bootstrap", goos, goarch); err != nil {
		cleanup()
		return "", nil, err
	}

	for _, file := range []string{"start-vmdocker-agent.sh", "openclaw.default.json", "sandbox-profile.sh"} {
		if err := copyArtifactFile(filepath.Join(sourceDir, file), filepath.Join(tempDir, file)); err != nil {
			cleanup()
			return "", nil, err
		}
	}
	return tempDir, cleanup, nil
}

func goBuildArtifact(workdir, outputPath, pkg, goos, goarch string) error {
	args := []string{"build", "-o", outputPath, pkg}
	cmd := exec.Command("go", args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+goos,
		"GOARCH="+goarch,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s failed: %w\n%s", pkg, err, string(output))
	}
	return nil
}

func copyArtifactFile(src, dest string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dest, content, info.Mode()); err != nil {
		return err
	}
	return nil
}

func artifactRuntimeDockerfile() string {
	return strings.TrimSpace(`
FROM ghcr.io/openclaw/openclaw:latest AS openclaw

FROM docker/sandbox-templates:shell

USER root
WORKDIR /app

COPY --from=openclaw /usr/local/ /usr/local/
COPY --from=openclaw /app/ /app/
COPY app/main /app/main
COPY app/bootstrap /app/bootstrap
COPY start-vmdocker-agent.sh /usr/local/bin/start-vmdocker-agent.sh
COPY sandbox-profile.sh /etc/profile.d/vmdocker-sandbox.sh
COPY openclaw.default.json /app/openclaw.default.json

RUN set -eux; \
    if command -v apt-get >/dev/null 2>&1; then \
        apt-get update; \
        DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends curl bash ca-certificates; \
        rm -rf /var/lib/apt/lists/*; \
    elif command -v apk >/dev/null 2>&1; then \
        apk add --no-cache curl bash ca-certificates; \
    elif command -v microdnf >/dev/null 2>&1; then \
        microdnf install -y curl bash ca-certificates; \
        microdnf clean all; \
    fi

RUN set -eux; \
    usermod -G agent agent; \
    gpasswd -d agent sudo || true; \
    gpasswd -d agent docker || true; \
    rm -f /etc/sudoers.d/agent

RUN set -eux; \
    chmod +x /usr/local/bin/start-vmdocker-agent.sh /app/main /app/bootstrap /app/openclaw.mjs /etc/profile.d/vmdocker-sandbox.sh; \
    if [ -f /etc/profile ] && ! grep -q 'vmdocker-sandbox.sh' /etc/profile; then \
        printf '\n[ -f /etc/profile.d/vmdocker-sandbox.sh ] && . /etc/profile.d/vmdocker-sandbox.sh\n' >> /etc/profile; \
    fi; \
    rm -rf /home/agent/workspace; \
    chown -R agent:agent /app /usr/local/bin/start-vmdocker-agent.sh /etc/profile.d/vmdocker-sandbox.sh

RUN set -eux; \
    test ! -e /etc/sudoers.d/agent; \
    if command -v sudo >/dev/null 2>&1; then \
        su -s /bin/sh agent -c '! sudo -n true'; \
    fi

ENV VMDOCKER_AGENT_APP_ROOT=/app
ENV RUNTIME_TYPE=openclaw
ENV OPENCLAW_GATEWAY_PORT=18789
ENV OPENCLAW_GATEWAY_BIND=loopback
ENV OPENCLAW_GATEWAY_URL=http://127.0.0.1:18789
ENV OPENCLAW_CONFIG_TEMPLATE_PATH=/app/openclaw.default.json
ENV OPENCLAW_TIMEOUT_MS=30000
ENV HOME=/home/agent
ENV NODE_DISABLE_COMPILE_CACHE=1
ENV NODE_OPTIONS=--use-env-proxy

USER agent
WORKDIR /workspace
`) + "\n"
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
