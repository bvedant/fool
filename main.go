package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
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

func cmdCommit(args []string) {
	fs := flag.NewFlagSet("commit", flag.ExitOnError)
	msg := fs.String("m", "", "commit message")
	fs.Parse(args)
	if *msg == "" {
		fmt.Println("Usage: fool commit -m <message>")
		return
	}
	indexPath := ".fool/index"
	data, err := os.ReadFile(indexPath)
	if err != nil || len(data) == 0 {
		fmt.Println("Nothing to commit. Staging area is empty.")
		return
	}
	files := splitLines(string(data))
	commitTime := time.Now().UTC().Format(time.RFC3339)
	commitID := genCommitID(commitTime, *msg)
	commitDir := filepath.Join(".fool", "objects", commitID)
	if err := os.MkdirAll(commitDir, 0755); err != nil {
		fmt.Println("Error creating commit directory:", err)
		return
	}
	var committedFiles []string
	for _, file := range files {
		if file == "" {
			continue
		}
		in, err := os.Open(file)
		if err != nil {
			fmt.Printf("Warning: could not open '%s', skipping.\n", file)
			continue
		}
		outPath := filepath.Join(commitDir, file)
		os.MkdirAll(filepath.Dir(outPath), 0755)
		out, err := os.Create(outPath)
		if err != nil {
			fmt.Printf("Warning: could not write '%s', skipping.\n", file)
			in.Close()
			continue
		}
		io.Copy(out, in)
		in.Close()
		out.Close()
		committedFiles = append(committedFiles, file)
	}
	if len(committedFiles) == 0 {
		fmt.Println("No files were committed.")
		return
	}
	meta := fmt.Sprintf("commit: %s\ndate: %s\nmessage: %s\nfiles: %v\n", commitID, commitTime, *msg, committedFiles)
	os.WriteFile(filepath.Join(commitDir, "meta.txt"), []byte(meta), 0644)
	// Append to log
	logEntry := fmt.Sprintf("commit %s\nDate: %s\nMessage: %s\nFiles: %v\n\n", commitID, commitTime, *msg, committedFiles)
	f, _ := os.OpenFile(".fool/log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f.WriteString(logEntry)
	f.Close()
	// Clear index
	os.WriteFile(indexPath, []byte{}, 0644)
	fmt.Printf("Committed %d file(s) with id %s\n", len(committedFiles), commitID)
}

func genCommitID(ts, msg string) string {
	h := sha1.New()
	h.Write([]byte(ts + msg))
	return fmt.Sprintf("%x", h.Sum(nil))[:8]
}

func cmdLog() {
	logPath := ".fool/log"
	data, err := os.ReadFile(logPath)
	if err != nil || len(data) == 0 {
		fmt.Println("No commits yet.")
		return
	}
	entries := splitLogEntries(string(data))
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i] != "" {
			fmt.Println(entries[i])
		}
	}
}

func splitLogEntries(s string) []string {
	var entries []string
	start := 0
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '\n' && s[i+1] == '\n' {
			entries = append(entries, s[start:i])
			start = i + 2
		}
	}
	if start < len(s) {
		entries = append(entries, s[start:])
	}
	return entries
}

func cmdStatus() {
	// List staged files
	indexPath := ".fool/index"
	staged := map[string]bool{}
	if data, err := os.ReadFile(indexPath); err == nil && len(data) > 0 {
		for _, f := range splitLines(string(data)) {
			if f != "" {
				staged[f] = true
			}
		}
	}
	if len(staged) > 0 {
		fmt.Println("Staged files:")
		for f := range staged {
			fmt.Println("  ", f)
		}
	} else {
		fmt.Println("No files staged for commit.")
	}

	// List untracked files (in project root, not staged, not in last commit)
	files, _ := os.ReadDir(".")
	lastCommitFiles := getLastCommitFiles()
	untracked := []string{}
	for _, file := range files {
		name := file.Name()
		if file.IsDir() || name == ".fool" || name == ".git" {
			continue
		}
		if !staged[name] && !lastCommitFiles[name] {
			untracked = append(untracked, name)
		}
	}
	if len(untracked) > 0 {
		fmt.Println("Untracked files:")
		for _, f := range untracked {
			fmt.Println("  ", f)
		}
	}
}

func getLastCommitFiles() map[string]bool {
	logPath := ".fool/log"
	data, err := os.ReadFile(logPath)
	if err != nil || len(data) == 0 {
		return map[string]bool{}
	}
	entries := splitLogEntries(string(data))
	if len(entries) == 0 {
		return map[string]bool{}
	}
	last := entries[len(entries)-1]
	files := map[string]bool{}
	for _, line := range splitLines(last) {
		if len(line) > 7 && line[:7] == "Files: " {
			// crude parse: Files: [a b c]
			var fname string
			for _, v := range line[7:] {
				if v != '[' && v != ']' && v != ' ' && v != ',' {
					fname += string(v)
				} else if fname != "" {
					files[fname] = true
					fname = ""
				}
			}
			if fname != "" {
				files[fname] = true
			}
		}
	}
	return files
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
		cmdCommit(os.Args[2:])
	case "log":
		cmdLog()
	case "status":
		cmdStatus()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
	}
}
