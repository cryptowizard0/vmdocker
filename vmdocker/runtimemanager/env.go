package runtimemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cryptowizard0/vmdocker/vmdocker/utils"
)

const (
	openclawStateDirName  = ".openclaw"
	openclawConfigFile    = "openclaw.json"
	openclawWorkspaceDir  = "workspace"
	sandboxHomeDirName    = ".home"
	sandboxTmpDirName     = ".tmp"
	sandboxXDGDirName     = ".xdg"
	envOpenclawHome       = "OPENCLAW_HOME"
	envOpenclawStateDir   = "OPENCLAW_STATE_DIR"
	envOpenclawConfigPath = "OPENCLAW_CONFIG_PATH"
	envOpenclawWorkspace  = "OPENCLAW_AGENT_WORKSPACE"
	envHome               = "HOME"
	envTmpDir             = "TMPDIR"
	envXDGConfigHome      = "XDG_CONFIG_HOME"
	envXDGCacheHome       = "XDG_CACHE_HOME"
	envXDGStateHome       = "XDG_STATE_HOME"
	runtimeWorkspaceDir   = "sandbox_workspace"
)

func ensureRuntimeWorkspace(pid, root string) (string, error) {
	workspace, err := resolveRuntimeWorkspace(pid, root)
	if err != nil {
		return "", err
	}
	if err := ensureRuntimeWorkspaceLayout(workspace); err != nil {
		return "", err
	}
	return workspace, nil
}

func ensureRuntimeWorkspaceRoot(pid, root string) (string, error) {
	workspace, err := resolveRuntimeWorkspace(pid, root)
	if err != nil {
		return "", err
	}
	if err := ensureRuntimeWorkspaceDirs([]string{workspace}); err != nil {
		return "", err
	}
	return workspace, nil
}

func ensureRuntimeWorkspaceLayout(workspace string) error {
	return ensureRuntimeWorkspaceDirs(runtimeWorkspaceLayoutDirs(workspace))
}

func resolveRuntimeWorkspace(pid, root string) (string, error) {
	var err error
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			return "", err
		}
	} else {
		root, err = filepath.Abs(root)
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(root, runtimeWorkspaceDir, pid), nil
}

func appendRuntimePersistenceEnv(runtimeEnv []string, workspace string) []string {
	env := append([]string(nil), runtimeEnv...)
	if workspace == "" {
		return env
	}

	stateDir := envValue(env, envOpenclawStateDir, filepath.Join(workspace, openclawStateDirName))
	agentWorkspace := envValue(env, envOpenclawWorkspace, filepath.Join(stateDir, openclawWorkspaceDir))
	homeDir := envValue(env, envHome, filepath.Join(workspace, sandboxHomeDirName))
	tmpDir := envValue(env, envTmpDir, filepath.Join(workspace, sandboxTmpDirName))
	xdgConfigHome := envValue(env, envXDGConfigHome, filepath.Join(workspace, sandboxXDGDirName, "config"))
	xdgCacheHome := envValue(env, envXDGCacheHome, filepath.Join(workspace, sandboxXDGDirName, "cache"))
	xdgStateHome := envValue(env, envXDGStateHome, filepath.Join(workspace, sandboxXDGDirName, "state"))

	if !hasEnvKey(env, envOpenclawStateDir) {
		env = append(env, envOpenclawStateDir+"="+stateDir)
	}
	if !hasEnvKey(env, envOpenclawHome) {
		env = append(env, envOpenclawHome+"="+workspace)
	}
	if !hasEnvKey(env, envOpenclawConfigPath) {
		env = append(env, envOpenclawConfigPath+"="+filepath.Join(stateDir, openclawConfigFile))
	}
	if !hasEnvKey(env, envOpenclawWorkspace) {
		env = append(env, envOpenclawWorkspace+"="+agentWorkspace)
	}
	if !hasEnvKey(env, envHome) {
		env = append(env, envHome+"="+homeDir)
	}
	if !hasEnvKey(env, envTmpDir) {
		env = append(env, envTmpDir+"="+tmpDir)
	}
	if !hasEnvKey(env, envXDGConfigHome) {
		env = append(env, envXDGConfigHome+"="+xdgConfigHome)
	}
	if !hasEnvKey(env, envXDGCacheHome) {
		env = append(env, envXDGCacheHome+"="+xdgCacheHome)
	}
	if !hasEnvKey(env, envXDGStateHome) {
		env = append(env, envXDGStateHome+"="+xdgStateHome)
	}
	return env
}

func runtimeWorkspaceLayoutDirs(workspace string) []string {
	return []string{
		workspace,
		filepath.Join(workspace, openclawStateDirName),
		filepath.Join(workspace, openclawStateDirName, openclawWorkspaceDir),
		filepath.Join(workspace, sandboxHomeDirName),
		filepath.Join(workspace, sandboxTmpDirName),
		filepath.Join(workspace, sandboxXDGDirName, "config"),
		filepath.Join(workspace, sandboxXDGDirName, "cache"),
		filepath.Join(workspace, sandboxXDGDirName, "state"),
	}
}

func ensureRuntimeWorkspaceDirs(dirs []string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o777); err != nil {
			return fmt.Errorf("create runtime workspace dir %s failed: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o777); err != nil {
			return fmt.Errorf("chmod runtime workspace dir %s failed: %w", dir, err)
		}
	}
	return nil
}

