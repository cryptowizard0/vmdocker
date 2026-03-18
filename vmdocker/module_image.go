package vmdocker

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	runtimeSchema "github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	goarSchema "github.com/permadao/goar/schema"
	goarUtils "github.com/permadao/goar/utils"
)

var dockerLookPath = exec.LookPath

func ensureModuleImageAvailable(ctx context.Context, moduleID string, image runtimeSchema.ImageInfo) error {
	if image.Source == "" && image.ArchiveFormat == "" {
		return nil
	}
	if image.Source != runtimeSchema.ImageSourceModuleData {
		return fmt.Errorf("unsupported image source %q", image.Source)
	}
	if image.ArchiveFormat != runtimeSchema.ImageArchiveDockerSaveGZ {
		return fmt.Errorf("unsupported image archive format %q", image.ArchiveFormat)
	}

	cliBin, err := dockerBinary()
	if err != nil {
		return err
	}

	matched, err := imageMatchesRef(ctx, cliBin, image.Name, image.SHA)
	if err == nil && matched {
		return nil
	}
	if err == nil && !matched {
		log.Info("local image tag exists but sha mismatched, reloading from module", "module", moduleID, "image", image.Name, "expected_sha", image.SHA)
	}

	if err := ensureImageTaggedByID(ctx, cliBin, image.SHA, image.Name); err == nil {
		return nil
	}

	payload, err := readModuleImagePayload(moduleID)
	if err != nil {
		return err
	}
	if err := dockerLoadArchive(ctx, cliBin, payload); err != nil {
		return err
	}
	if err := ensureImageTaggedByID(ctx, cliBin, image.SHA, image.Name); err != nil {
		return fmt.Errorf("loaded image from module %s but failed to tag/verify %s: %w", moduleID, image.Name, err)
	}
	return nil
}

func dockerBinary() (string, error) {
	cliBin, err := dockerLookPath("docker")
	if err != nil {
		return "", fmt.Errorf("docker CLI is not available: %w", err)
	}
	return cliBin, nil
}

func imageMatchesRef(ctx context.Context, cliBin, ref, expectedID string) (bool, error) {
	actualID, err := inspectLocalImageID(ctx, cliBin, ref)
	if err != nil {
		return false, err
	}
	return actualID == expectedID, nil
}

func inspectLocalImageID(ctx context.Context, cliBin, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, cliBin, "image", "inspect", "--format", "{{.Id}}", ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect image %s failed: %w: %s", ref, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func ensureImageTaggedByID(ctx context.Context, cliBin, imageID, imageName string) error {
	if _, err := inspectLocalImageID(ctx, cliBin, imageID); err != nil {
		return err
	}
	if matched, err := imageMatchesRef(ctx, cliBin, imageName, imageID); err == nil && matched {
		return nil
	}
	cmd := exec.CommandContext(ctx, cliBin, "image", "tag", imageID, imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tag image %s as %s failed: %w: %s", imageID, imageName, err, strings.TrimSpace(string(output)))
	}
	matched, err := imageMatchesRef(ctx, cliBin, imageName, imageID)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("image %s does not match expected id %s after tagging", imageName, imageID)
	}
	return nil
}

func readModuleImagePayload(moduleID string) ([]byte, error) {
	data, err := os.ReadFile(moduleFilePath(moduleID))
	if err != nil {
		return nil, fmt.Errorf("read module file %s failed: %w", moduleFilePath(moduleID), err)
	}

	var item goarSchema.BundleItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("decode module file %s failed: %w", moduleFilePath(moduleID), err)
	}
	if item.Data == "" {
		return nil, fmt.Errorf("module %s data is empty", moduleID)
	}

	payload, err := goarUtils.Base64Decode(item.Data)
	if err != nil {
		return nil, fmt.Errorf("decode module %s payload failed: %w", moduleID, err)
	}
	return payload, nil
}

func dockerLoadArchive(ctx context.Context, cliBin string, payload []byte) error {
	reader, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("open gzip payload failed: %w", err)
	}
	defer reader.Close()

	cmd := exec.CommandContext(ctx, cliBin, "image", "load")
	cmd.Stdin = reader
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker image load failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func moduleFilePath(moduleID string) string {
	return filepath.Join("mod", fmt.Sprintf("mod-%s.json", moduleID))
}
