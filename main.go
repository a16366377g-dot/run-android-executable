package main

import (
	"archive/tar"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"bytes"
	"syscall"
	"github.com/klauspost/compress/zstd"
)

//go:embed res.tar.zst
var embeddedRes embed.FS

//go:embed available_commands.txt
var availableCommands []byte

var (
	executableDir  string
	executableName string
	androidLinker  string
)

func main() {
	cmd := cmdBuilder(os.Args)
	if cmd == nil {
		cleanup(1)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	cleanup(getExitCode(err))
}

func init() {
	getExecutableNameAndDir()
	checkAndroidLinker()
	getTmpDir()
	os.Setenv("LD_LIBRARY_PATH", filepath.Join(os.TempDir(), "bin", "lib"))

	if !extractBin() {
		fmt.Fprintln(os.Stderr, "Error during extraction.")
		cleanup(1)
	}
}

func getExitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				return 128 + int(status.Signal())
			}
			return status.ExitStatus()
		}
	}
	return 0
}

func cleanup(exitCode int) {
	os.RemoveAll(os.TempDir())
	os.Exit(exitCode)
}

func checkAndroidLinker() {
	cmd := exec.Command("getprop", "ro.build.version.sdk")
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting Android API level:", err)
		os.Exit(1)
	}

	apiLevel, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error converting API level:", err)
		os.Exit(1)
	}

	if apiLevel > 28 {
		switch 32 << (^uint(0) >> 63) {
		case 64:
			androidLinker = "/system/bin/linker64"
		case 32:
			androidLinker = "/system/bin/linker"
		}
	}
}

func getExecutableNameAndDir() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	absPath, _ := filepath.EvalSymlinks(exePath)

	linkerPaths := []string{
		"/apex/com.android.runtime/bin/linker64",
		"/system/bin/linker64",
		"/apex/com.android.runtime/bin/linker",
		"/system/bin/linker",
	}

	for _, path := range linkerPaths {
		if absPath == path {
			executableName = filepath.Base(os.Args[1])
			executableDir = filepath.Dir(os.Args[1])
			return
		}
	}
	executableName = filepath.Base(exePath)
	executableDir = filepath.Dir(exePath)
}

func getTmpDir() {
	tmpDir := filepath.Join(os.TempDir(), ".tmp_run_dir_"+strconv.Itoa(os.Getpid()))
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		tmpDir = filepath.Join(executableDir, ".tmp_run_dir_"+strconv.Itoa(os.Getpid()))
		if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
			fmt.Fprintln(os.Stderr, "Temporary directory not available. Specify it using TMPDIR environment variable")
			os.Exit(1)
		}
	}
	os.Setenv("TMPDIR", tmpDir)
}

func extractBin() bool {
	extractionPath := os.TempDir()

	data, err := embeddedRes.ReadFile("res.tar.zst")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read embedded resource:", err)
		return false
	}

	if err := extractTarZst(data, extractionPath); err != nil {
		fmt.Fprintln(os.Stderr, "Extraction failed:", err)
		return false
	}

	return true
}

func extractTarZst(data []byte, destDir string) error {
	zstReader, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer zstReader.Close()

	tarReader := tar.NewReader(zstReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(targetPath, os.ModePerm)
		case tar.TypeReg:
			err = func() error {
				outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
				defer outFile.Close()
				_, err = io.Copy(outFile, tarReader)
				return err
			}()
		}

		if err != nil {
			return err
		}
	}
}

func cmdBuilder(args []string) *exec.Cmd {
	absPath, _ := filepath.EvalSymlinks(args[0])

	args = args[1:]

	linkerPaths := []string{
		"/apex/com.android.runtime/bin/linker64",
		"/system/bin/linker64",
		"/apex/com.android.runtime/bin/linker",
		"/system/bin/linker",
	}

	for _, path := range linkerPaths {
		if absPath == path {
			args = args[1:]
		}
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Available commands: "+string(availableCommands))
		return nil
	}

	binDir := filepath.Join(os.TempDir(), "bin")

	if androidLinker == "" {
		if len(args) > 1 {
			return exec.Command(filepath.Join(binDir, args[0]), args[1:]...)
		} else {
			return exec.Command(filepath.Join(binDir, args[0]))
		}
	} else {
		if len(args) > 1 {
			return exec.Command(androidLinker, append([]string{filepath.Join(binDir, args[0])}, args[1:]...)...)
		} else {
			return exec.Command(androidLinker, []string{filepath.Join(binDir, args[0])}...)
		}
	}
}
