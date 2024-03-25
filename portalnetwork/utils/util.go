package utils

import (
	"errors"
	"os"
)

func EnsureDir(dir string) error {
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				return err
			}
		}
		return err
	}

	if !stat.IsDir() {
		return errors.New("node dir should be a dir")
	}
	return nil
}
