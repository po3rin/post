package cmd

import (
	"os"
	"path/filepath"
)

func createDir(dirpath string) error {
	dirpath = filepath.Clean(dirpath)
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		return os.MkdirAll(dirpath, os.ModePerm)
	}
	return nil
}

func isPostMdfile(path string) bool {
	e := filepath.Ext(path)
	if e != ".md" {
		return false
	}
	b := filepath.Base(path)
	if b == "README.md" {
		return false
	}
	return true
}
