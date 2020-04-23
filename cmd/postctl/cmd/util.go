package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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

func mdTitle(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", errors.Wrap(err, "mdcon: file open")
	}
	defer f.Close()

	var title string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		title = scanner.Text()
		if !strings.HasPrefix(title, "# ") {
			return "", fmt.Errorf("invalid title format in %s", path)
		}
		title = strings.ReplaceAll(title, "# ", "")
		break // first line only // TODO: parse markdown.
	}

	if err = scanner.Err(); err != nil {
		return "", errors.Wrap(err, "mdcon: get first line")
	}
	return title, nil
}
