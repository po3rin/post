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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const indexName = "blog"

var (
	endpoint, user, pass string
)

// contentsCmd represents the contents command
var esSyncCmd = &cobra.Command{
	Use:   "essync",
	Short: "post blog contens to es",
	Long:  "post blog contens to es",
	Run: func(cmd *cobra.Command, args []string) {
		// sync all process
		var filepaths []string
		var err error

		filepaths, err = allPostFiles(filepath.Join(root, workdir))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		es, err := newEs(user, pass, endpoint)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		ctx := context.Background()
		err = es.syncEsPost(ctx, filepaths)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(esSyncCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// contentsCmd.PersistentFlags().String("foo", "", "A help for foo")
	esSyncCmd.Flags().StringVarP(&root, "root", "r", "", "Git repository root")

	esSyncCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "http://localhost:9200", "Elasticsearch Endpoint")

	esSyncCmd.Flags().StringVarP(&user, "user", "u", "", "Elasticserach User")

	esSyncCmd.Flags().StringVarP(&pass, "pass", "p", "", "Elasticsearch Pass")
}

type es struct {
	client *elastic.Client
}

func newEs(user, pass string, urls ...string) (*es, error) {
	var sniff bool
	if len(urls) > 1 {
		sniff = true
	}

	var client *elastic.Client
	var err error
	operation := func() error {
		client, err = elastic.NewClient(
			elastic.SetURL(urls...),
			elastic.SetSniff(sniff),
			// elastic.SetTraceLog(log.New(os.Stdout, "", log.LstdFlags)),
			elastic.SetBasicAuth(user, pass),
		)
		if err != nil {
			return errors.Wrap(err, "new Elasticsearch Client with retry")
		}
		return nil
	}

	err = backoff.Retry(operation, backoff.WithMaxRetries(
		backoff.NewExponentialBackOff(),
		10,
	))
	if err != nil {
		return nil, errors.Wrap(err, "new Elasticsearch client")
	}
	return &es{client: client}, nil
}

func (e *es) syncEsPost(ctx context.Context, files []string) error {
	for _, f := range files {
		source, err := ioutil.ReadFile(f)
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
			IsExternal:  m.isExternal,
			ExternalURL: m.externalURL,
		}

		_, err = e.client.Index().
			Index(indexName).
			Id(req.ID).
			BodyJson(req).
			Do(ctx)
		if err != nil {
			return err
		}
		log.Infof("sync: %+v", m.id)
	}
	return nil
}
