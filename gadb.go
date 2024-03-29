package gadb

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const customAdbPathEnv = "ADB_PATH"

var adbPath string

func getAdbPath() (string, error) {
	// if adb is installed in a custom path
	if customPath := os.Getenv(customAdbPathEnv); customPath != "" {
		d, err := os.Stat(customPath)
		if err != nil {
			return "", fmt.Errorf("%s error: %v", customAdbPathEnv, err)
		}

		if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
			return customPath, nil
		}

		return "", fmt.Errorf("adb path set by %s, byt the file is not executable", customAdbPathEnv)
	}

	lp, err := exec.LookPath("adb")
	if err != nil {
		return "", fmt.Errorf("cannot find 'adb' (under PATH neither on %s), is it installed?", customAdbPathEnv)
	}

	return lp, nil
}

// Not the most elegant way.. whatever
func init() {
	p, err := getAdbPath()
	if err != nil {
		panic(err)
	}

	adbPath = p
}

func getCmd(args []string, defaultIo bool) *exec.Cmd {
	cmd := &exec.Cmd{
		Path: adbPath,
		Args: args,
	}

	if defaultIo {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd
}

func trimSpace(s []byte) string {
	return strings.TrimSpace(string(s))
}

func Command(name string, args ...string) *exec.Cmd {
	return getCmd(append([]string{"adb", "shell", name}, args...), true)
}

func UserCommand(user string, name string, args ...string) *exec.Cmd {
	return getCmd(append([]string{"adb", "shell", "su", user, "-c", name}, args...), true)
}

func Shell(user string) error {
	return getCmd([]string{"adb", "shell", "-t", "su", user}, true).Run()
}

func getOwnership(path string) (string, error) {
	var out bytes.Buffer
	cmd := UserCommand("root", "stat", "-c", "\"%U:%G\"", path)
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}

const gadbTmpDeviceDir = "/data/local/tmp/.gadb-tmp"

func resetTmpDeviceDir() error {
	err := UserCommand("root", "rm", "-rf", gadbTmpDeviceDir).Run()
	if err != nil {
		return err
	}

	return Command("mkdir", "-p", gadbTmpDeviceDir).Run()
}

func fileExists(devicePath string) bool {
	// A bit vigorous..
	return UserCommand("root", "test", "-f", devicePath).Run() == nil
}

func Push(local, remote string) error {
	err := resetTmpDeviceDir()
	if err != nil {
		return err
	}

	cmd := &exec.Cmd{
		Path:   adbPath,
		Args:   append([]string{"adb", "push", local, gadbTmpDeviceDir}),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	err = cmd.Run()
	if err != nil {
		return err
	}

	var bname bytes.Buffer
	cmd = UserCommand("root", "basename", local)
	cmd.Stdout = &bname
	err = cmd.Run()
	if err != nil {
		return err
	}

	var ownershipToCheck string
	if fileExists(remote) {
		ownershipToCheck = remote
	} else {
		ownershipToCheck = filepath.Dir(remote)
	}

	ownerAndGroup, err := getOwnership(ownershipToCheck)
	if err != nil {
		return err
	}

	deviceTmpFilePath := path.Join(gadbTmpDeviceDir, strings.TrimSpace(bname.String()))
	err = UserCommand("root", "cp", "-R", deviceTmpFilePath, remote).Run()
	if err != nil {
		return err
	}

	return UserCommand("root", "chown", "-R", ownerAndGroup, remote).Run()
}

func Pull(remote, local string) error {
	err := resetTmpDeviceDir()
	if err != nil {
		return err
	}

	err = UserCommand("root", "cp", "-R", remote, gadbTmpDeviceDir).Run()
	if err != nil {
		return err
	}

	err = UserCommand("root", "chown", "-R", "shell:shell", gadbTmpDeviceDir).Run()
	if err != nil {
		return err
	}

	var bname bytes.Buffer
	cmd := UserCommand("root", "basename", remote)
	cmd.Stdout = &bname
	err = cmd.Run()
	if err != nil {
		return err
	}

	toPullPath := path.Join(gadbTmpDeviceDir, strings.TrimSpace(bname.String()))

	cmd = &exec.Cmd{
		Path:   adbPath,
		Args:   append([]string{"adb", "pull", toPullPath, local}),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd.Run()
}

func PackagePath(packageName string) (string, error) {
	var stderr, stdout bytes.Buffer
	cmd := UserCommand("root", "pm", "path", packageName)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	if stderrStr := strings.TrimSpace(stderr.String()); stderrStr != "" {
		return "", fmt.Errorf("%s package not found: %s", packageName, stderrStr)
	}

	stdoutStr := strings.TrimSpace(stdout.String())
	return strings.TrimPrefix(stdoutStr, "package:"), nil
}

func DeviceSerial() (string, error) {
	out, err := getCmd([]string{"adb", "get-serialno"}, false).Output()
	if err != nil {
		return "", err
	}

	return trimSpace(out), nil
}

func DeviceModel() (string, error) {
	out, err := getCmd([]string{"adb", "shell", "getprop", "ro.product.model"}, false).Output()
	if err != nil {
		return "", err
	}

	return trimSpace(out), nil
}
