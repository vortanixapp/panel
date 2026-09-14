package main

import (
	"os"

	"github.com/vortanixapp/panel/internal/agent/agent"
	"github.com/vortanixapp/panel/internal/agent/selfupdate"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "agent-upgrade" {
		os.Exit(selfupdate.RunHelper(os.Args[2:]))
	}
	agent.New().Run()
}
