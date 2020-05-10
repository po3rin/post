# :pencil: posts of pon tech blog

![Go Status](https://github.com/po3rin/post/workflows/Go%20Status/badge.svg) ![Contents Status](https://github.com/po3rin/post/workflows/Contents%20Status/badge.svg)

This repository manages my texh blog post.
* generates Markdown file.
* creates tech blog contents.
* stores post's media to datastore (Now supports only S3)

## :star: All posts  is here.
https://github.com/po3rin/post/blob/master/CONTENTS.md

## :triangular_flag_on_post: Contributing

Did you find something technically wrong, something to fix, or something? Please give me Issue or Pull Request !!

## :triangular_ruler: Usage

this repository provides ```postctl``` cli

```bash
go get -u github.com/po3rin/post/cmd/postctl
```

#### Write new post

creates new Markdown file for blog.

```bash
$ postctl new < id >
```

#### Manage media

postctl media subcommand uploads media to S3 and replace S3 object url from loacl retrive path to media.

```bash
$ postctl media -b < bucket name>  < image file path>

----------------------
img/test.jpeg
↓
https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/test.jpeg
----------------------
```

#### Sync contents table

sync contens table of post

```bash
$ postctl contents -p "https://github.com/po3rin/post/tree/master"
```

#### Sync Posts with External Database

sync all posts

```bash
$ postctl sync -u http://localhost:8081/post -a
```

specify id

```bash
$ postctl sync -u http://localhost:8081/post filepath
```
