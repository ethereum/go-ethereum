package common

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kardianos/osext"
)

// MakeName creates a node name that follows the ethereum convention
// for such names. It adds the operation system name and Go runtime version
// the name.
func MakeName(name, version string) string {
	return fmt.Sprintf("%s/v%s/%s/%s", name, version, runtime.GOOS, runtime.Version())
}

func ExpandHomePath(p string) (path string) {
	path = p
	sep := fmt.Sprintf("%s", os.PathSeparator)

	// Check in case of paths like "/something/~/something/"
	if len(p) > 1 && p[:1+len(sep)] == "~"+sep {
		usr, _ := user.Current()
		dir := usr.HomeDir

		path = strings.Replace(p, "~", dir, 1)
	}

	return
}

func FileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return true
}

func AbsolutePath(Datadir string, filename string) string {
	if path.IsAbs(filename) {
		return filename
	}
	return path.Join(Datadir, filename)
}

func DefaultAssetPath() string {
	var assetPath string
	pwd, _ := os.Getwd()
	srcdir := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist")

	// If the current working directory is the go-ethereum dir
	// assume a debug build and use the source directory as
	// asset directory.
	if pwd == srcdir {
		assetPath = path.Join(pwd, "assets")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// Get Binary Directory
			exedir, _ := osext.ExecutableFolder()
			assetPath = filepath.Join(exedir, "..", "Resources")
		case "linux":
			assetPath = path.Join("usr", "share", "mist")
		case "windows":
			assetPath = path.Join(".", "assets")
		default:
			assetPath = "."
		}
	}

	// Check if the assetPath exists. If not, try the source directory
	// This happens when binary is run from outside cmd/mist directory
	if _, err := os.Stat(assetPath); os.IsNotExist(err) {
		assetPath = path.Join(srcdir, "assets")
	}

	return assetPath
}

func DefaultDataDir() string {
	usr, _ := user.Current()
	if runtime.GOOS == "darwin" {
		return path.Join(usr.HomeDir, "Library", "Ethereum")
	} else if runtime.GOOS == "windows" {
		return path.Join(usr.HomeDir, "AppData", "Roaming", "Ethereum")
	} else {
		return path.Join(usr.HomeDir, ".ethereum")
	}
}

func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func WindonizePath(path string) string {
	if string(path[0]) == "/" && IsWindows() {
		path = path[1:]
	}
	return path
}
