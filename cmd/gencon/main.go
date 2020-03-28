package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type Post struct {
	title string
	url   string
	year  string
}

type Posts []Post

func (p Posts) Len() int {
	return len(p)
}

func (p Posts) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Posts) Less(i, j int) bool {
	return p[i].url > p[j].url
}

type Contents struct {
	prefix string
	Posts  Posts
}

func NewContents(prefix string) (*Contents, error) {
	return &Contents{
		prefix: prefix,
		Posts:  make(Posts, 0),
	}, nil
}

func (c *Contents) Walk(path string, info os.FileInfo, err error) error {
	if !isPostMdfile(path) {
		return nil
	}

	title, err := mdTitle(path)
	if err != nil {
		return errors.Wrap(err, "mdcon: get title")
	}

	url := c.prefix + "/" + path
	year := strings.Split(path, "/")[1] // must */<< year >>/*
	c.Posts = append(c.Posts, Post{title, url, year})

	return nil
}

func (c *Contents) Write(w io.Writer) error {
	_, err := w.Write([]byte("# Blog post\n\n"))
	if err != nil {
		return errors.Wrap(err, "mdcon: write header")
	}

	posts := c.Posts
	sort.Sort(posts)

	var tmpYear string // for year label
	for _, p := range posts {
		if tmpYear != p.year {
			_, err = w.Write([]byte(fmt.Sprintf("## %s\n\n", p.year)))
			if err != nil {
				return errors.Wrap(err, "mdcon: write header")
			}
			tmpYear = p.year
		}

		_, err = w.Write([]byte(fmt.Sprintf("[%s](%s)\n\n", p.title, p.url)))
		if err != nil {
			return errors.Wrap(err, "mdcon: write header")
		}
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

func main() {
	output := flag.String("out", "CONTENTS.md", "output file name")
	root := flag.String("root", "./posts", "posts root directory flag")
	prefix := flag.String("prefix", "", "link prefix")
	difflint := flag.Bool("difflint", false, "contents table diff linter")

	flag.Parse()

	c, err := NewContents(*prefix)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = filepath.Walk(*root, c.Walk)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var w io.Writer
	if *difflint {
		w = bytes.NewBuffer(nil)
	} else {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}

	err = c.Write(w)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if !*difflint {
		fmt.Println("done")
		return
	}

	buf, ok := w.(*bytes.Buffer)
	if !ok {
		fmt.Println("failed to type assertion io.Writer to *bytes.Buffer")
		os.Exit(1)
	}

	body, err := ioutil.ReadFile(*output)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if !bytes.Equal(body, buf.Bytes()) {
		fmt.Println("mismatch!")
		os.Exit(1)
	}

	fmt.Println("ok!")
}
