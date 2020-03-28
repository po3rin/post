package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/po3rin/post/cmd/mediactl/store"
)

func isSupportedMedia(path string) bool {
	e := filepath.Ext(path)
	return e == ".png" || e == ".jpeg" || e == ".jpg" || e == ".gif"
}

type Uploader interface {
	Upload(path string) (string, error)
}

type Walker struct {
	Uploader
	mdpath string
}

func NewWalker(uploader Uploader, dir string) *Walker {
	return &Walker{
		Uploader: uploader,
		mdpath:   filepath.Join(dir, "post.md"),
	}
}

func (w *Walker) Walk(path string, info os.FileInfo, err error) error {
	if !isSupportedMedia(path) {
		return nil
	}

	res, err := w.Upload(path)
	if err != nil {
		return errors.Wrap(err, "upload media")
	}

	b, err := ioutil.ReadFile(w.mdpath)
	if err != nil {
		return errors.Wrap(err, "read old file")
	}

	new := bytes.ReplaceAll(b, []byte(path), []byte(res))

	f, err := os.Create(w.mdpath)
	if err != nil {
		return errors.Wrap(err, "create new file")
	}

	_, err = f.Write(new)
	if err != nil {
		return errors.Wrap(err, "write new body")
	}

	fmt.Println("----------------------")
	fmt.Printf("%v\n↓\n%v\n", path, res)

	// err = os.Remove()
	// if err != nil {
	// 	return "", errors.Wrap(err, "delete file")
	// }

	return nil
}

func main() {
	id := flag.String("id", "", "post identifier (ex 2019/<< unixtime >>")
	dir := flag.String("dir", "posts", "posts directory path")
	bucketName := flag.String("bucket", "", "S3 bucket name")
	flag.Parse()

	postdir := filepath.Join(*dir, *id)
	imgpath := filepath.Join(postdir, "img")

	s := store.New(*bucketName, *id)
	w := NewWalker(s, postdir)

	err := filepath.Walk(imgpath, w.Walk)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
