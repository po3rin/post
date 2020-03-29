/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/po3rin/post/cmd/postctl/store"
	"github.com/spf13/cobra"
)

// mediaCmd represents the media command
var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "controls media",
	Long:  "controls media",
	Run: func(cmd *cobra.Command, args []string) {
		postdir := filepath.Join(workdir, id)
		imgpath := filepath.Join(postdir, "img")

		s := store.New(bucket, id)
		w := NewWalker(s, postdir)

		fmt.Printf("target is %v\n", postdir)
		err := filepath.Walk(imgpath, w.Walk)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

var (
	id, bucket string
)

func init() {
	rootCmd.AddCommand(mediaCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mediaCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mediaCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	mediaCmd.Flags().StringVarP(&id, "id", "i", "", "post identifier (format: << year >>/<< unixtime >>")
	mediaCmd.Flags().StringVarP(&bucket, "bucket", "b", "", "S3 bucket name")
}

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

	imgrtr := filepath.Join("img", filepath.Base(path))
	fmt.Println("----------------------")
	fmt.Printf("%v\n↓\n%v\n", imgrtr, res)

	new := bytes.ReplaceAll(b, []byte(imgrtr), []byte(res))

	f, err := os.Create(w.mdpath)
	if err != nil {
		return errors.Wrap(err, "create new file")
	}

	_, err = f.Write(new)
	if err != nil {
		return errors.Wrap(err, "write new body")
	}

	// delete local media file.
	err = os.Remove(path)
	if err != nil {
		return errors.Wrap(err, "delete file")
	}

	return nil
}
