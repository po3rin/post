# 「Vue.js + Go言語 + Docker」で作る！画像アップロード機能実装ハンズオン

<img width="812" alt="cover.png" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/qiita-c70105f684e6816621d2-1.png">

こんにちはpo3rinです。Vue.js Advent Calender 2018 9日目の記事です。
8日目の記事は [vue.js(nuxt.js) の plugin はとても便利](https://qiita.com/waterada/items/9ae9d977a543bda1214f) でした。

11月にフリーの案件で Vue.js + Go言語で画像アップロード機能のあるCMSを作りました。Vue.jsでの実装の際には npmモジュールである ```vue2-dropzone``` を使うと、Vue.js にとって便利な機能が提供されており、すぐにアップロード機能が作れました。なので今回は Vue.js + Go言語 で画像アップロードを行う機能の実装をハンズオン形式で紹介していきます。

今回は Vue.js のアドベントカレンダーとしての投稿なので、Go言語の実装を飛ばしたい方向けに、Go言語のインストールが不要になるように、すでにDocker環境を用意してあります。せっかくなので今回は Docker を使った　開発環境構築も紹介します。

Go言語の実装を飛ばしたい方は、下記のリポジトリから server ディレクトリをローカルに置いておけば大丈夫です。

今回の実装の Github リポジトリはこちら！
 <a href="https://github.com/po3rin/vue-go-image-uploader"><img width="460px" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/qiita-c70105f684e6816621d2-2.png" /></a>

## vue2-dropzone の概要

![dropzone-js-logo.png](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/qiita-c70105f684e6816621d2-3.png)

Dropzone.js を使用したファイルアップロード用のVueコンポーネントです。Vue.jsに特化されており、めちゃくちゃ便利です。今回はこの ```vue2-dropzone``` の機能を数多く使うので、こちらのドキュメントを参照しながらの実装をお勧めします。
https://rowanwins.github.io/vue-dropzone/docs/dist/#/installation

## 今回の目指す形

構成は単純です。Dockerコンテナで Vue.js でクライアント、Go言語でAPIサーバーを実装します。

<img width="512" alt="v-g.png" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/qiita-c70105f684e6816621d2-4.png">

下のように Vue.js + Go言語で画像をアップロードを作ってみます。

<img width="512" alt="v-g.png" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/5b492d1b-87bc-d535-867a-a4224cd44255.gif">

当然削除やリスト機能も付けます。

## 開発用環境構築

今回は Docker を使った　開発環境構築を紹介します。開発環境をDockerで作れば、他のメンバーもすぐに同じ環境を再現できる & グローバルが汚れにくい & 本番環境も立てやすくなるので便利です。一方で Docker の準備がめんどくさい & 自分で準備可能な場合はこのセッションは飛ばしてもらっても大丈夫です。https://qiita.com/po3rin/items/c70105f684e6816621d2#vue2-dropzone-で-画像アップロードを作ってみる

```bash
├── client # Vue.js によるクライアント
├── server # Go言語による APIサーバー
└── docker-compose.yml # docker-compose 設定ファイル
```

### Vue.js の 開発用 Docker 環境

Vue.jsの開発環境をDockerで作れば、他のメンバーもすぐに同じ環境を再現できる & グローバルが汚れにくい & 本番環境も立てやすくなるので便利です。

まずは Vue CLI 3 が使えることを確認します。もし、入れたことがない場合は下記ドキュメントの手順に従います。

> Vue CLI 3 - Installation
> https://cli.vuejs.org/guide/installation.html

```bash
$ vue --version
3.2.1
```

Vue.js のプロジェクトの雛形を作ります。そのまま作られたディレクトリの中で Dockerfile を作りましょう。今回はシンプルなVueで十分なのでpreset は default を選択しましょう。

```bash
$ vue create client
? Please pick a preset: default (babel, eslint) # <- こちらを選択

$ cd client
$ touch Dockerfile
```

そのままDockerfileに以下の記述を記載します。

```Dockerfile
# 開発環境
FROM node:10.12-alpine as build-stage
WORKDIR /app
COPY . .
RUN yarn install
```

コンテナ内に```/app```ディレクトリを作り、 ローカルの```client```ディレクトリにあるファイルを全てコンテナにコピーし、最後に依存モジュールをインストールします。

これで このイメージにはあと``` yarn serve ```するだけでホットリロード入りの開発環境が走る状態まで出来ました。

### Go言語の開発用 Docker 環境

(Go言語はパスしたい人や、Vue.jsのセクションだけ学びたい人は、僕の GitHub レポジトリの ```server```ディレクトリだけダウンロードすれば 完成形APIサーバーが動きます。)

```client``` ディレクトリと同じ階層に ```server``` ディレクトリを作成します

```
$ mkdir server
$ cd server
```

今回は Go1.11から導入された vgo + modules を使って環境構築します。vgo が入っていることを確認しましょう。入ってない人は下記の手順に従ってください。

> Tour of Versioned Go (vgo)
> https://research.swtch.com/vgo-tour

```bash
$ vgo help
Go is a tool for managing Go source code.
```

そして ```go.mod``` と ```main.go``` を作ります。 

```bash
$ touch go.mod main.go
```

```server/main.go``` を記載します。とりあえずはpingを返すだけで良いでしょう。

```go:main.go
package main // import "server"

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "ping",
		})
	})
	r.Run(":8888")
}
```

あとは ```main.go``` と同じ階層にDockerfileを作ります。

```Dockerfile
# 開発環境
FROM golang:1.11

WORKDIR /api
COPY . .
ENV GO111MODULE=on

RUN go get github.com/pilu/fresh
CMD ["fresh"]
```

ここでは ```github.com/pilu/fresh``` パッケージでAPIがホットリロードで起動します。ファイルを更新したらそのまま API も更新されます。これで Docker 環境が整いました。

### docker-compose で起動してみる

```docker-compose.yml``` を作りましょう。

```yml:docker-compose.yml 

version: '3'

services:
  client:
    build: ./client
    ports:
      - 8080:8080
    volumes:
      - ./client:/app
    command: yarn serve

  server:
    build: ./server
    ports:
      - 8888:8888
    volumes:
      - ./server:/api
```

これで起動するはずです。ホットリロードはファイルの更新を監視している為、 volumes でローカルのファイルとコンテナ内のファイルを同期させれば、そのままローカルで作業できます。

```bash
$ docker-compose up -d
```

これで環境が整いました。実際に動くか試してみましょう。

client: [localhost:8080](localhost:8080)
server: [localhost:8000](localhost:8000)

client に関しては実際にブラウザで、server に関しては curlコマンドで確認してみましょう。

```bash
$ curl localhost:8888
{"message":"ping"}
```

これでDockerによる開発環境が構築出来ました。止める際には下記のコマンドを実行しましょう。

```bash 
$ docker-compose down
```

## vue2-dropzone で 画像アップロードを作ってみる

まずは モジュールである vue2-dropzone をインストールします。ついでにサーバーにリクエストを送るための ```axios``` も後で使うので入れておきます。

```bash
$ npm install vue2-dropzone axios
```

これで 画像をアップロードできる便利なコンポーネントを使えます。早速 ```client/src/components/HelloWorld.vue``` に ```vue2-dropzone```　を追加しましょう。

```vue:HelloWorld.vue 
<template>
  <div class="hello">
    <vue-dropzone ref="myVueDropzone" id="dropzone" :options="dropzoneOptions"></vue-dropzone>
  </div>
</template>

<script>
// vue2-dropzone と vue2-dropzone用のcssをimport
import vue2Dropzone from 'vue2-dropzone'
import 'vue2-dropzone/dist/vue2Dropzone.min.css'

export default {
  name: 'HelloWorld',
  data: function () {
    return {
      dropzoneOptions: {
        url: `http://localhost:8888/images`,
        method: 'post'
      }
    }
  },
  components: {
    vueDropzone: vue2Dropzone
  }
}
</script>

