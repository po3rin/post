package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
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

type metaData struct {
	id          string
	title       string
	cover       string
	description string
	date        time.Time
	tags        []string
	draft       bool
}

func mdMeta(md []byte) (metaData, error) {
	markdown := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
		),
	)

	var buf bytes.Buffer
	context := parser.NewContext()
	if err := markdown.Convert(md, &buf, parser.WithContext(context)); err != nil {
		return metaData{}, err
	}

	m := meta.Get(context)
	id, ok := m["id"]
	if !ok {
		return metaData{}, errors.New("no id")
	}
	title, ok := m["title"]
	if !ok {
		return metaData{}, errors.New("no title")
	}
	cover, ok := m["cover"]
	if !ok {
		return metaData{}, errors.New("no cover")
	}
	description, ok := m["description"]
	if !ok {
		return metaData{}, errors.New("no description")
	}
	date, ok := m["date"]
	if !ok {
		return metaData{}, errors.New("no date")
	}
	layout := "2006/01/02"
	t, err := time.Parse(layout, fmt.Sprintf("%v", date))
	if err != nil {
		return metaData{}, errors.New("unsupported format ( required format 2006/01/02 )")
	}
	tags, ok := m["tags"]
	if !ok {
		return metaData{}, errors.New("no tag")
	}
	tagsSlice, err := interface2Slice(tags)
	if err != nil {
		return metaData{}, errors.Wrap(err, "could not assert tags interface to slice")
	}

	meta := metaData{
		id:          fmt.Sprintf("%v", id),
		title:       fmt.Sprintf("%v", title),
		description: fmt.Sprintf("%v", description),
		cover:       fmt.Sprintf("%v", cover),
		date:        t,
		tags:        tagsSlice,
	}

	d, ok := m["draft"]
	if !ok {
		return meta, nil
	}
	draft, ok := d.(bool)
	if !ok {
		return metaData{}, errors.New("draft field requires boolean")
	}
	meta.draft = draft

	return meta, nil
}

func interface2Slice(slice interface{}) ([]string, error) {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return nil, errors.New("is not slice")
	}

	ret := make([]string, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = fmt.Sprintf("%v", s.Index(i))
	}

	return ret, nil
}
