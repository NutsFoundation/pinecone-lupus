package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/NutsFoundation/pinecone-lupus/etherium/mushroom/php/arch"
	"github.com/NutsFoundation/pinecone-lupus/etherium/mushroom/php/arch/entity"
)

type RuntimeManager struct {
	config   entity.MushroomConfig
	resolver arch.ExecutableResolver

	mu            sync.Mutex
	cmd           *exec.Cmd
	startedByHost bool
}

var _ arch.RuntimeManager = (*RuntimeManager)(nil)

func NewRuntimeManager(config entity.MushroomConfig, resolver arch.ExecutableResolver) *RuntimeManager {
	return &RuntimeManager{
		config:   normalizeConfig(config),
		resolver: resolver,
	}
}

func (m *RuntimeManager) EnsureReady(ctx context.Context) error {
	if !m.config.Runtime.Enable {
		return errors.New("PHP_MUSHROOM_DISABLED")
	}
	if m.isReady(ctx) {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isReady(ctx) {
		return nil
	}
	if !m.config.Runtime.LazyStart && m.cmd == nil {
		return errors.New("PHP_MUSHROOM_RUNTIME_NOT_STARTED")
	}
	if err := m.startLocked(); err != nil {
		return err
	}
	return m.waitUntilReady(ctx)
}

func (m *RuntimeManager) Shutdown(ctx context.Context) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.startedByHost || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	err := m.cmd.Process.Kill()
	_, _ = m.cmd.Process.Wait()
	m.cmd = nil
	m.startedByHost = false
	return err
}

func (m *RuntimeManager) startLocked() error {
	if m.cmd != nil && m.cmd.Process != nil {
		if m.cmd.ProcessState == nil || !m.cmd.ProcessState.Exited() {
			return nil
		}
		m.cmd = nil
	}

	executable, workDir, err := m.resolveExecutable()
	if err != nil {
		return err
	}
	if strings.TrimSpace(executable) == "" {
		return errors.New("PHP_MUSHROOM_CGI_EXECUTABLE_REQUIRED")
	}
	if _, err := os.Stat(executable); err != nil {
		return fmt.Errorf("PHP_MUSHROOM_CGI_NOT_FOUND: %w", err)
	}

	address := net.JoinHostPort(m.config.Runtime.Host, fmt.Sprintf("%d", m.config.Runtime.Port))
	command := exec.Command(executable, "-b", address)
	command.Dir = workDir
	command.Env = os.Environ()
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{}
	if err := command.Start(); err != nil {
		return fmt.Errorf("PHP_MUSHROOM_START_FAILED: %w", err)
	}

	m.cmd = command
	m.startedByHost = true
	return nil
}

func (m *RuntimeManager) resolveExecutable() (string, string, error) {
	if strings.TrimSpace(m.config.Runtime.CGIExecutable) != "" && filepath.IsAbs(m.config.Runtime.CGIExecutable) {
		executable := m.config.Runtime.CGIExecutable
		return executable, filepath.Dir(executable), nil
	}
	if m.resolver == nil {
		return "", "", errors.New("PHP_MUSHROOM_EXECUTABLE_RESOLVER_NIL")
	}
	spec, err := m.resolver.ResolvePHPFastCGI(m.config.Runtime.ToolchainName)
	if err != nil {
		return "", "", err
	}
	if spec == nil {
		return "", "", errors.New("PHP_MUSHROOM_TOOLCHAIN_NOT_FOUND")
	}

	executable := strings.TrimSpace(spec.ExecutablePath)
	if strings.TrimSpace(m.config.Runtime.CGIExecutable) != "" {
		candidate := m.config.Runtime.CGIExecutable
		if !filepath.IsAbs(candidate) {
			if strings.TrimSpace(spec.InstallPath) == "" {
				return "", "", errors.New("PHP_MUSHROOM_CGI_EXECUTABLE_RELATIVE_WITHOUT_INSTALL_PATH")
			}
			candidate = filepath.Join(spec.InstallPath, candidate)
		}
		executable = candidate
	}
	workDir := resolveRuntimeWorkingDir(spec, executable)
	return executable, workDir, nil
}

func (m *RuntimeManager) waitUntilReady(ctx context.Context) error {
	timeout := time.Duration(m.config.Runtime.StartupTimeoutMillis) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	waitCtx := nonNilContext(ctx)
	var cancel context.CancelFunc
	if _, ok := waitCtx.Deadline(); !ok {
		waitCtx, cancel = context.WithTimeout(waitCtx, timeout)
		defer cancel()
	}
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		if m.isReady(waitCtx) {
			return nil
		}
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("PHP_MUSHROOM_STARTUP_TIMEOUT: %w", waitCtx.Err())
		case <-ticker.C:
			if m.cmd != nil && m.cmd.ProcessState != nil && m.cmd.ProcessState.Exited() {
				return errors.New("PHP_MUSHROOM_PROCESS_EXITED")
			}
		}
	}
}

func (m *RuntimeManager) isReady(ctx context.Context) bool {
	dialer := net.Dialer{Timeout: 500 * time.Millisecond}
	conn, err := dialer.DialContext(nonNilContext(ctx), "tcp", net.JoinHostPort(m.config.Runtime.Host, fmt.Sprintf("%d", m.config.Runtime.Port)))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func resolveRuntimeWorkingDir(spec *arch.ExecutableSpec, executable string) string {
	if spec != nil {
		if strings.TrimSpace(spec.WorkPath) != "" {
			return spec.WorkPath
		}
		if strings.TrimSpace(spec.InstallPath) != "" {
			return spec.InstallPath
		}
	}
	return filepath.Dir(executable)
}
