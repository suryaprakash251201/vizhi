package process

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type AppManager struct {
	allowedApps map[string]string
	allowedUID  int
	allowedGID  int
	timeout     time.Duration
	dbusSession string
	xdgRuntime  string
}

type AppInfo struct {
	Name        string `json:"name"`
	Binary      string `json:"binary"`
	DisplayName string `json:"display_name"`
	Running     bool   `json:"running"`
	PID         int32  `json:"pid,omitempty"`
}

func NewAppManager(allowed []string, uid, gid int, timeout time.Duration) *AppManager {
	am := &AppManager{
		allowedApps: make(map[string]string, len(allowed)),
		allowedUID:  uid,
		allowedGID:  gid,
		timeout:     timeout,
	}
	for _, name := range allowed {
		am.allowedApps[name] = name
	}

	if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
		am.dbusSession = fmt.Sprintf("unix:path=/run/user/%d/bus", uid)
		am.xdgRuntime = fmt.Sprintf("/run/user/%d", uid)
		_ = u
	}

	return am
}

func (am *AppManager) IsAllowed(binary string) bool {
	_, ok := am.allowedApps[strings.TrimSpace(binary)]
	return ok
}

func (am *AppManager) AllowedApps() []string {
	names := make([]string, 0, len(am.allowedApps))
	for name := range am.allowedApps {
		names = append(names, name)
	}
	return names
}

func (am *AppManager) ListApps(ctx context.Context) ([]AppInfo, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}

	runningMap := make(map[string]struct{})
	for _, p := range procs {
		name, err := p.NameWithContext(ctx)
		if err != nil || name == "" {
			continue
		}
		if _, allowed := am.allowedApps[name]; allowed {
			runningMap[name] = struct{}{}
		}
	}

	apps := make([]AppInfo, 0, len(am.allowedApps))
	for binary, display := range am.allowedApps {
		_, running := runningMap[binary]
		app := AppInfo{
			Name:        binary,
			Binary:      binary,
			DisplayName: display,
			Running:     running,
		}
		if running {
			if pid, err := am.findPID(ctx, binary); err == nil {
				app.PID = pid
			}
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (am *AppManager) LaunchApp(ctx context.Context, binary string) error {
	if !am.IsAllowed(binary) {
		return fmt.Errorf("application %q is not in the allowlist", binary)
	}

	fullPath, err := exec.LookPath(binary)
	if err != nil {
		paths := []string{
			"/usr/bin/" + binary,
			"/usr/local/bin/" + binary,
			"/snap/bin/" + binary,
			"/var/lib/flatpak/exports/bin/" + binary,
		}
		for _, p := range paths {
			if fileExists(p) {
				fullPath = p
				break
			}
		}
	}
	if fullPath == "" {
		return fmt.Errorf("binary %q not found on host filesystem", binary)
	}

	cmd := exec.CommandContext(ctx, fullPath)
	am.setupHostEnv(cmd)
	cmd.SysProcAttr = detachAttr()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch %s: %w", binary, err)
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("process %s (PID %d) exited: %v", binary, cmd.Process.Pid, err)
		}
	}()

	return nil
}

func (am *AppManager) TerminateApp(ctx context.Context, binary string) error {
	if !am.IsAllowed(binary) {
		return fmt.Errorf("application %q is not in the allowlist", binary)
	}

	pid, err := am.findPID(ctx, binary)
	if err != nil {
		return fmt.Errorf("find PID for %s: %w", binary, err)
	}

	proc, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("access process %d: %w", pid, err)
	}

	if err := proc.TerminateWithContext(ctx); err != nil {
		return fmt.Errorf("terminate %s (PID %d): %w", binary, pid, err)
	}

	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			if err := proc.KillWithContext(ctx); err != nil {
				return fmt.Errorf("force kill %s (PID %d): %w", binary, pid, err)
			}
			return nil
		case <-ticker.C:
			running, err := proc.IsRunningWithContext(ctx)
			if err != nil || !running {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (am *AppManager) findPID(ctx context.Context, binary string) (int32, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("list processes: %w", err)
	}

	for _, p := range procs {
		name, err := p.NameWithContext(ctx)
		if err != nil {
			continue
		}
		if name == binary {
			return p.Pid, nil
		}
	}
	return 0, fmt.Errorf("no running process found for %s", binary)
}

func (am *AppManager) setupHostEnv(cmd *exec.Cmd) {
	if cmd.Env == nil {
		cmd.Env = []string{}
	}
	cmd.Env = append(cmd.Env,
		"DISPLAY=:0",
		fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=%s", am.dbusSession),
		fmt.Sprintf("XDG_RUNTIME_DIR=%s", am.xdgRuntime),
		fmt.Sprintf("HOME=/home/%s", am.lookupUsername()),
		"PATH=/usr/local/bin:/usr/bin:/bin:/snap/bin",
	)
}

func (am *AppManager) lookupUsername() string {
	if u, err := user.LookupId(strconv.Itoa(am.allowedUID)); err == nil {
		return u.Username
	}
	return "user"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
