package vmdocker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/schema"
	"github.com/cryptowizard0/vmdocker/vmdocker/utils"
	"github.com/docker/docker/api/types/checkpoint"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	ContainerNamePrefix = "hymatrix_"
)

var (
	once     sync.Once
	instance schema.IDockerManager
)

// DockerManager handles all Docker operations
type DockerManager struct {
	cli           *client.Client
	containers    map[string]*schema.ContainerInfo // key -> ao process id
	mutex         sync.RWMutex
	portAllocator *PortAllocator
}

// ensureImageExists checks if image exists locally, pulls it if not, and verifies SHA
func (dm *DockerManager) ensureImageExists(ctx context.Context, imageInfo schema.ImageInfo) error {
	// Check if image exists locally
	_, err := dm.cli.ImageInspect(ctx, imageInfo.Name)
	if err == nil {
		// Image exists, verify SHA if provided
		if imageInfo.SHA != "" {
			return dm.verifyImageSHA(ctx, imageInfo)
		}
		return nil
	}

	// Image doesn't exist, pull it
	log.Info("pulling image", "image", imageInfo.Name)
	reader, err := dm.cli.ImagePull(ctx, imageInfo.Name, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %v", imageInfo.Name, err)
	}
	defer reader.Close()

	// Read pull output to ensure completion
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %v", err)
	}

	log.Info("image pulled successfully", "image", imageInfo.Name)

	// Verify SHA if provided
	if imageInfo.SHA != "" {
		return dm.verifyImageSHA(ctx, imageInfo)
	}

	return nil
}

// verifyImageSHA verifies the SHA256 digest of the pulled image
func (dm *DockerManager) verifyImageSHA(ctx context.Context, imageInfo schema.ImageInfo) error {
	inspect, err := dm.cli.ImageInspect(ctx, imageInfo.Name)
	if err != nil {
		return fmt.Errorf("failed to inspect image %s: %v", imageInfo.Name, err)
	}

	// Check RepoDigests for SHA verification
	for _, digest := range inspect.RepoDigests {
		if strings.Contains(digest, imageInfo.SHA) {
			log.Info("image SHA verified", "image", imageInfo.Name, "sha", imageInfo.SHA)
			return nil
		}
	}

	// If RepoDigests don't match, check image ID
	if inspect.ID == imageInfo.SHA {
		log.Info("image ID verified", "image", imageInfo.Name, "id", imageInfo.SHA)
		return nil
	}

	return fmt.Errorf("image SHA verification failed for %s: expected %s, got digests %v and ID %s",
		imageInfo.Name, imageInfo.SHA, inspect.RepoDigests, inspect.ID)
}

func newDockerManager() (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithVersion(schema.DockerVersion),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %v", err)
	}

	dm := &DockerManager{
		cli:           cli,
		containers:    make(map[string]*schema.ContainerInfo),
		portAllocator: NewPortAllocator(10000, 20000), // port range: start - end
	}

	return dm, nil
}

