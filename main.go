package main

import "ragie-cli/cmd"

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	cmd.Execute()
}
