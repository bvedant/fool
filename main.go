package main

import (
	"fmt"
	"os"
)

func printUsage() {
	fmt.Println("fool - a minimal version control system")
	fmt.Println("Usage:")
	fmt.Println("  fool init")
	fmt.Println("  fool add <file>")
	fmt.Println("  fool commit -m <message>")
	fmt.Println("  fool log")
	fmt.Println("  fool status")
}

func cmdInit() {
	dir := ".fool"
	if _, err := os.Stat(dir); err == nil {
		fmt.Println("Repository already initialized.")
		return
	}
	err := os.Mkdir(dir, 0755)
	if err != nil {
		fmt.Println("Error initializing repository:", err)
		os.Exit(1)
	}
	fmt.Println("Initialized empty fool repository in .fool/")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "add":
		fmt.Println("add: not implemented yet")
	case "commit":
		fmt.Println("commit: not implemented yet")
	case "log":
		fmt.Println("log: not implemented yet")
	case "status":
		fmt.Println("status: not implemented yet")
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
	}
}
