package runtimemanager

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	"github.com/stretchr/testify/assert"
)

func waitForEnter(step string) {
	fmt.Printf("=== Step Completed: %s ===\n\n", step)
}

func requireDockerIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("VMDOCKER_INTEGRATION") != "1" {
		t.Skip("set VMDOCKER_INTEGRATION=1 to run docker integration tests")
	}
}

func TestDockerManager(t *testing.T) {
	requireDockerIntegration(t)
	ctx := context.Background()

	dm, err := GetDockerManager()
	assert.NoError(t, err)
	assert.NotNil(t, dm)
	waitForEnter("Initialize DockerManager")

	t.Run("CreateContainer", func(t *testing.T) {
		requireDockerIntegration(t)
		imageInfo := schema.ImageInfo{
			Name: os.Getenv("VMDOCKER_TEST_IMAGE"),
			SHA:  os.Getenv("VMDOCKER_TEST_IMAGE_SHA"),
		}
		if imageInfo.Name == "" {
			imageInfo.Name = "chriswebber/docker-golua:v0.0.2"
		}
		instanceInfo, err := dm.CreateInstance(ctx, "test-container", schema.RuntimeSpec{
			Backend: schema.BackendDocker,
			Image:   imageInfo,
		}, nil)
		if !assert.NoError(t, err) {
			return
		}
		if !assert.NotNil(t, instanceInfo) {
			return
		}
		assert.Equal(t, "test-container", instanceInfo.Name)
		assert.Equal(t, "created", instanceInfo.Status)
		waitForEnter("Create Container")
	})

	t.Run("GetContainer", func(t *testing.T) {
		requireDockerIntegration(t)
		instanceInfo, err := dm.GetInstance("test-container")
		if !assert.NoError(t, err) {
			return
		}
		if !assert.NotNil(t, instanceInfo) {
			return
		}
		assert.Equal(t, "test-container", instanceInfo.Name)
		waitForEnter("Get Container Info")
	})

	t.Run("StartContainer", func(t *testing.T) {
		requireDockerIntegration(t)
		err := dm.StartInstance(ctx, "test-container")
		if !assert.NoError(t, err) {
			return
		}

		instanceInfo, err := dm.GetInstance("test-container")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, "running", instanceInfo.Status)
		waitForEnter("Start Container")
		time.Sleep(2 * time.Second)
	})

	t.Run("StopContainer", func(t *testing.T) {
		requireDockerIntegration(t)
		err := dm.StopInstance(ctx, "test-container")
		if !assert.NoError(t, err) {
			return
		}

		instanceInfo, err := dm.GetInstance("test-container")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, "stopped", instanceInfo.Status)
		waitForEnter("Stop Container")
	})

	t.Run("RemoveContainer", func(t *testing.T) {
		requireDockerIntegration(t)
		err := dm.RemoveInstance(ctx, "test-container")
		if !assert.NoError(t, err) {
			return
		}

		_, err = dm.GetInstance("test-container")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "instance not found")
		waitForEnter("Remove Container")
	})

	t.Run("ErrorCases", func(t *testing.T) {
		requireDockerIntegration(t)
		_, err := dm.GetInstance("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "instance not found")

		err = dm.StopInstance(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "instance not found")

		err = dm.RemoveInstance(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "instance not found")
		waitForEnter("Error Test Cases")
	})

	t.Run("CheckpointAndRestoreNotSupported", func(t *testing.T) {
		requireDockerIntegration(t)
		imageInfo := schema.ImageInfo{
			Name: os.Getenv("VMDOCKER_TEST_IMAGE"),
			SHA:  os.Getenv("VMDOCKER_TEST_IMAGE_SHA"),
		}
		if imageInfo.Name == "" {
			imageInfo.Name = "chriswebber/docker-golua:v0.0.2"
		}
		_, err := dm.CreateInstance(ctx, "checkpoint-test", schema.RuntimeSpec{
			Backend: schema.BackendDocker,
			Image:   imageInfo,
		}, nil)
		if !assert.NoError(t, err) {
			return
		}
		err = dm.StartInstance(ctx, "checkpoint-test")
		if !assert.NoError(t, err) {
			return
		}
		err = dm.StopInstance(ctx, "checkpoint-test")
		assert.NoError(t, err)

		_, err = dm.Checkpoint(ctx, "checkpoint-test", "test-checkpoint")
		assert.ErrorIs(t, err, schema.ErrNotSupported)
		err = dm.Restore(ctx, "checkpoint-test", "test-checkpoint", "snapshot")
		assert.ErrorIs(t, err, schema.ErrNotSupported)

		err = dm.RemoveInstance(ctx, "checkpoint-test")
		assert.NoError(t, err)
	})
}

