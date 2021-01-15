package cmd

import (
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

func TestMdMeta(t *testing.T) {
	layout := "2006/01/02"
	date, _ := time.Parse(layout, "2019/05/03")

	tests := []struct {
		name     string
		filepath string
		want     metaData
	}{
		{
			name:     "draft",
			filepath: "testdata/withmeta.md",
			want: metaData{
				title:       "Try Go",
				cover:       "img/gopher.png",
				date:        date,
				id:          "dsds",
				description: "Go is a programming language that makes it easy to build simple",
				tags: []string{
					"golang", "markdown",
				},
				draft: true,
			},
		},
		{
			name:     "external",
			filepath: "testdata/external.md",
			want: metaData{
				title:       "Try Go",
				cover:       "img/gopher.png",
				date:        date,
				id:          "dsds",
				description: "Go is a programming language that makes it easy to build simple",
				tags: []string{
					"golang", "markdown",
				},
				isExternal:  true,
				externalURL: "xxx.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ioutil.ReadFile(tt.filepath)
			if err != nil {
				t.Fatalf("unexpected err: %+v", err)
			}

			got, err := mdMeta(input)
			if err != nil {
				t.Fatalf("unexpected err: %+v", err)
			}

			if !reflect.DeepEqual(tt.want, got) {
				t.Fatalf("\nwant: %+v\ngot : %+v\n", tt.want, got)
			}
		})
	}
}
