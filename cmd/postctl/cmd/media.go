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
	"fmt"
	"os"
	"path/filepath"

	"github.com/po3rin/post/cmd/postctl/store"
	"github.com/spf13/cobra"
)

var (
	bucket string
)

func init() {
	rootCmd.AddCommand(mediaCmd)
	mediaCmd.Flags().StringVarP(&bucket, "bucket", "b", "", "S3 bucket name")
}

func isSupportedMedia(path string) bool {
	e := filepath.Ext(path)
	return e == ".png" || e == ".jpeg" || e == ".jpg" || e == ".gif"
}

// mediaCmd represents the media command
var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "controls media",
	Long:  "controls media",
	Run: func(cmd *cobra.Command, args []string) {
		filepath := args[0]
		if filepath == "" {
			fmt.Println("filepath arg is required")
			os.Exit(1)
		}

		if !isSupportedMedia(filepath) {
			fmt.Println("unsupported format file")
			os.Exit(1)
		}
		resutl, err := store.New(bucket, "media").Upload(filepath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("-------------")
		fmt.Println(filepath)
		fmt.Println("↓")
		fmt.Println(resutl)
		fmt.Println("-------------")

		err = os.Remove(filepath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