// GetDockerManager returns the docker manager instance
func GetDockerManager() (schema.IDockerManager, error) {
	var err error
	once.Do(func() {
		instance, err = newDockerManager()
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (dm *DockerManager) CreateContainer(ctx context.Context, pid string, imageInfo schema.ImageInfo) (*schema.ContainerInfo, error) {
	log.Debug("create container", "pid", pid)
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	// Ensure image exists locally, pull if necessary
	if err := dm.ensureImageExists(ctx, imageInfo); err != nil {
		log.Error("failed to ensure image exists", "image", imageInfo.Name, "err", err)
		return nil, fmt.Errorf("failed to ensure image exists: %v", err)
	}

	// Allocate port
	port, err := dm.portAllocator.Allocate()
	if err != nil {
		return nil, err
	}

	// Set port bindings
	pidsLimit := int64(256)
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(schema.ExprotPort): []nat.PortBinding{
				{
					HostIP:   schema.AllowHost,
					HostPort: fmt.Sprintf("%d", port),
				},
			},
		},
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     []string{"ALL"},
		Resources: container.Resources{
			Memory:     int64(schema.MaxMem),
			MemorySwap: -1, // no swap
			PidsLimit:  &pidsLimit,
			CPUPeriod:  100000, // 100ms
			CPUQuota:   50000,  // 0.5 CPU
			CPUShares:  1024,   // Standard weight
		},
	}
	if schema.UseMount {
		hostConfig.Mounts = []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: schema.MountSource,
				Target: schema.MountTarget,
			},
		}
	}
	config := &container.Config{
		Image: imageInfo.Name,
		User:  "65532:65532",
		ExposedPorts: nat.PortSet{
			nat.Port(schema.ExprotPort): struct{}{},
		},
	}
	resp, err := dm.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, ContainerNamePrefix+pid)
	if err != nil {
		log.Error("failed to create container", "pid", pid, "err", err)
		dm.portAllocator.Release(port)
		return nil, err
	}

	containerInfo := &schema.ContainerInfo{
		ID:       resp.ID,
		Name:     pid,
		Port:     port,
		Status:   "created",
		CreateAt: time.Now(),
	}
	dm.containers[pid] = containerInfo

	log.Debug("container created", "pid", pid, "container id", resp.ID)
	return containerInfo, nil
}

func (dm *DockerManager) GetContainer(pid string) (*schema.ContainerInfo, error) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	if container, exists := dm.containers[pid]; exists {
		return container, nil
	}
	return nil, fmt.Errorf("container not found")
}

func (dm *DockerManager) RemoveContainer(ctx context.Context, pid string) error {
	log.Debug("remove container", "pid", pid)

	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if ctner, exists := dm.containers[pid]; exists {
		err := dm.cli.ContainerRemove(ctx, ctner.ID, container.RemoveOptions{Force: true})
		if err != nil {
			log.Error("failed to remove container", "pid", pid, "container id", ctner.ID, "err", err)
			return err
		}

		dm.portAllocator.Release(ctner.Port)
		delete(dm.containers, pid)
		return nil
	}
	log.Error("container not found", "pid", pid)
	return fmt.Errorf("container not found: %s", pid)
}

func (dm *DockerManager) StartContainer(ctx context.Context, pid string) error {
	log.Debug("start container", "pid", pid)
	if ctner, exists := dm.containers[pid]; exists {
		err := dm.cli.ContainerStart(ctx, ctner.ID, container.StartOptions{})
		if err != nil {
			log.Error("failed to start container", "pid", pid, "container id", ctner.ID, "err", err)
			return err
		}
		ctner.Status = "running"
		log.Debug("container started", "pid", pid, "container id", ctner.ID)
		return nil
	}
	log.Error("container not found", "pid", pid)
	return fmt.Errorf("container not found: %s", pid)
}

func (dm *DockerManager) StopContainer(ctx context.Context, pid string) error {
	log.Debug("stop container", "pid", pid)
	if ctner, exists := dm.containers[pid]; exists {
		timeout := time.Second * 10
		timeoutSeconds := int(timeout.Seconds())
		err := dm.cli.ContainerStop(ctx, ctner.ID, container.StopOptions{Timeout: &timeoutSeconds})
		if err != nil {
			log.Error("failed to stop container", "pid", pid, "container id", ctner.ID, "err", err)
			return err
		}
		ctner.Status = "stopped"
		log.Debug("container stopped", "pid", pid, "container id", ctner.ID)
		return nil
	}
	log.Error("container not found", "pid", pid)
	return fmt.Errorf("container not found: %s", pid)
}

