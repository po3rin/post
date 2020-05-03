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
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var (
	timef string
)

func init() {
	rootCmd.AddCommand(newCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// newCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	newCmd.Flags().StringVarP(&timef, "time", "t", "", "new post date time (format: 2006-01-02)")
}

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "generate new post file",
	Long:  "generate new post file",
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		if id == "" {
			fmt.Println("id is required")
			os.Exit(1)
		}

		t := time.Now()
		year := strconv.Itoa(t.Year())
		dirpath := filepath.Join(workdir, year)

		err := createDir(dirpath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		mdname := id + ".md"
		p := filepath.Join(dirpath, mdname)

		f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer f.Close()

		template := fmt.Sprintf(`---
title: Try Go
cover: img/gopher.png
date: %+v
id: %s
description: Go is a programming language
---

## Overview
`, t.Format("2006/01/02"), id)

		_, err = f.Write([]byte(template))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
