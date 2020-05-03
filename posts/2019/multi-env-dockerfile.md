---
title: 1つのDockerfileだけでGoの開発環境(ホットリロード)と本番環境(マルチステージビルド)を記述する
cover: img/gopher.png
date: 2019/05/11
id: multi-env-dockerfile
description: Dockerfile1つでGoの開発環境(ホットリロード)と本番環境(マルチステージビルド)を記述する方法を紹介します。
tags:
    - Go
    - Docker
---

こんにちは。po3rinです。今回は[Docker Meetup Tokyo #29 (Docker Bday #6)](https://dockerjp.connpass.com/event/122084/)で少し話題になった小ネタです。タイトル通りDockerfile1つでGoの開発環境(ホットリロード)と本番環境(マルチステージビルド)を記述する方法を紹介します。今回は「この方法をおすすめします！」というよりかは「こういう方法もあるよー」という紹介なので、開発の状況に合わせて方法を選んでいくと良いでしょう。

## イントロ

開発環境用と本番環境でイメージビルド過程を分けるモチベーションとしては、開発環境用はホットリロードしたいけど、本番はビルドしたバイナリだけを使いたいという思いなどがあります。

これらを２つのDockerfileに分ける場合、同じディレクトリ階層に「Dockerfile」という名前のファイルを２つ置けません。これに関して、下記の記事のように```docker build```の```-f```フラッグで任意の名前のDockerfileを指定してビルドするという対処法があります。
https://www.kabegiwablog.com/entry/2018/08/01/120000

当然これでも対処できますが、これだといつも使っているエディターでシンタックスハイライトの恩恵を受けれないなどの様々なストレスポイントが発生します。Dockerfileの階層を分ける方法もありますが、Dockerfileの数だけディレクトリがどんどん分割されていくのも開発の状況によっては見通しが悪くなりそうです。

実はこれらを１つのDockerfileで記述する方法があります。今回はそれを紹介します。

## Goを例にDockerfile1つで開発環境と本番環境を記述する

ポイントは```docker build```がサポートする```--target```フラグです。これでビルド対象のステージ名を指定して、ビルドするステージを分岐できます。Goの例を見てみましょう。ディレクトリ構成も含めた具体的な例は僕のリポジトリをみてください。
https://github.com/po3rin/go_playground/tree/master/try-docker-target

```Dockerfile
// 本番用の中間ステージ
FROM golang:1.12 as builder
WORKDIR /go/api
COPY . .
ENV GO111MODULE=on
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

// 本番用ステージ
FROM alpine:latest as prod
EXPOSE 8080
RUN apk --no-cache add ca-certificates
WORKDIR /api
COPY --from=builder /go/api/ .
RUN pwd
CMD ["./try-docker-target"]

// 開発環境用ホットリロード
FROM golang:1.12 as dev
EXPOSE 8080
WORKDIR /go/api
ENV GO111MODULE=on
COPY . .
RUN go get github.com/pilu/fresh
CMD ["fresh"]
```

上２つが本番用のマルチステージビルド用のステージで、一番下のが開発用ホットリロード環境のステージです。このように記載すると下記のように```--target```でビルドしたいステージを指定できます。

```bash
// 開発環境用ビルド
$ docker build -t api-dev --target dev .
// 本番環境用ビルド
$ docker build -t api-prod --target prod .
```

素晴らしいのは最終的に欲しいステージを指定すれば、ステージ間の依存を解決してビルドしてくれる点です。```--target prod```は依存する```builder stage```を解決してビルドしてくれています。

target指定がめんどくさい&チームで運用する場合はMekefileに記載しておくと良いでしょう。

```Makefile
all: dev prod

dev:
	docker build -t api-dev --target dev .

prod:
	docker build -t api-prod --target prod .
```

この方法のデメリットとしてはDokcerfileが長くなることです。しかし、今回のように開発&本番の2パターンであれば1つにまとめてのそこまで可読性が下がることはないでしょう。このやり方で厳しさがある場合はイントロで紹介した```docker build```の```-f```フラッグで任意の名前のDockerfileを指定してビルドする方法か、そもそもDockerfileのディレクトリ階層を分ける方法を検討ください。

以上ですー

