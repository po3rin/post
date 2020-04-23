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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	url string
	all bool
)

func init() {
	rootCmd.AddCommand(syncCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// newCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	syncCmd.Flags().StringVarP(&url, "url", "u", "", "API URL which handles posts")
	syncCmd.Flags().BoolVarP(&all, "all", "a", false, "whether taget is all posts")
}

type request struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func allID(workdir string) ([]string, error) {
	var years []string
	info, err := ioutil.ReadDir(workdir)
	if err != nil {
		return nil, err
	}
	for _, file := range info {
		if !file.IsDir() {
			continue
		}
		years = append(years, file.Name())
	}

	var ids []string
	for _, year := range years {
		info, err := ioutil.ReadDir(path.Join(workdir, year))
		if err != nil {
			return nil, err
		}

		for _, id := range info {
			if !id.IsDir() {
				continue
			}
			ids = append(ids, id.Name())
		}
	}
	return ids, nil
}

func syncPost(r request) error {
	reqJSON, err := json.Marshal(r)
	if err != nil {
		return err
	}
	res, err := http.Post(url, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return errors.New("status is not OK")
	}
	return nil
}

// TODO: use markdown parser.
func titleBodyInMD(filepath string) (title, body string, err error) {
	if !isPostMdfile(filepath) {
		return "", "", errors.New("is not Markdown")
	}

	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", "", err
	}

	if !strings.HasPrefix(string(b), "# ") {
		return "", "", fmt.Errorf("invalid title format in %s", filepath)
	}

	lines := strings.Split(string(b), "\n")
	if len(lines) == 0 {
		return "", "", errors.New("file is empty")
	}

	title = strings.ReplaceAll(lines[0], "# ", "")
	body = strings.Join(lines[1:], "\n")

	for {
		if strings.HasPrefix(body, "\n") {
			body = body[1:]
			continue
		}
		break
	}

	return title, body, nil
}

// newCmd represents the new command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync post external store",
	Long:  "sync post external store",
	Run: func(cmd *cobra.Command, args []string) {
		var ids []string
		if all {
			all, err := allID(workdir)
			fmt.Println(all)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			ids = all
		} else {
			ids = args
		}

		for _, id := range ids {
			u, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			t := time.Unix(u, 0)
			year := t.Year()
			yearStr := strconv.Itoa(year)
			postPath := path.Join(workdir, yearStr, id, "post.md")

			title, body, err := titleBodyInMD(postPath)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			req := request{
				ID:        id,
				Title:     title,
				Body:      body,
				CreatedAt: t,
				UpdatedAt: time.Now(),
			}

			err = syncPost(req)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	},
}
