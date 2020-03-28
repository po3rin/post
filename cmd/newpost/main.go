package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func createDir(dirpath string) error {
	dirpath = filepath.Clean(dirpath)
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		return os.MkdirAll(dirpath, os.ModePerm)
	}
	return nil
}

func main() {
	postpath := flag.String("root", "./posts", "posts root directory flag")
	title := flag.String("title", "new", "new post title")
	date := flag.String("date", "", "new post date. format: 2006-01-02") // for export another blog service.
	flag.Parse()

	var t time.Time
	if *date == "" {
		t = time.Now()
	} else {
		layout := "2006-01-02"
		var err error
		t, err = time.Parse(layout, *date)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	year := strconv.Itoa(t.Year())
	unix := strconv.FormatInt(t.Unix(), 10)
	dirpath := filepath.Join(*postpath, year, unix)

	err := createDir(dirpath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mdname := "post.md"
	p := filepath.Join(dirpath, mdname)

	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	_, err = f.Write([]byte(fmt.Sprintf("# %s", *title)))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