<!-- CSS 省略 -->

```

```<vue-dropzone>``` コンポーネントは ```:option``` で設定を渡せます。今回は ```dropzoneOptions``` には アップロード対象となるサーバーのエンドポイント ```url``` や アップロード時に使うメソッド ```method``` をセッティングしています。すでにファイルをアップロードできるフォームが出来ています。

<img width="512" alt="v-g.png" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/qiita-c70105f684e6816621d2-5.png">

この他に option に設定できるもののリストはこちらになります。
https://www.dropzonejs.com/#configuration-options

ここまでで画像をドロップもしくは選択できるようになってます。しかしまだアップロード先のサーバーが出来てないので、今から作っていきます。

## Go言語でアップロードを受け取るAPIサーバーを作る。

```main.go``` を修正しましょう。

```go:main.go 
package main // import "server"

import (
	"server/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:8080"},
		AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"*"},
	}))

    r.POST("/images", handler.Upload)
	r.Run(":8888")
}

```

Go言語でJSONを扱うのが少し面倒なのでここでフレームワークの gin を使いましょう。GitHubにあるドキュメントが参考になります。
https://github.com/gin-gonic/gin

```/gin-gonic/gin``` は最近有志による日本語訳のページができたので、日本語が良い方ははこちらから確認できます。
https://gin-gonic.com/ja/

POST メソッドで ```http://localhost:8888/images``` でファイルアップロードを受け付けます。ちなみに ```github.com/gin-contrib/cors``` は gin 用のCORS設定パッケージです。今回は Vue から叩くのでこちらも使います。

