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

func cmdAdd(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: fool add <file>")
		return
	}
	file := args[0]
	if _, err := os.Stat(file); err != nil {
		fmt.Printf("File '%s' does not exist.\n", file)
		return
	}
	indexPath := ".fool/index"
	var staged []string
	if data, err := os.ReadFile(indexPath); err == nil {
		lines := string(data)
		for _, line := range splitLines(lines) {
			if line == file {
				fmt.Printf("File '%s' is already staged.\n", file)
				return
			}
			if line != "" {
				staged = append(staged, line)
			}
		}
	}
	staged = append(staged, file)
	f, err := os.OpenFile(indexPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error updating index:", err)
		return
	}
	defer f.Close()
	for _, s := range staged {
		fmt.Fprintln(f, s)
	}
	fmt.Printf("Added '%s' to staging area.\n", file)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
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
		cmdAdd(os.Args[2:])
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
