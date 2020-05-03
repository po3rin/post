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
	"path/filepath"
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
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Description string    `json:"description"`
	Cover       string    `json:"cover"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func allFilePath(workdir string) ([]string, error) {
	var result []string
	err := filepath.Walk(workdir, func(path string, info os.FileInfo, err error) error {
		if !isPostMdfile(path) {
			return nil
		}
		result = append(result, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
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

// newCmd represents the new command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync post external store",
	Long:  "sync post external store",
	Run: func(cmd *cobra.Command, args []string) {
		var filepaths []string
		if all {
			allpath, err := allFilePath(workdir)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(allpath)
			filepaths = allpath
		} else {
			filepaths = args
		}

		for _, filepath := range filepaths {
			source, err := ioutil.ReadFile(filepath)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			m, err := mdMeta(source)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			req := request{
				ID:          m.id,
				Title:       m.title,
				Body:        string(source),
				Description: m.description,
				Tags:        m.tags,
				CreatedAt:   m.date,
				UpdatedAt:   time.Now(),
			}

			err = syncPost(req)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	},
}
