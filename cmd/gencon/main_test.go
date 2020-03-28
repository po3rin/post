package main

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		goldenpath string
		posts      Posts
	}{
		{
			name:       "write_test",
			prefix:     "test.com",
			goldenpath: "testdata/golden.md",
			posts: Posts{
				Post{
					title: "What's Go language",
					url:   "test.com/post/182",
					year:  "2018",
				},
				Post{
					title: "What's Rust language",
					url:   "test.com/post/183",
					year:  "2018",
				},
				Post{
					title: "What's Python language",
					url:   "test.com/post/171",
					year:  "2017",
				},
				Post{
					title: "What's Ruby language",
					url:   "test.com/post/191",
					year:  "2019",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewContents(tt.prefix)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			c.Posts = tt.posts
			w := bytes.NewBuffer(nil)
			c.Write(w)

			body, err := ioutil.ReadFile(tt.goldenpath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !bytes.Equal(body, w.Bytes()) {
				t.Fatalf("want:\n%v\ngot:\n%v\n", string(body), w.String())
			}
		})
	}
}

func TestMdTitle(t *testing.T) {
	testmd := "testdata/test.md"
	want := "How to test package"

	title, err := mdTitle(testmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want != title {
		t.Fatalf("want:\n%v\ngot:\n%v\n", want, title)
	}
}
