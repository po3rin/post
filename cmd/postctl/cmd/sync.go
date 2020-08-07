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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

const maxRetry = 10

var (
	url, root string
	agentMode bool
	port      int

	oldRevision string
)

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().StringVarP(&url, "url", "u", "", "API URL which handles posts")
	syncCmd.Flags().StringVarP(&root, "root", "r", "", "Git repository root")
	syncCmd.Flags().IntVarP(&port, "port", "p", 9300, "API port")
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

// newCmd represents the new command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync post external store",
	Long:  "sync post external store",
	Run: func(cmd *cobra.Command, args []string) {
		var filepaths []string
		var err error
		if !agentMode {
			filepaths, err = postFiles(workdir)
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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eg, ctx := errgroup.WithContext(ctx)

		// git helthz with retry
		eg.Go(func() error {
			var retryCounter int
			ticker := time.NewTicker(30 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					cmd := exec.Command("git", "status")
					cmd.Dir = root
					out, err := cmd.CombinedOutput()
					if err != nil {
						if maxRetry < retryCounter {
							return err
						}
						log.Errorf("git command exec: %+v, msg: %+v", err, string(out))
						retryCounter++
						continue
					}
					retryCounter = 0 // reset
				}
			}
		})

		// sync hook API
		eg.Go(func() error {
			s := newServer(string(port))
			return s.run(ctx)
		})

		// self sync cycle
		eg.Go(func() error {
			ticker := time.NewTicker(1 * time.Hour)
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()

				case <-ticker.C:
					err = syncDiff()
					if err != nil {
						log.Errorf("self sync cycle: %+v", err)
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

func postFiles(workdir string) ([]string, error) {
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

func gitDiffPostFiles() ([]string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	revision, err := cmd.CombinedOutput()
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
	cmd = exec.Command("git", "diff", "--name-only", string(oldRevision))
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
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

		reqJSON, err := json.Marshal(req)
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
		log.Infof("sync: %+v", m.id)
	}
	return nil
}

func syncDiff() error {
	filepaths, err := gitDiffPostFiles()
	if err != nil {
		return err
	}
	err = syncPost(filepaths)
	if err != nil {
		return err
	}
	return nil
}

type server struct {
	server http.Server
}

func router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET"},
		AllowHeaders: []string{"Origin", "Content-Type"},
	}))

	rg := r.Group("api/v1")
	{
		rg.GET("/healthz", func(c *gin.Context) {
			c.JSON(http.StatusOK, "ok!")
		})
		rg.GET("/sync", func(c *gin.Context) {
			err := syncDiff()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
			}
			c.JSON(http.StatusOK, gin.H{"msg": "success data sync"})
		})
	}
	return r
}

func newServer(port string) *server {
	return &server{
		server: http.Server{
			Addr:    port,
			Handler: router(),
		},
	}
}

func (s *server) run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.server.ListenAndServe()
	})

	<-ctx.Done()
	sCtx, sCancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	defer sCancel()
	if err := s.server.Shutdown(sCtx); err != nil {
		return err
	}

	return eg.Wait()
}
