package vmdocker

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/schema"
	"github.com/stretchr/testify/assert"
)

func waitForEnter(step string) {
	fmt.Printf("=== Step Completed: %s ===\n\n", step)
}

// go test -run ^TestDockerManager$
func TestDockerManager(t *testing.T) {
	ctx := context.Background()

	// Get DockerManager instance
	dm, err := GetDockerManager()
	assert.NoError(t, err)
	assert.NotNil(t, dm)
	waitForEnter("Initialize DockerManager")

	// Test container creation
	t.Run("CreateContainer", func(t *testing.T) {
		imageInfo := schema.ImageInfo{
			Name: "chriswebber/docker-golua:v0.0.2",
			SHA:  "sha256:b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b",
		}
		containerInfo, err := dm.CreateContainer(ctx, "test-container", imageInfo)
		assert.NoError(t, err)
		assert.NotNil(t, containerInfo)
		assert.Equal(t, "test-container", containerInfo.Name)
		assert.Equal(t, "created", containerInfo.Status)
		waitForEnter("Create Container")
	})

	// Test get container info
	t.Run("GetContainer", func(t *testing.T) {
		containerInfo, err := dm.GetContainer("test-container")
		assert.NoError(t, err)
		assert.NotNil(t, containerInfo)
		assert.Equal(t, "test-container", containerInfo.Name)
		waitForEnter("Get Container Info")
	})

	// Test start container
	t.Run("StartContainer", func(t *testing.T) {
		err := dm.StartContainer(ctx, "test-container")
		assert.NoError(t, err)

		// Verify container status
		containerInfo, err := dm.GetContainer("test-container")
		assert.NoError(t, err)
		assert.Equal(t, "running", containerInfo.Status)
		waitForEnter("Start Container")

		// Wait for container to fully start
		time.Sleep(2 * time.Second)
	})

	// Test stop container
	t.Run("StopContainer", func(t *testing.T) {
		err := dm.StopContainer(ctx, "test-container")
		assert.NoError(t, err)

		// Verify container status
		containerInfo, err := dm.GetContainer("test-container")
		assert.NoError(t, err)
		assert.Equal(t, "stopped", containerInfo.Status)
		waitForEnter("Stop Container")
	})

	// Test remove container
	t.Run("RemoveContainer", func(t *testing.T) {
		err := dm.RemoveContainer(ctx, "test-container")
		assert.NoError(t, err)

		// Verify container is removed
		_, err = dm.GetContainer("test-container")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "container not found")
		waitForEnter("Remove Container")
	})

	// Test error cases
	t.Run("ErrorCases", func(t *testing.T) {
		// Test getting non-existent container
		_, err := dm.GetContainer("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "container not found")

		// Test stopping non-existent container
		err = dm.StopContainer(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "container not found")

		// Test removing non-existent container
		err = dm.RemoveContainer(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "container not found")
		waitForEnter("Error Test Cases")
	})

	// Test checkpoint and restore functionality
	// go test -run ^TestDockerManager$/^CheckpointAndRestore$
	t.Run("CheckpointAndRestore", func(t *testing.T) {
		// Create and start a container for testing
		imageInfo := schema.ImageInfo{
			Name: "chriswebber/docker-golua:v0.0.2",
			SHA:  "sha256:b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b",
		}
		containerInfo, err := dm.CreateContainer(ctx, "checkpoint-test", imageInfo)
		assert.NoError(t, err)
		assert.NotNil(t, containerInfo)

		err = dm.StartContainer(ctx, "checkpoint-test")
		assert.NoError(t, err)
		time.Sleep(2 * time.Second) // Wait for container to fully start
		waitForEnter("Container Started for Checkpoint Test")

		// Create a checkpoint of the running container
		zipdata, err := dm.Checkpoint(ctx, "checkpoint-test", "test-checkpoint")
		assert.NoError(t, err)
		waitForEnter("Checkpoint Created")

		// Stop the container to simulate a shutdown
		err = dm.StopContainer(ctx, "checkpoint-test")
		assert.NoError(t, err)

		// Restore the container from checkpoint
		err = dm.Restore(ctx, "checkpoint-test", "test-checkpoint", zipdata)
		assert.NoError(t, err)

		// Verify container status after restore
		containerInfo, err = dm.GetContainer("checkpoint-test")
		assert.NoError(t, err)

		// Cleanup test resources
		err = dm.RemoveContainer(ctx, "checkpoint-test")
		assert.NoError(t, err)

	})
}

