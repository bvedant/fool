package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to run fool CLI in a temp dir
type FoolTestEnv struct {
	tmpDir string
	bin    string
}

func setupFoolTestEnv(t *testing.T) *FoolTestEnv {
	tmpDir, err := os.MkdirTemp("", "fooltest-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	bin := filepath.Join(tmpDir, "fool")
	// Copy the built binary to the temp dir
	origBin := "./fool"
	data, err := os.ReadFile(origBin)
	if err != nil {
		t.Fatalf("failed to read fool binary: %v", err)
	}
	if err := os.WriteFile(bin, data, 0755); err != nil {
		t.Fatalf("failed to write fool binary: %v", err)
	}
	return &FoolTestEnv{tmpDir, bin}
}

func (env *FoolTestEnv) run(args ...string) (string, error) {
	cmd := exec.Command(env.bin, args...)
	cmd.Dir = env.tmpDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestInit(t *testing.T) {
	env := setupFoolTestEnv(t)
	defer os.RemoveAll(env.tmpDir)
	out, err := env.run("init")
	if err != nil {
		t.Fatalf("init failed: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "Initialized empty fool repository") {
		t.Errorf("unexpected output: %s", out)
	}
	// Check .fool dir exists
	if _, err := os.Stat(filepath.Join(env.tmpDir, ".fool")); err != nil {
		t.Errorf(".fool directory not created")
	}
}

func TestAdd(t *testing.T) {
	env := setupFoolTestEnv(t)
	defer os.RemoveAll(env.tmpDir)
	_, err := env.run("init")
	if err != nil {
		t.Fatalf("init failed")
	}
	// Create a test file
	testFile := filepath.Join(env.tmpDir, "foo.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	out, err := env.run("add", "foo.txt")
	if err != nil {
		t.Fatalf("add failed: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "Added 'foo.txt' to staging area") {
		t.Errorf("unexpected output: %s", out)
	}
	// Check .fool/index contains foo.txt
	indexData, err := os.ReadFile(filepath.Join(env.tmpDir, ".fool", "index"))
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if !strings.Contains(string(indexData), "foo.txt") {
		t.Errorf("index does not contain staged file")
	}
}

func TestAddMultipleFiles(t *testing.T) {
	env := setupFoolTestEnv(t)
	defer os.RemoveAll(env.tmpDir)
	_, err := env.run("init")
	if err != nil {
		t.Fatalf("init failed")
	}
	// Create test files
	file1 := filepath.Join(env.tmpDir, "foo1.txt")
	file2 := filepath.Join(env.tmpDir, "foo2.txt")
	file3 := filepath.Join(env.tmpDir, "foo3.txt")
	os.WriteFile(file1, []byte("one"), 0644)
	os.WriteFile(file2, []byte("two"), 0644)
	os.WriteFile(file3, []byte("three"), 0644)
	// Add multiple files at once
	out, err := env.run("add", "foo1.txt", "foo2.txt", "foo3.txt")
	if err != nil {
		t.Fatalf("add failed: %v, output: %s", err, out)
	}
	// Debug: print output and directory contents
	t.Logf("add output: %s", out)
	files, _ := os.ReadDir(env.tmpDir)
	var fnames []string
	for _, f := range files {
		fnames = append(fnames, f.Name())
	}
	t.Logf("temp dir files: %v", fnames)
	indexPath := filepath.Join(env.tmpDir, ".fool", "index")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	t.Logf("index contents:\n%s", string(indexData))
	if !strings.Contains(out, "Added 'foo1.txt") || !strings.Contains(out, "Added 'foo2.txt") || !strings.Contains(out, "Added 'foo3.txt") {
		t.Errorf("not all files reported as added: %s", out)
	}
	for _, fname := range []string{"foo1.txt", "foo2.txt", "foo3.txt"} {
		if !strings.Contains(string(indexData), fname) {
			t.Errorf("index does not contain staged file %s", fname)
		}
	}
	// Add one file again, should not duplicate
	out, err = env.run("add", "foo1.txt")
	if err != nil {
		t.Fatalf("add failed: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "already staged") {
		t.Errorf("should report already staged: %s", out)
	}
	indexData2, err := os.ReadFile(filepath.Join(env.tmpDir, ".fool", "index"))
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if strings.Count(string(indexData2), "foo1.txt") > 1 {
		t.Errorf("file staged more than once")
	}
}

func TestCommitAndLog(t *testing.T) {
	env := setupFoolTestEnv(t)
	defer os.RemoveAll(env.tmpDir)
	_, err := env.run("init")
	if err != nil {
		t.Fatalf("init failed")
	}
	testFile := filepath.Join(env.tmpDir, "bar.txt")
	if err := os.WriteFile(testFile, []byte("world"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	_, err = env.run("add", "bar.txt")
	if err != nil {
		t.Fatalf("add failed")
	}
	out, err := env.run("commit", "-m", "my commit")
	if err != nil {
		t.Fatalf("commit failed: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "Committed 1 file(s)") {
		t.Errorf("unexpected commit output: %s", out)
	}
	// Check index is cleared
	indexData, err := os.ReadFile(filepath.Join(env.tmpDir, ".fool", "index"))
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	if len(strings.TrimSpace(string(indexData))) != 0 {
		t.Errorf("index not cleared after commit")
	}
	// Check log contains commit message
	logData, err := os.ReadFile(filepath.Join(env.tmpDir, ".fool", "log"))
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	if !strings.Contains(string(logData), "my commit") {
		t.Errorf("log does not contain commit message")
	}
}

func TestStatus(t *testing.T) {
	env := setupFoolTestEnv(t)
	defer os.RemoveAll(env.tmpDir)
	_, err := env.run("init")
	if err != nil {
		t.Fatalf("init failed")
	}
	// Create and stage a file
	file1 := filepath.Join(env.tmpDir, "a.txt")
	os.WriteFile(file1, []byte("a"), 0644)
	_, err = env.run("add", "a.txt")
	if err != nil {
		t.Fatalf("add failed")
	}
	// Create an untracked file
	file2 := filepath.Join(env.tmpDir, "b.txt")
	os.WriteFile(file2, []byte("b"), 0644)
	out, err := env.run("status")
	if err != nil {
		t.Fatalf("status failed: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "a.txt") {
		t.Errorf("staged file not shown in status: %s", out)
	}
	if !strings.Contains(out, "b.txt") {
		t.Errorf("untracked file not shown in status: %s", out)
	}
}

func TestVersionAndHelp(t *testing.T) {
	env := setupFoolTestEnv(t)
	defer os.RemoveAll(env.tmpDir)
	out, err := env.run("version")
	if err != nil {
		t.Fatalf("version failed: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "fool version") {
		t.Errorf("version output unexpected: %s", out)
	}
	out, err = env.run("help")
	if err != nil {
		t.Fatalf("help failed: %v, output: %s", err, out)
	}
	if !strings.Contains(out, "fool - a minimal version control system") {
		t.Errorf("help output unexpected: %s", out)
	}
}
