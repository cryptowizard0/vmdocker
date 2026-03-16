package runtimemanager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	vmdockerUtils "github.com/cryptowizard0/vmdocker/vmdocker/utils"

	"github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
)

const buildCacheLabel = "io.hymx.vmdocker.build-cache-key"

// buildImageFromSpec builds a Docker image from the module-embedded build spec.
// The build cache is keyed by the module build inputs rather than just the tag,
// so updating the module forces a rebuild even if the tag is reused locally.
func buildImageFromSpec(ctx context.Context, cliBin string, spec *schema.BuildSpec) error {
	log.Info("building docker image from module build spec", "tag", spec.Tag, "context_url", spec.ContextURL, "has_context_archive", spec.ContextArchive != "")

	cacheKey := buildSpecCacheKey(spec)
	existingCacheKey, err := inspectImageLabel(ctx, cliBin, spec.Tag, buildCacheLabel)
	if err == nil && existingCacheKey == cacheKey {
		log.Info("image already matches module build spec, skipping build", "tag", spec.Tag)
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "vmdocker-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for build: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	contextPath, err := prepareBuildContext(tmpDir, spec)
	if err != nil {
		return err
	}

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(spec.Dockerfile), 0o600); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	args := []string{"build", "-f", dockerfilePath, "-t", spec.Tag, "--label", buildCacheLabel + "=" + cacheKey}
	for _, buildArg := range sortedBuildArgs(spec.Args) {
		args = append(args, "--build-arg", buildArg)
	}
	for _, proxyKey := range []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"} {
		if val := os.Getenv(proxyKey); val != "" {
			val = strings.ReplaceAll(val, "127.0.0.1", "host.docker.internal")
			val = strings.ReplaceAll(val, "localhost", "host.docker.internal")
			args = append(args, "--build-arg", proxyKey+"="+val)
		}
	}
	args = append(args, contextPath)

	log.Info("running docker build", "tag", spec.Tag, "args", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, cliBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed for %s: %w\n%s", spec.Tag, err, string(output))
	}

	log.Info("docker build completed", "tag", spec.Tag)
	return nil
}

func prepareBuildContext(tmpDir string, spec *schema.BuildSpec) (string, error) {
	switch {
	case spec.ContextURL != "":
		return spec.ContextURL, nil
	case spec.ContextArchive != "":
		if err := vmdockerUtils.DecompressToDirectory(spec.ContextArchive, tmpDir); err != nil {
			return "", fmt.Errorf("failed to extract build context archive: %w", err)
		}
		return tmpDir, nil
	default:
		return "", fmt.Errorf("build context missing for %s: provide Build-Context-URL or Build-Context-Archive", spec.Tag)
	}
}

func buildSpecCacheKey(spec *schema.BuildSpec) string {
	sum := sha256.New()
	sum.Write([]byte(spec.Dockerfile))
	sum.Write([]byte{0})
	sum.Write([]byte(spec.ContextURL))
	sum.Write([]byte{0})
	sum.Write([]byte(spec.ContextArchive))
	sum.Write([]byte{0})
	for _, buildArg := range sortedBuildArgs(spec.Args) {
		sum.Write([]byte(buildArg))
		sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func sortedBuildArgs(args map[string]string) []string {
	if len(args) == 0 {
		return nil
	}
	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	buildArgs := make([]string, 0, len(keys))
	for _, key := range keys {
		buildArgs = append(buildArgs, key+"="+args[key])
	}
	return buildArgs
}

func inspectImageLabel(ctx context.Context, cliBin, tag, label string) (string, error) {
	format := fmt.Sprintf("{{ index .Config.Labels %q }}", label)
	cmd := exec.CommandContext(ctx, cliBin, "image", "inspect", "--format", format, tag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