func TestEnsureImageExists(t *testing.T) {
	requireDockerIntegration(t)
	ctx := context.Background()

	dm, err := GetDockerManager()
	assert.NoError(t, err)
	assert.NotNil(t, dm)

	dockerManager := dm.(*DockerManager)

	t.Run("EnsureImageExistsWithValidImage", func(t *testing.T) {
		requireDockerIntegration(t)
		imageInfo := schema.ImageInfo{
			Name: os.Getenv("VMDOCKER_TEST_IMAGE"),
			SHA:  os.Getenv("VMDOCKER_TEST_IMAGE_SHA"),
		}
		if imageInfo.Name == "" {
			imageInfo.Name = "chriswebber/docker-golua:v0.0.2"
		}

		err := dockerManager.ensureImageExists(ctx, imageInfo)
		assert.NoError(t, err)
		waitForEnter("Ensure Image Exists - Valid Image")
	})

	t.Run("EnsureImageExistsWithInvalidSHA", func(t *testing.T) {
		requireDockerIntegration(t)
		imageInfo := schema.ImageInfo{
			Name: "alpine:latest",
			SHA:  "sha256:invalid-sha-that-will-not-match",
		}

		err := dockerManager.ensureImageExists(ctx, imageInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "image SHA verification failed")
		waitForEnter("Ensure Image Exists - Invalid SHA")
	})
}

func TestVerifyImageSHA(t *testing.T) {
	requireDockerIntegration(t)
	ctx := context.Background()

	dm, err := GetDockerManager()
	assert.NoError(t, err)
	assert.NotNil(t, dm)

	dockerManager := dm.(*DockerManager)

	t.Run("VerifyImageSHAWithExistingImage", func(t *testing.T) {
		requireDockerIntegration(t)
		imageInfo := schema.ImageInfo{
			Name: os.Getenv("VMDOCKER_TEST_IMAGE"),
			SHA:  os.Getenv("VMDOCKER_TEST_IMAGE_SHA"),
		}
		if imageInfo.Name == "" {
			imageInfo.Name = "chriswebber/docker-golua:v0.0.2"
		}
		err := dockerManager.ensureImageExists(ctx, imageInfo)
		assert.NoError(t, err)

		imageInfo.SHA = "sha256:dummy-sha-that-will-not-match"
		err = dockerManager.verifyImageSHA(ctx, imageInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "image SHA verification failed")
		waitForEnter("Verify Image SHA - Existing Image with Wrong SHA")
	})

	t.Run("VerifyImageSHAWithNonExistentImage", func(t *testing.T) {
		requireDockerIntegration(t)
		imageInfo := schema.ImageInfo{
			Name: "nonexistent/invalid-image:nonexistent-tag",
			SHA:  "sha256:dummy-sha",
		}

		err := dockerManager.verifyImageSHA(ctx, imageInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to inspect image")
		waitForEnter("Verify Image SHA - Non-existent Image")
	})
}

func TestImageInspectID(t *testing.T) {
	requireDockerIntegration(t)
	ctx := context.Background()

	dm, err := GetDockerManager()
	assert.NoError(t, err)
	assert.NotNil(t, dm)

	dockerManager := dm.(*DockerManager)

	t.Run("GetAlpineImageID", func(t *testing.T) {
		requireDockerIntegration(t)
		imageInfo := schema.ImageInfo{
			Name: "chriswebber/docker-golua:latest",
			SHA:  "b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b",
		}

		inspect, err := dockerManager.cli.ImageInspect(ctx, imageInfo.Name)
		assert.NoError(t, err)
		t.Logf("Image Name: %s", imageInfo.Name)
		t.Logf("Image ID: %s", inspect.ID)
		t.Logf("Image Size: %d bytes", inspect.Size)
		t.Logf("Image Created: %s", inspect.Created)
		t.Logf("Image Architecture: %s", inspect.Architecture)
		t.Logf("Image OS: %s", inspect.Os)
		assert.NotEmpty(t, inspect.ID)
		assert.True(t, strings.HasPrefix(inspect.ID, "sha256:"))
		waitForEnter("Alpine Image Inspect ID")
	})
}