実際のリクエストを捌く ```handler.Upload``` メソッドを作りましょう。handler/handler.go を作ります。

```go:handler/handler.go 
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Upload upload files.
func Upload(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File["file"]

	for _, file := range files {
		err := c.SaveUploadedFile(file, "images/"+file.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "success!!"})
}
```

```gin.Context``` にはアップロードされたfile達を処理するための、メソッドがいくつか生えています。

今回は ```gin.Context.SaveUploadedFile``` を使います。

```go
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error
```

指定されたディレクトリにファイルを保存します。その為、```server/images``` ディレクトリを作っておきましょう。

最終的にserver側の構成はこのようになっています。

```bash
.
└── server
    ├── Dockerfile
    ├── go.mod
    ├── go.sum
    ├── handler
    │   └── handler.go
    ├── images # image保存用
    └── main.go
```

これだけで画像アップロードを受けれます。Vue.js で作ったクライアントから画像をアップロードしてみましょう。

<img width="512" alt="v-g.png" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/b97d35a5-9500-0097-6734-817b5b133278.gif">
動きました!

## ファイルの名前の重複防ぐ

さて、ここまでの実装だと、同じ名前の画像を上げてしまうと、前にあった画像が上書きされてしまいます。そのため重複を防ぐために uuid で管理するようにしましょう。

```HelloWorld.vue```を編集しましょう。まずは ```vue-dropzone``` に ```v-on``` ディレクティブで送信直前に発火するイベントを追加しておきます。他のv-onで発火させることのできるイベント一覧はこちらで確認できます。
https://rowanwins.github.io/vue-dropzone/docs/dist/#/events

変更点はコメントアウトで補足しています。

```vue:HelloWorld.vue 
<template>
  <div class="hello">

    <!-- sendingEventを追加 -->
    <vue-dropzone ref="myVueDropzone" id="dropzone" :options="dropzoneOptions"
      v-on:vdropzone-sending="sendingEvent"
    ></vue-dropzone>

  </div>
</template>

<script>
// 省略 ...

export default {
  // 省略 ...

  // methods を追加 formデータとして fileに付けられた任意のuuidを付加
  methods: {
    sendingEvent: function (file, xhr, formData) {
      formData.append('uuid', file.upload.uuid)
    }
  }
}
</script>

<!-- CSS 省略 -->

```

これで ```vue-dropzone``` がファイル別に自動で付けてくれている ```uuid``` をサーバーに送れます。これを使ってサーバー側で画像を判別できます。

それでは ```handler/handler.go``` も uuid で画像を判別できるように編集しましょう