func (dm *DockerManager) Checkpoint(ctx context.Context, pid, checkpointName string) (zipCode string, err error) {
	log.Debug("create checkpoint", "pid", pid)
	if err = checkCheckpointRequirements(); err != nil {
		return "", err
	}

	var checkpointDir string
	checkpointDir, err = getCheckpointCacheDir(pid)
	if err != nil {
		log.Error("failed to get checkpoint cache directory", "pid", pid, "err", err)
		return "", err
	}
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		log.Error("failed to create checkpoint directory", "pid", pid, "err", err)
		return "", fmt.Errorf("failed to create checkpoint directory: %v", err)
	}

	ctner, exists := dm.containers[pid]
	if !exists {
		log.Error("container not found", "pid", pid)
		return "", fmt.Errorf("container not found: %s", pid)
	}

	createOptions := checkpoint.CreateOptions{
		CheckpointID:  checkpointName,
		CheckpointDir: checkpointDir,
		Exit:          false, // keep container running
	}

	if err := dm.cli.CheckpointCreate(ctx, ctner.ID, createOptions); err != nil {
		log.Error("failed to create checkpoint", "pid", pid, "container id", ctner.ID, "err", err)
		return "", err
	}

	// Compress checkpoint directory
	checkpointPath := filepath.Join(checkpointDir, checkpointName)
	compressedData, err := utils.CompressDirectory(checkpointPath)
	if err != nil {
		log.Error("failed to compress checkpoint", "pid", pid, "err", err)
		return "", fmt.Errorf("failed to compress checkpoint: %v", err)
	}

	log.Info("checkpoint created", "pid", pid, "container id", ctner.ID, "checkpoint name", checkpointName)
	return compressedData, nil

}

// Restore restores a container from a compressed checkpoint snapshot
// ctx: context for the operation
// pid: process ID of the container
// snapshot: compressed checkpoint data returned by Checkpoint function
func (dm *DockerManager) Restore(ctx context.Context, pid, checkpointName, snapshot string) error {
	log.Debug("restore container", "pid", pid)

	if err := checkCheckpointRequirements(); err != nil {
		return err
	}

	if ctner, exists := dm.containers[pid]; exists {
		// Extract the snapshot directly to docker checkpoint directory
		dockerCheckpointDir := fmt.Sprintf("/var/lib/docker/containers/%s/checkpoints", ctner.ID)
		checkpointPath := filepath.Join(dockerCheckpointDir, checkpointName)
		if err := utils.DecompressToDirectory(snapshot, checkpointPath); err != nil {
			log.Error("failed to decompress checkpoint", "pid", pid, "err", err)
			return fmt.Errorf("failed to decompress checkpoint: %v", err)
		}

		startOpts := container.StartOptions{
			CheckpointID: checkpointName,
		}

		if err := dm.cli.ContainerStart(ctx, ctner.ID, startOpts); err != nil {
			log.Error("failed to restore container", "pid", pid, "container id", ctner.ID, "err", err)
			return err
		}
		log.Info("container restored", "pid", pid, "container id", ctner.ID, "checkpoint name", checkpointName)
		return nil
	}
	log.Error("container not found", "pid", pid)
	return fmt.Errorf("container not found: %s", pid)
}

func getCheckpointCacheDir(pid string) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		log.Error("failed to get working directory", "err", err)
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s/", workDir, schema.CheckpointDir, pid), nil
}

// checkCheckpointRequirements checks if the system meets the requirements for checkpoint functionality
func checkCheckpointRequirements() error {
	// Check operating system
	if runtime.GOOS != "linux" {
		return fmt.Errorf("checkpoint only supports Linux, current system is %s", runtime.GOOS)
	}

	// Check CRIU version
	output, err := exec.Command("criu", "--version").Output()
	if err != nil {
		return fmt.Errorf("CRIU is not installed: %v", err)
	}
	version := strings.TrimSpace(string(output))
	log.Debug("CRIU version", "version", version)
	// if !strings.Contains(version, "4.1") {
	// 	return fmt.Errorf("CRIU version 4.1 is required, current version is %s", version)
	// }

	return nil
}
