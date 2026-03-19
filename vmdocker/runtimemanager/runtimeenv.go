package runtimemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
