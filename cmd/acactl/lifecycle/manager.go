package lifecycle

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type Service struct {
	Name    string
	Port    int
	Binary  string
	Args    []string
	Podman  bool // if true, use podman instead of binary
	cmd     *exec.Cmd
}

type ProcessManager struct {
	BinDir   string
	DataDir  string
	LogDir   string
	PidDir   string
	Services []*Service
}

func (pm *ProcessManager) BuildBinary(modulePath string, outputPath string) error {
	cmd := exec.Command("go", "build",
		"-o", outputPath,
		modulePath)
	cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (pm *ProcessManager) Start(svc *Service) error {
	if svc.Podman {
		return pm.startPodman(svc)
	}
	svc.cmd = exec.Command(svc.Binary, svc.Args...)
	logFile, _ := os.Create(pm.LogDir + "/" + svc.Name + ".log")
	svc.cmd.Stdout = logFile
	svc.cmd.Stderr = logFile
	if err := svc.cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", svc.Name, err)
	}
	WritePID(pm.PidDir, svc.Name, svc.cmd.Process.Pid)
	return nil
}

func (pm *ProcessManager) startPodman(svc *Service) error {
	// podman run -d --name <Name> --cap-add NET_RAW ... -p <Port>:<Port> <image>
	args := []string{"run", "-d", "--name", svc.Name,
		"--cap-add", "NET_RAW", "--cap-add", "NET_ADMIN",
		"--sysctl", "net.ipv6.conf.all.disable_ipv6=0",
		"-p", fmt.Sprintf("%d:%d", svc.Port, svc.Port),
		svc.Binary,
	}
	cmd := exec.Command("podman", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (pm *ProcessManager) Stop(svc *Service) error {
	if svc.Podman {
		exec.Command("podman", "stop", svc.Name).Run()
		exec.Command("podman", "rm", svc.Name).Run()
		return nil
	}
	pid, _ := ReadPID(pm.PidDir, svc.Name)
	if pid > 0 {
		p, _ := os.FindProcess(pid)
		p.Signal(os.Interrupt)
		time.Sleep(3 * time.Second)
		p.Kill()
	}
	RemovePID(pm.PidDir, svc.Name)
	return nil
}

func (pm *ProcessManager) HealthCheck(svc *Service) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/health", svc.Port)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	for ctx.Err() == nil {
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%s health check timeout", svc.Name)
}
