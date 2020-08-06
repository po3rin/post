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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

const maxRetry = 10

var (
	url       string
	agentMode bool

	oldRevision string
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
	syncCmd.Flags().BoolVarP(&agentMode, "agent-mode", "a", false, "whether run with agent mode")
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

func gitDiffFiles() ([]string, error) {
	revision, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return nil, err
	}

	if oldRevision == string(revision) {
		return []string{}, nil
	}

	if oldRevision == "" {
		oldRevision = "HEAD"
	}

	// diff from old revision
	out, err := exec.Command("git", "diff", "--name-only", string(oldRevision)).Output()
	if err != nil {
		return nil, err
	}
	oldRevision = string(revision)
	files := strings.Split(string(out), "\n")

	var posts []string
	for _, path := range files {
		if isPostMdfile(path) {
			posts = append(posts, path)
		}
	}
	return posts, nil
}

func sendPost(r request) error {
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

func syncPost(files []string) error {
	for _, filepath := range files {
		source, err := ioutil.ReadFile(filepath)
		if err != nil {
			return err
		}

		m, err := mdMeta(source)
		if err != nil {
			return err
		}

		if m.draft {
			fmt.Printf("passed draft: %+v", m.id)
			continue
		}

		req := request{
			ID:          m.id,
			Title:       m.title,
			Body:        string(source),
			Description: m.description,
			Cover:       m.cover,
			Tags:        m.tags,
			CreatedAt:   m.date,
			UpdatedAt:   time.Now(),
		}

		err = sendPost(req)
		if err != nil {
			return err
		}
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
		var err error
		if !agentMode {
			filepaths, err = allFilePath(workdir)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			err = syncPost(filepaths)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
		defer cancel()
		eg, ctx := errgroup.WithContext(ctx)

		// git helthz
		eg.Go(func() error {
			var retryCounter int
			ticker := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					_, err := exec.Command("git", "status").Output()
					if err != nil {
						if maxRetry < retryCounter {
							return err
						}
						log.Errorf("failed to git command exec: %+v", err)
						retryCounter++
						continue
					}
					retryCounter = 0 // reset
				}
			}
		})

		// agent cycle
		eg.Go(func() error {
			ticker := time.NewTicker(10 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()

				case <-ticker.C:
					filepaths, err = gitDiffFiles()
					if err != nil {
						log.Errorf("get diff files: %+v", err)
						continue
					}
					err = syncPost(filepaths)
					if err != nil {
						log.Errorf("sync post: %+v", err)
						continue
					}
				}
			}
		})

		quit := make(chan os.Signal, 1)
		defer close(quit)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		log.Info("start sync-agent")

		select {
		case <-quit:
			cancel()
		case <-ctx.Done():
		}

		if err := eg.Wait(); err != nil {
			log.Error(err)
		}

		log.Info("done")
	},
}
