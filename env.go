package server_register

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// getHostName : get the hostname of the host machine if the container is started by docker run --net=host
func getHostName() (string, error) {
	cmd := exec.Command("/bin/hostname")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	hostname := strings.TrimSpace(string(out))
	if hostname == "" {
		return "", fmt.Errorf("no hostname get from cmd '/bin/hostname' in the container, please check")
	}
	return hostname, nil
}

// GetHostName : get the hostname of host machine
func GetHostName() (string, error) {
	hostName := os.Getenv("HOST_NAME")
	if hostName != "" {
		return hostName, nil
	}
	fmt.Println("get HOST_NAME from env failed, is env.(\"HOST_NAME\") already set? Will use hostname instead")
	return getHostName()
}

const CommandTimeoutEnv = "DATE_AGENT_CMD_TIMEOUT"
const CommandTimeout = 15

func Exec(suffix []string) (out string, err error) {
	t := CommandTimeout
	if x := os.Getenv(CommandTimeoutEnv); x != "" {
		if t, err = strconv.Atoi(x); err != nil {
			fmt.Println(err)
		} else {
			t = CommandTimeout
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(t))
	var res []byte
	commands := []string{"-c"}
	commands = append(commands, suffix...)
	res, err = exec.CommandContext(ctx, "/bin/bash", commands...).Output()
	cancel()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return string(res), nil
}
