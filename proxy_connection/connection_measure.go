package proxy_connection

import (
	"http-proxy/config"
	"log"
	"os/exec"
	"strings"
)

func MeasureEstablishedConnections() (int, error) {
	cmd := "netstat -an"

	// Split the command into base command and arguments
	parts := strings.Fields(cmd)
	baseCmd := parts[0]
	args := parts[1:]

	// Execute the command
	output, err := exec.Command(baseCmd, args...).Output()
	//output, err := exec.Command(cmd).Output()
	if err != nil {
		log.Fatal("Error:", err)
		return 0, err
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	liveConnections := 0

	for _, line := range lines {
		if !strings.ContainsAny(line, "ESTABLISHED") {
			continue
		}
		for _, s := range config.ParentProxy {
			if strings.ContainsAny(line, s) {
				liveConnections++
				break
			}
		}
	}

	return liveConnections, nil
}