```go:handler/handler.go 
// 省略...

func Upload(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File["file"]

	// uuid を所得
	uuid := c.PostForm("uuid")

	for _, file := range files {
		// ファイル名にuuidを仕込む
		err := c.SaveUploadedFile(file, "images/"+uuid+".png")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		}
	}

	c.String(http.StatusOK, fmt.Sprintf("%d files uploaded!", len(files)))
}
```

client から 貰った uuid を画像名にして保存します。これで重複を防げるようになりました。試してみてください。

## 画像の削除

やはりエンジニアたるもの、アップロードした画像を削除したくなってきます。
```vue2-dropzone``` では削除に便利な機能も提供してます。サクッと削除機能を追加しましょう。

まずは　削除機能を追加します。dropzone の設定に一行加えます。

```vue:HelloWorld.vue 
<!-- 省略 -->

<script>
export default {
  name: 'HelloWorld',
  data: function () {
    return {
      dropzoneOptions: {
        url: 'http://localhost:8888/images',
        method: 'post',
        addRemoveLinks: 'true' // ここに1行追加 !!!
      }
    }
  },
  // 省略...
}
</script>
```

これで ホバーしたら削除ボタンが表示されます。

<img width="512" alt="vue-image2.png" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/qiita-c70105f684e6816621d2-6.png">

削除ボタンを押すと、form から画像が消えます。しかし、画面上から消えるだけで、サーバーにアップロードした画像は消えていません。削除ボタンを推すと同時にサーバーの画像を削除するリクエストを送れるようにしましょう。

```vue2-dropzone``` では削除ボタンを押した際のイベントも、送信時同様、```v-on``` で登録できます。

```vue:HelloWorld.vue 
<template>
  <div class="hello">
<img width="772" alt="vue-image2.png" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/qiita-c70105f684e6816621d2-7.png">

    <!-- removeEventを追加 -->
    <vue-dropzone ref="myVueDropzone" id="dropzone" :options="dropzoneOptions"
      v-on:vdropzone-sending="sendingEvent"
      v-on:vdropzone-removed-file="removeEvent"
    ></vue-dropzone>

  </div>
</template>

<script>
// 省略 ...

// axios の import を追加。
import axios from "axios";

export default {
  // 省略 ...

  // methods を追加。uuidを指定して Delete API を叩く
  methods: {
    sendingEvent: function (file, xhr, formData) {
      formData.append('uuid', file.upload.uuid)
    },
    removeEvent: function (file, error, xhr) {
      axios.delete(`http://localhost:8888/images/${file.upload.uuid}`).then(res => {
        console.log(res.data)
      }).catch(err => {
        console.error(err)
      })
    }
  }
}
</script>

<!-- CSS 省略 -->

```

これで削除時にサーバーにリクエストを送り、画像を削除してもらいます。ではサーバー側で Delete API を作りましょう。

```server/main.go``` を修正します。

```go:main.go 
func main() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:8080"},
		AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"*"},
	}))

    r.POST("/images", handler.Upload)

    // DELETEメソッドを追加
    r.DELETE("/images/:uuid", handler.Delete)

	r.Run(":8888")
}
```

そして ```handler.go``` に Delete ハンドラーを追加します。

```go:handler/handler.go 
// Delete remove file.
func Delete(c *gin.Context) {
	uuid := c.Param("uuid")
	err := os.Remove(fmt.Sprintf("images/%s.png", uuid))
	if err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("id: %s is deleted!", uuid)})
}
```

これで削除も完成しました。




## 画像のリストを表示する

ここまでの実装では1回アップロードしてからページを更新すると、dropzoneから画像情報が消えるので、何がアップロード済みなのか分からなくなります。このセッションでは1度アップロードした画像をいつでも受け取ってdropzoneにアップロード済みファイルの情報を表示できるようにします。

まずは下準備として、静的ファイルがサーバーから配信できるよにしておきましょう。Nginx などを使っても良いですが、今回はGo言語で画像を返すようにします。

```main.go``` を修正します。ginが提供している静的ファイル配信用のパッケージ ```github.com/gin-gonic/contrib/static``` を使います。

```go:main.go 

