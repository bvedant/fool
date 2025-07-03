package main

import (
	"bufio"
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const foolVersion = "0.1.1"

func ensureRepo() {
	if _, err := os.Stat(".fool"); os.IsNotExist(err) {
		fmt.Println("Error: not a fool repository (run 'fool init' first)")
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("fool - a minimal version control system")
	fmt.Println("Usage:")
	fmt.Println("  fool <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  init         Initialize a new repository")
	fmt.Println("  add <file>   Add a file to the staging area")
	fmt.Println("  commit -m <message>  Commit staged files with a message")
	fmt.Println("  log          Show commit history")
	fmt.Println("  status       Show the status of the working directory")
	fmt.Println("  help [cmd]   Show help for a command")
	fmt.Println("  version      Show fool version")
}

func printCommandHelp(cmd string) {
	switch cmd {
	case "init":
		fmt.Println("Usage: fool init\n  Initialize a new repository.")
	case "add":
		fmt.Println("Usage: fool add <file>\n  Add a file to the staging area.")
	case "commit":
		fmt.Println("Usage: fool commit -m <message>\n  Commit staged files with a message.")
	case "log":
		fmt.Println("Usage: fool log\n  Show commit history.")
	case "status":
		fmt.Println("Usage: fool status\n  Show the status of the working directory.")
	case "version":
		fmt.Println("Usage: fool version\n  Show fool version.")
	default:
		printUsage()
	}
}

func cmdHelp(args []string) {
	if len(args) > 0 {
		printCommandHelp(args[0])
	} else {
		printUsage()
	}
}

func cmdVersion() {
	fmt.Printf("fool version %s\n", foolVersion)
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
	ensureRepo()
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
	scanner := bufio.NewScanner(strings.NewReader(s))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func cmdCommit(args []string) {
	ensureRepo()
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
		defer in.Close()
		outPath := filepath.Join(commitDir, file)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			fmt.Printf("Warning: could not create directory for '%s', skipping.\n", file)
			in.Close()
			continue
		}
		out, err := os.Create(outPath)
		if err != nil {
			fmt.Printf("Warning: could not write '%s', skipping.\n", file)
			in.Close()
			continue
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			fmt.Printf("Warning: could not copy '%s', skipping.\n", file)
			continue
		}
		committedFiles = append(committedFiles, file)
	}
	if len(committedFiles) == 0 {
		fmt.Println("No files were committed.")
		return
	}
	meta := fmt.Sprintf("commit: %s\ndate: %s\nmessage: %s\nfiles: %v\n", commitID, commitTime, *msg, committedFiles)
	if err := os.WriteFile(filepath.Join(commitDir, "meta.txt"), []byte(meta), 0644); err != nil {
		fmt.Println("Error writing commit metadata:", err)
		return
	}
	// Append to log
	logEntry := fmt.Sprintf("commit %s\nDate: %s\nMessage: %s\nFiles: %v\n\n", commitID, commitTime, *msg, committedFiles)
	f, err := os.OpenFile(".fool/log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error writing to log:", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(logEntry); err != nil {
		fmt.Println("Error writing log entry:", err)
		return
	}
	// Clear index
	if err := os.WriteFile(indexPath, []byte{}, 0644); err != nil {
		fmt.Println("Error clearing index:", err)
		return
	}
	fmt.Printf("Committed %d file(s) with id %s\n", len(committedFiles), commitID)
}

func genCommitID(ts, msg string) string {
	h := sha1.New()
	h.Write([]byte(ts + msg))
	return fmt.Sprintf("%x", h.Sum(nil))[:8]
}

func cmdLog() {
	ensureRepo()
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
	ensureRepo()
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
	lastCommitFiles, lastCommitID := getLastCommitFilesAndID()
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

	// Show modified files (in last commit, not staged, and contents differ)
	modified := []string{}
	for f := range lastCommitFiles {
		if staged[f] {
			continue // staged files already shown
		}
		wdData, err1 := os.ReadFile(f)
		commitData, err2 := os.ReadFile(filepath.Join(".fool", "objects", lastCommitID, f))
		if err1 == nil && err2 == nil && string(wdData) != string(commitData) {
			modified = append(modified, f)
		}
	}
	if len(modified) > 0 {
		fmt.Println("Modified files:")
		for _, f := range modified {
			fmt.Println("  ", f)
		}
	}
}

func getLastCommitFilesAndID() (map[string]bool, string) {
	logPath := ".fool/log"
	data, err := os.ReadFile(logPath)
	if err != nil || len(data) == 0 {
		return map[string]bool{}, ""
	}
	entries := splitLogEntries(string(data))
	if len(entries) == 0 {
		return map[string]bool{}, ""
	}
	last := entries[len(entries)-1]
	files := map[string]bool{}
	var commitID string
	for _, line := range splitLines(last) {
		if len(line) > 7 && line[:7] == "Files: " {
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
		if len(line) > 7 && line[:7] == "commit " {
			commitID = line[7:]
		}
	}
	return files, commitID
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	// Global help/version
	if cmd == "--help" || cmd == "-h" {
		printUsage()
		return
	}
	if cmd == "version" {
		cmdVersion()
		return
	}
	if cmd == "help" {
		cmdHelp(args)
		return
	}

	switch cmd {
	case "init":
		if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
			printCommandHelp("init")
			return
		}
		cmdInit()
	case "add":
		if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
			printCommandHelp("add")
			return
		}
		cmdAdd(args)
	case "commit":
		if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
			printCommandHelp("commit")
			return
		}
		cmdCommit(args)
	case "log":
		if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
			printCommandHelp("log")
			return
		}
		cmdLog()
	case "status":
		if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
			printCommandHelp("status")
			return
		}
		cmdStatus()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
	}
}
