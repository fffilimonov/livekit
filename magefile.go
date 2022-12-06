//go:build mage
// +build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"

	"github.com/livekit/livekit-server/version"
	"github.com/livekit/mageutil"
)

const (
	goChecksumFile = ".checksumgo"
	imageName      = "livekit/livekit-server"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
var Default = Build
var checksummer = mageutil.NewChecksummer(".", goChecksumFile, ".go", ".mod")

func init() {
	checksummer.IgnoredPaths = []string{
		"pkg/service/wire_gen.go",
		"pkg/rtc/types/typesfakes",
	}
}

// explicitly reinstall all deps
func Deps() error {
	return installTools(true)
}

// builds LiveKit server
func Build() error {
	mg.Deps(generateWire)
	if !checksummer.IsChanged() {
		fmt.Println("up to date")
		return nil
	}

	fmt.Println("building...")
	if err := os.MkdirAll("bin", 0755); err != nil {
		return err
	}
	if err := mageutil.RunDir(context.Background(), "cmd/server", "go build -o ../../bin/livekit-server"); err != nil {
		return err
	}

	checksummer.WriteChecksum()
	return nil
}

// run unit tests, skipping integration
func Test() error {
	mg.Deps(generateWire, setULimit)
	return mageutil.Run(context.Background(), "go test -short ./... -count=1")
}

// run all tests including integration
func TestAll() error {
	mg.Deps(generateWire, setULimit)
	return mageutil.Run(context.Background(), "go test ./... -count=1 -timeout=4m -v")
}

// cleans up builds
func Clean() {
	fmt.Println("cleaning...")
	os.RemoveAll("bin")
	os.Remove(goChecksumFile)
}

// regenerate code
func Generate() error {
	mg.Deps(installDeps, generateWire)

	fmt.Println("generating...")
	return mageutil.Run(context.Background(), "go generate ./...")
}

// code generation for wiring
func generateWire() error {
	mg.Deps(installDeps)
	if !checksummer.IsChanged() {
		return nil
	}

	fmt.Println("wiring...")

	wire, err := mageutil.GetToolPath("wire")
	if err != nil {
		return err
	}
	cmd := exec.Command(wire)
	cmd.Dir = "pkg/service"
	mageutil.ConnectStd(cmd)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// implicitly install deps
func installDeps() error {
	return installTools(false)
}

func installTools(force bool) error {
	tools := map[string]string{
		"github.com/google/wire/cmd/wire": "latest",
	}
	for t, v := range tools {
		if err := mageutil.InstallTool(t, v, force); err != nil {
			return err
		}
	}
	return nil
}