package main // import "server"

import (
	"server/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
)

func main() {
	// 省略 ...

	// 静的ファイル配信を追加!!
	r.Use(static.Serve("/", static.LocalFile("./images", true)))

	r.POST("/images", handler.Upload)
	r.DELETE("/images/:uuid", handler.Delete)
	r.Run(":8888")
}

```

これで localhost:8888/{uuid}.png でアクセスすればその画像が返ってくるようになります。確認してみてください。

それではクライアントで アクセス時に 画像のURLリストをAPIから所得し、すでにアップロードしていた画像を form にまとめて表示するようにしましょう。```HelloWorld.vue``` に追加します。

```vue:HelloWorld.vue 
<script>
export default {
  // 省略 ...

  mounted () {
    axios.get('http://localhost:8888/images').then(res => {
      res.data.forEach(res => {
        // filename 所得
        let filename = res.path.replace('http://localhost:8888/', '')
        // uuid 所得
        let id = filename.replace('.png', '')
        // file オブジェクト作成
        var file = {size: res.size, name: filename, type: "image/png", upload: {uuid: id}}
        // コードから　form に画像データをセット
        this.$refs.myVueDropzone.manuallyAddFile(file, res.path)
      })
    }).catch(err => {
      console.error(err)
    })
  },

  // 省略 ...
}
</script>
```

Vue.js の機能である ```mounted()``` を使ってインスタンスがマウントされた後にURLのリストを所得し、データを処理をします。

```myVueDropzone.manuallyAddFile(file, fileUrl, callback)``` は ```myVueDropzone``` から生えているメソッドです。これにファイル情報とパスを渡すことによってコードからファイルをdropzoneに渡してデータを表示できます。file は ```vue2-dropzone``` が期待するオブジェクトの形を渡してあげます。

```myVueDropzone.manuallyAddFile(file, fileUrl, callback)``` 以外のメソッドはこちらで確認できます。
https://rowanwins.github.io/vue-dropzone/docs/dist/#/methods

ここまできたら最後は URL とファイルサイズの JSON を返す APIを　作るだけです。
main.goに一行追加しましょう。

```go:main.go 
// 省略 ...

func main() {
  // 省略 ...

  // GETを追加。!!
  r.GET("/images", handler.List)
  r.POST("images", handler.Upload)
  r.DELETE("/images/:uuid", handler.Delete)
  r.Run(":8888")
}

```

そして ```handelr.handler.go``` にコードを追加します。

```go:handler/handler.go 

// 省略　...

// File has file's info.
type File struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func dirwalk(dir string) (files []File, err error) {
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		path = strings.Replace(path, "images/", "http://localhost:8888/", 1)
		size := info.Size()
		f := File{
			Path: path,
			Size: size,
		}
		files = append(files, f)
		return nil
	})
	if err != nil {
		return
	}
	files = files[1:]
	return
}

// List return url & size list
func List(c *gin.Context) {
	files, err := dirwalk("./images")
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, files)
}

// 省略 ...
```

本来ならデータベースでファイル情報を保存しておきたいところですが、複雑になるので今回は Go言語で頑張って URL と Size をファイル情報から所得して返してます。File構造体はURLとファイルサイズを保持します。そして、少し長いですが、dirwalk関数は指定したディレクトリの中のファイルの情報群 ```[]File``` を返します。これをJSONとして返却します。

## 動作確認

アップロード、削除、ページ更新してもアップロードした画像が確認できるのを確認しましょう。

![m3.gif](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2018/1544313600/c750560b-8f59-56e6-16fb-9bdac681087c.gif)

動いてます！！！

## まとめ

```vue2-dropzone``` を使ってファイルアップロードがスマートに作成できました。他にもいろんな設定ができるので、ドキュメントをご覧ください。
headerの付与や、アップロード最大数等も設定可能です。
https://rowanwins.github.io/vue-dropzone/docs/dist/#/installation

今回は```server/images```にアップロードされた画像を置きましたが、実戦では APIサーバーの開発に データベース、AWS S3 などを使えば更に開発しやすくなるでしょう。

