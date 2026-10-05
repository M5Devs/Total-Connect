package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/obscure"

	"github.com/M5Devs/Total-Connect/internal/models"
)

func TestRcloneEngineCreateAndDeleteRemote(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "rclone.conf")

	engine, err := NewRcloneEngine(cfgPath)
	if err != nil {
		t.Fatalf("failed to create RcloneEngine: %v", err)
	}

	ctx := context.Background()

	// Test CreateRemote
	params := map[string]string{
		"host": "ftp.example.com",
		"user": "testuser",
		"pass": "secretpassword",
		"port": "21",
	}

	err = engine.CreateRemote(ctx, "TestFTP", "ftp", params)
	if err != nil {
		t.Fatalf("CreateRemote failed: %v", err)
	}

	remotes, err := engine.ListRemotes(ctx)
	if err != nil {
		t.Fatalf("ListRemotes failed: %v", err)
	}

	found := false
	for _, r := range remotes {
		if r == "TestFTP" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected TestFTP in remotes list %v", remotes)
	}

	// Verify obfuscated password stored in config
	storedPass := config.FileGet("TestFTP", "pass")
	if storedPass == "secretpassword" {
		t.Errorf("expected password to be obscured in config, got plain text")
	}

	deobsPass, err := obscure.Reveal(storedPass)
	if err != nil || deobsPass != "secretpassword" {
		t.Errorf("expected revealed password to match 'secretpassword', got %q, err: %v", deobsPass, err)
	}

	// Verify file was written to disk
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Errorf("expected config file to exist at %s", cfgPath)
	}

	// Test DeleteRemote
	err = engine.DeleteRemote(ctx, "TestFTP")
	if err != nil {
		t.Fatalf("DeleteRemote failed: %v", err)
	}

	remotes, err = engine.ListRemotes(ctx)
	if err != nil {
		t.Fatalf("ListRemotes failed: %v", err)
	}

	for _, r := range remotes {
		if r == "TestFTP" {
			t.Errorf("expected TestFTP to be deleted, still found in %v", remotes)
		}
	}
}

func TestRcloneEngineLocalOps(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "rclone.conf")

	engine, err := NewRcloneEngine(cfgPath)
	if err != nil {
		t.Fatalf("failed to create RcloneEngine: %v", err)
	}

	ctx := context.Background()

	// Test ListRemotes (may be empty)
	remotes, err := engine.ListRemotes(ctx)
	if err != nil {
		t.Fatalf("ListRemotes failed: %v", err)
	}
	_ = remotes

	// Create test directory
	testSubDir := filepath.Join(tmpDir, "testdir")
	err = engine.Mkdir(ctx, testSubDir)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(testSubDir, "sample.txt")
	err = os.WriteFile(testFile, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// List entries
	items, err := engine.ListEntries(ctx, testSubDir)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(items) != 1 || items[0].Name != "sample.txt" {
		t.Errorf("expected sample.txt in entries, got %+v", items)
	}

	// Copy file
	dstFile := filepath.Join(testSubDir, "sample_copy.txt")
	err = engine.Copy(ctx, testFile, dstFile)
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Errorf("expected copied file to exist at %s", dstFile)
	}

	// Delete file
	err = engine.Delete(ctx, dstFile)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(dstFile); !os.IsNotExist(err) {
		t.Errorf("expected deleted file to no longer exist at %s", dstFile)
	}
}

func TestRcloneEngineMoveFileAndDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "rclone.conf")

	engine, err := NewRcloneEngine(cfgPath)
	if err != nil {
		t.Fatalf("failed to create RcloneEngine: %v", err)
	}

	var reportedProgress []models.Progress
	progressFn := func(p models.Progress) {
		reportedProgress = append(reportedProgress, p)
	}
	ctx := WithProgressHandler(context.Background(), progressFn)

	// Create src directory and file
	srcDir := filepath.Join(tmpDir, "srcdir")
	err = engine.Mkdir(ctx, srcDir)
	if err != nil {
		t.Fatalf("Mkdir srcDir failed: %v", err)
	}

	srcFile := filepath.Join(srcDir, "file_to_move.txt")
	err = os.WriteFile(srcFile, []byte("moving content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	dstFile := filepath.Join(srcDir, "file_moved.txt")
	err = engine.Move(ctx, srcFile, dstFile)
	if err != nil {
		t.Fatalf("Move file failed: %v", err)
	}

	// Verify original file is gone and destination exists
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Errorf("expected srcFile to be moved and not exist at %s", srcFile)
	}
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Errorf("expected dstFile to exist at %s", dstFile)
	}

	if len(reportedProgress) == 0 {
		t.Errorf("expected progress callbacks to be called")
	}

	// Test Directory Move
	dstDir := filepath.Join(tmpDir, "dstdir")
	err = engine.Move(ctx, srcDir, dstDir)
	if err != nil {
		t.Fatalf("Move directory failed: %v", err)
	}

	movedFilePath := filepath.Join(dstDir, "file_moved.txt")
	if _, err := os.Stat(movedFilePath); os.IsNotExist(err) {
		t.Errorf("expected moved directory content to exist at %s", movedFilePath)
	}
}

