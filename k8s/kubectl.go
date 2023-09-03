package k8s

import (
	"os/exec"
)

func RunKubectl(args []string) (output []byte, err error) {
	command := "kubectl"
	cmd := exec.Command(command, args...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return output, err
	}
	return output, nil
}
