package ethutil

import (
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"
)

func ExpandHomePath(p string) (path string) {
	path = p

	// Check in case of paths like "/something/~/something/"
	if path[:2] == "~/" {
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

func ReadAllFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func WriteFile(filePath string, content []byte) error {
	fh, err := os.OpenFile(filePath, os.O_TRUNC|os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = fh.Write(content)
	if err != nil {
		return err
	}

	return nil
}

func AbsolutePath(Datadir string, filename string) string {
	if path.IsAbs(filename) {
		return filename
	}
	return path.Join(Datadir, filename)
}