func TestRcloneEngineSFTPKeyUseAgentDefault(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "rclone.conf")

	engine, err := NewRcloneEngine(cfgPath)
	if err != nil {
		t.Fatalf("failed to create RcloneEngine: %v", err)
	}

	ctx := context.Background()

	// 1. Create SFTP remote without key_use_agent
	params := map[string]string{
		"host": "sftp.example.com",
		"user": "sftpuser",
		"pass": "sftppass",
	}

	err = engine.CreateRemote(ctx, "TestSFTP", "sftp", params)
	if err != nil {
		t.Fatalf("CreateRemote SFTP failed: %v", err)
	}

	val := config.FileGet("TestSFTP", "key_use_agent")
	if val != "false" {
		t.Errorf("expected key_use_agent to default to 'false', got %q", val)
	}

	remoteType, cfgParams, err := engine.GetRemoteConfig(ctx, "TestSFTP")
	if err != nil {
		t.Fatalf("GetRemoteConfig failed: %v", err)
	}
	if remoteType != "sftp" {
		t.Errorf("expected remoteType 'sftp', got %q", remoteType)
	}
	if cfgParams["key_use_agent"] != "false" {
		t.Errorf("expected key_use_agent='false' in GetRemoteConfig, got %q", cfgParams["key_use_agent"])
	}

	// 2. Create SFTP remote WITH key_use_agent = true
	paramsWithAgent := map[string]string{
		"host":          "sftp2.example.com",
		"user":          "sftpuser",
		"key_use_agent": "true",
	}
	err = engine.CreateRemote(ctx, "TestSFTPWithAgent", "sftp", paramsWithAgent)
	if err != nil {
		t.Fatalf("CreateRemote SFTP with agent failed: %v", err)
	}

	valWithAgent := config.FileGet("TestSFTPWithAgent", "key_use_agent")
	if valWithAgent != "true" {
		t.Errorf("expected key_use_agent to be 'true', got %q", valWithAgent)
	}
}

func TestRcloneEngineBareFilenameCopy(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "rclone.conf")

	engine, err := NewRcloneEngine(cfgPath)
	if err != nil {
		t.Fatalf("failed to create RcloneEngine: %v", err)
	}

	ctx := context.Background()

	// Switch working directory to tmpDir so bare "README.md" is in current working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tmpDir: %v", err)
	}

	// Create bare file "README.md" in root/working dir
	bareFile := "README.md"
	if err := os.WriteFile(bareFile, []byte("root file content"), 0644); err != nil {
		t.Fatalf("failed to create README.md: %v", err)
	}

	// Create target directory
	targetDir := "target_folder"
	if err := engine.Mkdir(ctx, targetDir); err != nil {
		t.Fatalf("Mkdir targetDir failed: %v", err)
	}

	// Copy bare filename "README.md" to "target_folder"
	dstPath := filepath.Join(targetDir, "README.md")
	if err := engine.Copy(ctx, bareFile, dstPath); err != nil {
		t.Fatalf("Copy bare filename %q failed: %v", bareFile, err)
	}

	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Errorf("expected copied file to exist at %s", dstPath)
	}

	// Test copying bare filename directly to target directory path
	dstDirOnly := targetDir
	if err := engine.Copy(ctx, bareFile, dstDirOnly); err != nil {
		t.Fatalf("Copy bare filename to directory %q failed: %v", dstDirOnly, err)
	}

	// Test moving bare file
	moveSrc := "MOVE_ME.txt"
	if err := os.WriteFile(moveSrc, []byte("move content"), 0644); err != nil {
		t.Fatalf("failed to create MOVE_ME.txt: %v", err)
	}
	moveDst := filepath.Join(targetDir, "MOVED.txt")
	if err := engine.Move(ctx, moveSrc, moveDst); err != nil {
		t.Fatalf("Move bare filename failed: %v", err)
	}

	if _, err := os.Stat(moveSrc); !os.IsNotExist(err) {
		t.Errorf("expected original moveSrc file to no longer exist")
	}
	if _, err := os.Stat(moveDst); os.IsNotExist(err) {
		t.Errorf("expected moved file to exist at %s", moveDst)
	}
}
