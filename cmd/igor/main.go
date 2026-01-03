package main

import "fmt"

var (
	Version   = "1.1.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	fmt.Printf("Igor - NVIDIA TUI Installer v%s\n", Version)
	fmt.Printf("Build: %s (%s)\n", BuildTime, GitCommit)
}
