package lifecycle

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

func CheckPort(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("port %d is in use", port)
	}
	ln.Close()
	return nil
}

func WritePID(pidDir, name string, pid int) error {
	if pidDir == "" {
		home, _ := os.UserHomeDir()
		pidDir = home + "/.aca/pids"
	}
	os.MkdirAll(pidDir, 0755)
	return os.WriteFile(pidDir+"/"+name+".pid", []byte(strconv.Itoa(pid)), 0644)
}

func ReadPID(pidDir, name string) (int, error) {
	if pidDir == "" {
		home, _ := os.UserHomeDir()
		pidDir = home + "/.aca/pids"
	}
	data, err := os.ReadFile(pidDir + "/" + name + ".pid")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

func RemovePID(pidDir, name string) error {
	if pidDir == "" {
		home, _ := os.UserHomeDir()
		pidDir = home + "/.aca/pids"
	}
	return os.Remove(pidDir + "/" + name + ".pid")
}