// go test -run ^TestEnsureImageExists$
func TestEnsureImageExists(t *testing.T) {
	ctx := context.Background()

	// Get DockerManager instance
	dm, err := GetDockerManager()
	assert.NoError(t, err)
	assert.NotNil(t, dm)

	// Convert to concrete type to access ensureImageExists method
	dockerManager := dm.(*DockerManager)

	// Test case 1: Ensure image exists with valid image (should work with existing images)
	t.Run("EnsureImageExistsWithValidImage", func(t *testing.T) {
		// Use a lightweight, commonly available image for testing
		imageInfo := schema.ImageInfo{
			Name: "chriswebber/docker-golua:v0.0.2",
			SHA:  "sha256:b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b",
		}

		err := dockerManager.ensureImageExists(ctx, imageInfo)
		assert.NoError(t, err)
		waitForEnter("Ensure Image Exists - Valid Image")
	})

	// Test case 2: Ensure image exists with invalid SHA (should fail verification)
	t.Run("EnsureImageExistsWithInvalidSHA", func(t *testing.T) {
		imageInfo := schema.ImageInfo{
			Name: "alpine:latest",
			SHA:  "sha256:invalid-sha-that-will-not-match", // Invalid SHA should fail
		}

		err := dockerManager.ensureImageExists(ctx, imageInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "image SHA verification failed")
		waitForEnter("Ensure Image Exists - Invalid SHA")
	})
}

// go test -run ^TestVerifyImageSHA$
func TestVerifyImageSHA(t *testing.T) {
	ctx := context.Background()

	// Get DockerManager instance
	dm, err := GetDockerManager()
	assert.NoError(t, err)
	assert.NotNil(t, dm)

	// Convert to concrete type to access verifyImageSHA method
	dockerManager := dm.(*DockerManager)

	// Test case 1: Verify SHA with existing image (alpine:latest)
	t.Run("VerifyImageSHAWithExistingImage", func(t *testing.T) {
		// First ensure the image exists
		imageInfo := schema.ImageInfo{
			Name: "chriswebber/docker-golua:v0.0.2",
			SHA:  "sha256:b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b",
		}
		err := dockerManager.ensureImageExists(ctx, imageInfo)
		assert.NoError(t, err)

		// Now test SHA verification with a dummy SHA (should fail)
		imageInfo.SHA = "sha256:dummy-sha-that-will-not-match"
		err = dockerManager.verifyImageSHA(ctx, imageInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "image SHA verification failed")
		waitForEnter("Verify Image SHA - Existing Image with Wrong SHA")
	})

	// Test case 2: Verify SHA with non-existent image (should fail)
	t.Run("VerifyImageSHAWithNonExistentImage", func(t *testing.T) {
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

// go test -run ^TestImageInspectID$
func TestImageInspectID(t *testing.T) {
	ctx := context.Background()

	// Get DockerManager instance
	dm, err := GetDockerManager()
	assert.NoError(t, err)
	assert.NotNil(t, dm)

	// Convert to concrete type to access Docker client
	dockerManager := dm.(*DockerManager)

	// Test with alpine:latest image
	t.Run("GetAlpineImageID", func(t *testing.T) {
		// First ensure the image exists
		imageInfo := schema.ImageInfo{
			Name: "chriswebber/docker-golua:latest",
			SHA:  "b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b",
		}
		// err := dockerManager.ensureImageExists(ctx, imageInfo)
		// assert.NoError(t, err)

		// Get image inspect information
		inspect, err := dockerManager.cli.ImageInspect(ctx, imageInfo.Name)
		assert.NoError(t, err)

		// Print the image ID
		t.Logf("Image Name: %s", imageInfo.Name)
		t.Logf("Image ID: %s", inspect.ID)
		t.Logf("Image Size: %d bytes", inspect.Size)
		t.Logf("Image Created: %s", inspect.Created)
		t.Logf("Image Architecture: %s", inspect.Architecture)
		t.Logf("Image OS: %s", inspect.Os)

		// Verify that ID is not empty
		assert.NotEmpty(t, inspect.ID)
		// Verify that ID starts with sha256:
		assert.True(t, strings.HasPrefix(inspect.ID, "sha256:"))

		waitForEnter("Alpine Image Inspect ID")
	})
}
