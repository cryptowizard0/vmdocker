package vmdocker

import (
	"context"
	"fmt"
	"testing"
	"time"

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
		containerInfo, err := dm.CreateContainer(ctx, "test-container", "golua")
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
		containerInfo, err := dm.CreateContainer(ctx, "checkpoint-test", "golua")
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
