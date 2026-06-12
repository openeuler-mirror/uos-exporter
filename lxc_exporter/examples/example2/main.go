package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	lxcLsCmd := "lxc-ls"
	lxcLsRunArg := []string{"--running"}
	var paths []string

	cmd := exec.Command(lxcLsCmd, lxcLsRunArg...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error running lxc-ls: %v\n", err)
		return
	}
	//for _, line := range strings.Split(string(output), " ") {
	//	if line == "" {
	//		continue
	//	}
	//	paths = append(paths, line)
	//}
	paths = strings.Fields(string(output))
	fmt.Printf("Running containers:\n%#v\n", paths)
}
