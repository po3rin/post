# :pencil: posts of pon tech blog

![Go Status](https://github.com/po3rin/post/workflows/Go%20Status/badge.svg) ![Contents Status](https://github.com/po3rin/post/workflows/Contents%20Status/badge.svg)

This repository manages my texh blog post.
* generates Markdown file.
* creates tech blog contents.
* stores posts to datastore (TODO)

## :star: All posts  is here.
https://github.com/po3rin/post/blob/master/CONTENTS.md

## :triangular_flag_on_post: Contributing

Did you find something technically wrong, something to fix, or something? Please give me Pull Request !!

## :triangular_ruler: Usage

#### Write new post

creates new Markdown file for blog.

```bash
# if you already installed go
$ go get -u github.com/po3rin/post/cmd/newpost

$ newpost
# or
$ make new
```

#### Manage media

mediactl upload media to S3 and replace S3 object url from loacl retrive path to media.

```bash
# if you already installed go
$ go get -u github.com/po3rin/post/cmd/mediactl

$ go run cmd/mediactl/main.go -bucket < bucket name> -id < post id (year/unixtime) >
target is posts/<year>/<unixtime>
----------------------
img/test.jpeg
↓
https://< bucket name >.s3.ap-northeast-1.amazonaws.com/year/unixtime/test.jpeg
----------------------
```

#### Sync contents table

sync contens table of post

```bash
# if you already installed go
$ go get -u github.com/po3rin/post/cmd/gencon

$ gencon
# or
$ make contents
```
