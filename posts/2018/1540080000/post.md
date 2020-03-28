# Go v1.11 + Docker + fresh でホットリロード開発環境を作って愉快なGo言語生活

Go言語 + Docker で開発することが多くなってきました。ファイル更新の度にローカルでサーバー動作確認して、コンテナで動作確認してーなんてのはめんどくさいので、
Dockerコンテナ内とローカルで volume を貼ってホットリロードできる開発環境を試したので記事にします。

GitHub にも 試したプロジェクトは置いてあります。
https://github.com/po3rin/go-playground/tree/master/hot_reload_docker



## 構成
とりあえずローカルでの構成を参考に書いときます。プロジェクト名などは柔軟に。

```
hot_reload_docker
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── main.go
```

## ローカルでの作業

go1.11 から出た modules を使っています。go　のバージョンが1.11以上になっているか確認してください。

```bash
$ go version
go version go1.11.1 darwin/amd64
```

modules を使う手順もさらっと紹介していきますが、さらっとなので、くわしくは下記のサイトなどを参考に！Wantedlyさんの記事です。
> Go 1.11 の modules・vgo を試す - 実際に使っていく上で考えないといけないこと #golang
> https://www.wantedly.com/companies/wantedly/post_articles/132270

まず環境変数 GO111MODULE=on を設定します。

```bash
export GO111MODULE=on
```

これで module が有効になります。
go のプロジェクト内で下記を打ち込み、```go.mod```を作ります。

```
$ go mod init
```

これで適当に main.go を作ります。

```go:main.go
package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "ping",
		})
	})
	r.Run(":8083")
}
```

ここで

```bash
$ go run main.go
```

をすると サーバー軌道の前にパッケージの読み込みが行われます。modules では```go test```や```go build```などのタイミングで
必要なパッケージの読み込みを自動で行ってくれます。便利！とりあえず動作確認しておきましょう。

```bash
$ curl localhost:8083
{"message":"ping"}
```

いいですね。

## Docker化 と ホットリロード

まずは Dockerfile です。

```docker:Dockerfile

FROM golang:1.11.1

WORKDIR /go/src/hot_reload_docker
COPY . .
ENV GO111MODULE=on

RUN go get github.com/pilu/fresh
CMD ["fresh"]
```

ポイントは環境変数　GO111MODULE=on　の設定ですね。いままで　Dockerfile 内で　```go get```とか```dep ensure```とかやっていたのはもういりません。```go test```や```go build```などのタイミングで必要なパッケージの読み込みを自動で行ってくれるからです。

そして今回は```github.com/pilu/fresh``` を導入しています。これはファイルを監視し、更新されたらホットリロードしてくれます。ゆえに今回は```fresh```コマンドでサーバーを起動します。

あとはローカルでのファイル更新とコンテナ内のファイル更新を連打させるだけです。これは Docker の volumes という昨日を使います。
管理しやすいように　docker-compose を採用します。```docker-compose.yml``` を作りましょう。

```yaml:docker-compose.yml
version: '3'
services:
  app:
    build: .
    volumes:
      - ./:/go/src/hot_reload_docker
    ports:
      - "8083:8083"
```

ポイントは volumes です。```<<ローカルのディレクトリパス>>:<<コンテナのディレクトリパス>>```でvolumeをはり、ローカルのファイル更新でコンテナ内のファイルを更新するようにします。これでローカルでファイルを編集すれば、コンテナ内で起動しているサーバーがホットリロードされます。

## 動作確認

さっそく動作確認してみましょう。

Dockerコンテナ起動

```
$ docker-compose up -d
```

動作確認

```
$ curl localhost:8083
{"message":"ping"}
```

動いてますね。ではローカルでファイルを更新してみましょう。ping を pong に変えました。

```go:main.go
// ...

func main() {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Run(":8083")
}
```
ファイルを更新したらホットリロードされます。数秒更新に時間がかかるので少し待ったのち、動作確認しましょう。

```bash
curl localhost:8083
{"message":"pong"}
```

ping が pong になりました。これで愉快なGo言語生活を送れます。

GitHub にも 試したプロジェクトは置いてあります。
https://github.com/po3rin/go-playground/tree/master/hot_reload_docker