func checkpointRuntimeWorkspace(workspace string) (string, error) {
	if strings.TrimSpace(workspace) == "" {
		return "", fmt.Errorf("runtime workspace is empty")
	}
	return utils.CompressDirectory(workspace)
}

func normalizeRuntimeWorkspacePath(workspace string) (string, error) {
	cleanedWorkspace, err := filepath.Abs(filepath.Clean(workspace))
	if err != nil {
		return "", err
	}
	if cleanedWorkspace == string(os.PathSeparator) {
		return "", fmt.Errorf("refusing to use root path as runtime workspace")
	}
	return cleanedWorkspace, nil
}

func stageRuntimeWorkspaceRestore(workspace, snapshot string) (string, func(), error) {
	cleanedWorkspace, err := normalizeRuntimeWorkspacePath(workspace)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(snapshot) == "" {
		return "", nil, fmt.Errorf("runtime workspace snapshot is empty")
	}

	parentDir := filepath.Dir(cleanedWorkspace)
	prefix := filepath.Base(cleanedWorkspace) + ".restore-"
	stagedWorkspace, err := os.MkdirTemp(parentDir, prefix)
	if err != nil {
		return "", nil, fmt.Errorf("create staged runtime workspace failed: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(stagedWorkspace)
	}

	if err := utils.DecompressToDirectory(snapshot, stagedWorkspace); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("restore staged runtime workspace %s failed: %w", stagedWorkspace, err)
	}
	if err := ensureRuntimeWorkspaceLayout(stagedWorkspace); err != nil {
		cleanup()
		return "", nil, err
	}
	return stagedWorkspace, cleanup, nil
}

// StageRuntimeWorkspaceRestore validates and unpacks a workspace snapshot into a temporary sibling directory.
// The caller owns the returned cleanup function and should remove the staged directory if it is not promoted.
func StageRuntimeWorkspaceRestore(workspace, snapshot string) (string, func(), error) {
	return stageRuntimeWorkspaceRestore(workspace, snapshot)
}

func restoreRuntimeWorkspace(workspace, snapshot string) error {
	stagedWorkspace, cleanup, err := stageRuntimeWorkspaceRestore(workspace, snapshot)
	if err != nil {
		return err
	}
	defer cleanup()

	rollback, commit, err := promoteRuntimeWorkspace(workspace, stagedWorkspace)
	if err != nil {
		return err
	}
	if err := commit(); err != nil {
		_ = rollback()
		return err
	}
	return nil
}

func promoteRuntimeWorkspace(workspace, stagedWorkspace string) (rollback func() error, commit func() error, err error) {
	cleanedWorkspace, err := normalizeRuntimeWorkspacePath(workspace)
	if err != nil {
		return nil, nil, err
	}
	cleanedStagedWorkspace, err := normalizeRuntimeWorkspacePath(stagedWorkspace)
	if err != nil {
		return nil, nil, err
	}
	parentDir := filepath.Dir(cleanedWorkspace)
	backupWorkspace := filepath.Join(parentDir, fmt.Sprintf("%s.backup-%d", filepath.Base(cleanedWorkspace), time.Now().UnixNano()))
	hasBackup := false

	if _, err := os.Stat(cleanedWorkspace); err == nil {
		if err := os.Rename(cleanedWorkspace, backupWorkspace); err != nil {
			return nil, nil, fmt.Errorf("backup runtime workspace %s failed: %w", cleanedWorkspace, err)
		}
		hasBackup = true
	} else if !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("inspect runtime workspace %s failed: %w", cleanedWorkspace, err)
	}

	if err := os.Rename(cleanedStagedWorkspace, cleanedWorkspace); err != nil {
		if hasBackup {
			_ = os.Rename(backupWorkspace, cleanedWorkspace)
		}
		return nil, nil, fmt.Errorf("activate staged runtime workspace %s failed: %w", cleanedStagedWorkspace, err)
	}

	rollback = func() error {
		if err := os.RemoveAll(cleanedWorkspace); err != nil {
			return fmt.Errorf("remove activated runtime workspace %s failed: %w", cleanedWorkspace, err)
		}
		if hasBackup {
			if err := os.Rename(backupWorkspace, cleanedWorkspace); err != nil {
				return fmt.Errorf("restore runtime workspace backup %s failed: %w", backupWorkspace, err)
			}
		}
		return nil
	}

	commit = func() error {
		if !hasBackup {
			return nil
		}
		if err := os.RemoveAll(backupWorkspace); err != nil {
			return fmt.Errorf("remove runtime workspace backup %s failed: %w", backupWorkspace, err)
		}
		return nil
	}
	return rollback, commit, nil
}

// PromoteRuntimeWorkspace swaps a staged workspace into the active location.
// The caller should invoke commit after the restored runtime is healthy, or rollback on failure.
func PromoteRuntimeWorkspace(workspace, stagedWorkspace string) (rollback func() error, commit func() error, err error) {
	return promoteRuntimeWorkspace(workspace, stagedWorkspace)
}

func hasEnvKey(env []string, key string) bool {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}
	return false
}

func envValue(env []string, key, fallback string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return fallback
}
