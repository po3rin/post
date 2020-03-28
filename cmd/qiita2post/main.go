package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

func newWalker(postpath string) func(path string, info os.FileInfo, err error) error {
	return func(path string, info os.FileInfo, err error) error {
		if !isPostMdfile(path) {
			return nil
		}

		sl := strings.Split(filepath.Base(path), "-")
		yearStr := sl[0] + "-" + sl[1] + "-" + sl[2]
		t, err := time.Parse("2006-01-02", yearStr)
		if err != nil {
			return errors.Wrap(err, "qiita2post: parse time")
		}

		year := strconv.Itoa(t.Year())
		unix := strconv.FormatInt(t.Unix(), 10)
		dir := filepath.Join(postpath, year, unix)
		err = createDir(dir)
		if err != nil {
			return errors.Wrap(err, "qiita2post: create directory")
		}

		dist := filepath.Join(dir, "post.md")
		err = os.Rename(path, dist)
		if err != nil {
			return errors.Wrap(err, "qiita2post: mv post file")
		}

		return nil
	}
}

func main() {
	postpath := flag.String("root", "./posts", "posts root directory flag")
	root := flag.String("dir", ".", "posts dir created by qiitaexporter")
	flag.Parse()

	err := filepath.Walk(*root, newWalker(*postpath))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
