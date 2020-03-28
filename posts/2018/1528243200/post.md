# Go言語で理解するJWT認証 実装ハンズオン

![go-jwt.png](../../img/qiita-740445d21487dfcb5d9f-1.png)


僕がWEBアプリケーションを作る時、認証周りをどう実装するか毎回悩むのですが、JWT認証が非常に実装しやすかったのでハンズオン形式で紹介していきます。

##Go言語
<img src="../../img/qiita-740445d21487dfcb5d9f-2.png" width=20%>

サーバーサイドは Go言語でいきます。Googleが開発したプログラミング言語です。フロントエンド畑で育った僕が、なぜサーバーサイドにGo言語を選ぶかというと、主に下記の理由

* 構文 & 型システムがシンプル。動的言語しかやってこなかった僕でもすんなり受け入れてくれた。
* 並列処理が書きやすい。JavaScriptで苦戦した非同期処理が難なく書ける。
* 標準パッケージのサポートが強力。コードのフォーマットも自動でやってくれる。

とにかく開発者に無駄なことをさせないという気迫がすごい。ゆえに書いてて、読んでて気持ち良い。フロントエンドでずっとやってきた僕のような人間が速習でサーバサイドを学びたいと思ったらGo言語を推します。シンプルゆえにJWTがどのように動いているかも分かりやすいので、今回はGo言語を使ったハンズオンにしました。

##JWT
<img src="https://qiita-image-store.s3.amazonaws.com/0/186028/b3fe9aef-86d7-55a6-0b2b-f45abc8dcb80.jpeg" width=40%>

JWT(Json Web Token)と書いてジョットと読むらしい。このトークンを使ってクライアント＆サーバー間で認証できる。特徴としては、

* 発行者が鍵を使ってJSONを署名し、トークンとして使う。
* JSONなので任意の情報を含めることができる(ユーザー名など)。
* 発行者は鍵を使ってトークンの検証を行う為、改竄を検知できる。

上記の特徴から、僕が特にメリットに感じた点は「ステートレス」という点。サーバ側にセッションストアを持たなくても検証ができる。

JWTは以下のフォーマットに従った文字列です。

```
{base64エンコードしたheader}.{base64エンコードしたclaims}.{署名}
```

実際にこちらのサイトでJWTのエンコード、デコードが試せる。
>https://jwt.io/

実際の使い方はGo言語で実装しながら確認していきましょう。

##認証なしAPIをGo言語で実装

まずは認証なしのAPIサーバを作ります。ポート番号8080で立てます。
APIには軽量なウェブツールキット ```gorilla/mux``` を使って実装します。これはルーティングとかセッションとかを ```net/http``` よりは柔軟に使える感じのもの。

そしてmain.goを作り下記を実装

```go:main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type post struct {
	Title string `json:"title"`
	Tag   string `json:"tag"`
	URL   string `json:"url"`
}

func main() {
    r := mux.NewRouter()
    // localhost:8080/publicでpublicハンドラーを実行
	r.Handle("/public", public)
    
    //サーバー起動
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("ListenAndServe:", nil)
	}
}

var public = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	post := &post{
		Title: "VueCLIからVue.js入門①【VueCLIで出てくるファイルを概要図で理解】",
		Tag:   "Vue.js",
		URL:   "https://qiita.com/po3rin/items/3968f825f3c86f9c4e21",
	}
	json.NewEncoder(w).Encode(post)
})

```

muxで簡単にルーティングの設定ができています。```json.NewEncoder(w).Encode(user)```は```http.ResponseWriter``` にjsonにエンコードしたuserを書き込んでいます。これでJSONを返してます。

この ```main.go``` の1ファイルでJSONを返すAPIサーバの完成です。実際に叩いてみると記事のJSONが返ってきます。

```bash
$ curl localhost:8080/public
{"title":"VueCLIからVue.js入門①【VueCLIで出てくるファイルを概要図で理解】","tag":"Vue.js","url":"https://qiita.com/po3rin/items/3968f825f3c86f9c4e21"}
```

##JWT認証をつける

ここからが本番。JWT認証つけていきます。

###Go言語でJWT認証を追加する

```main.go``` と同じディレクトリに ```auth/auth.go``` を作成し自作のauthパッケージを作ります。

```go:auth.go
package auth

import (
	"net/http"
	"os"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	jwt "github.com/dgrijalva/jwt-go"
)

// GetTokenHandler get token
var GetTokenHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

    // headerのセット
	token := jwt.New(jwt.SigningMethodHS256)

    // claimsのセット
	claims := token.Claims.(jwt.MapClaims)
	claims["admin"] = true
	claims["sub"] = "54546557354"
	claims["name"] = "taro"
	claims["iat"] = time.Now()
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

    // 電子署名
	tokenString, _ := token.SignedString([]byte(os.Getenv("SIGNINGKEY")))

    // JWTを返却
	w.Write([]byte(tokenString))
})

// JwtMiddleware check token
var JwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SIGNINGKEY")), nil
	},
	SigningMethod: jwt.SigningMethodHS256,
})
```

そして環境変数に```"SIGNINGKEY"```をセットします。これはJWTに電子署名する為のシークレットキーです。絶対に外部に漏れないようにしなければいけません。僕はdirenvを使ってるので、そこでセットします。もちろん`.bashrc`とかでも大丈夫です。めんどくさい人は直接指定しちゃって下さい。

```
export SIGNINGKEY=<こちらで自由に設定するシークレットキー>
```

main.goの`GetTokenHandler`でclaimsや署名アルゴリズム(今回はHS256)を含めた上で署名したJWTが発行できます。JWTは以下のフォーマットに従った文字列ということを思い出してください。

```
{base64エンコードしたheader}.{base64エンコードしたclaims}.{電子署名}
```
これらの情報を含んだJWTを作成しています。header、claims、署名をそれぞれ簡単に仕様を紹介します。

###Header

ヘッダ情報には、署名アルゴリズムの種類やメタ情報が組み込まれます。今回は`token := jwt.New(jwt.SigningMethodHS256)`でJWTとHS256を指定しています。

|項目|説明|
|---|---|
|alg|署名アルゴリズム。(HS256など)|
|typ|トークのタイプ。(JWTが推奨)|

###Claims

claimsにはユーザー属性情報を埋め込んでいます。claimsのには標準クレーム群が定義されています。全部は紹介しませんが下記のようなものが存在します。こちらからTokenに埋め込みたい情報を追加します。

| クレーム名               | 説明
|:------------------------|:----|
| `sub`                   | ユーザーの一意識別子 |
| `name`                  | フルネーム |
| `profile`               | プロフィールページの URL |
| `picture`               | プロフィール画像の URL |
| `website`               | ウェブサイトもしくはブログの URL |
| `email`                 | メールアドレス |
| `gender`                | 性別 |
| `birthdate`             | 誕生日 |
| `zoneinfo`              | 地域情報 |
| `phone_number`          | 電話番号 |
| `address`               | 住所 |
| `updated_at`            | 情報最終更新時刻 |

今回はハードコーディングしてますが、実際にはデータベースからもってきたデータをJWTに仕込んだりします。

###電子署名
トークンの検証に利用します。これがある為にトークンの改竄ができなくなっています。今回は下記の部分で、シークレットキーを使い署名しています。
`tokenString, _ := token.SignedString([]byte(os.Getenv("SIGNINGKEY")))`


##APIをJWT検証Middlewareで守る
上のコードで自作のauthパッケージができました。これをmain.jsで読み込んで使います。main.goの中でfromでauthパッケージを読み込んでください

そして main.go のトップレベルにprivateハンドラーを追加します。

```go
var private = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	post := &post{
		Title: "VGolangとGoogle Cloud Vision APIで画像から文字認識するCLIを速攻でつくる",
		Tag:   "Go",
		URL:   "https://qiita.com/po3rin/items/bf439424e38757c1e69b",
	}
	json.NewEncoder(w).Encode(post)
})
```

そして main.goの```r.Handle("/public", public)```の下に二行追加します。

```go
r.Handle("/private", auth.JwtMiddleware.Handler(private))
r.Handle("/auth", auth.GetTokenHandler)
```

ここでは`/private`のみJwtMiddlewareでラップしています。つまり`/private`を叩くにはJWTによる認証が必要になります。


これでサーバ側の実装は終わりました。実際にAPIを叩きましょう。

```bash
$ curl localhost:8080/public
{"title":"VGolangとGoogle Cloud Vision APIで画像から文字認識するCLIを速攻でつくる","tag":"Go","url":"https://qiita.com/po3rin/items/bf439424e38757c1e69b"}

$ curl localhost:8080/private
Required authorization token not found
```

```/private``` だけ認証トークンが必要だと言われてしまいます。ではtokenを所得して```/private```を叩いてみましょう。まずは```/auth``` を叩いてJWTを手に入れましょう。

```bash
$ curl localhost:8080/auth
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhZG1pbiI6dHJ1ZSwiZXhwIjoxNTI4Mjk5NTE3LCJpYXQiOiIyMDE4LTA2LTA2VDAwOjM4OjM3LjQxMDM1OTc4MyswOTowMCIsIm5hbWUiOiJ0YXJvIiwic3ViIjoiNTQ1NDY1NTczNTQifQ.48QRN_eXXzwXv1daqUmOIusnHAZH21tabmiJ5wFrAWE
```

JWTが返ってきました。エンコードされていますが、先ほど紹介した{header}.{claims}.{署名}の形になっています。
実際にこちらのサイトでJWTをデコードしてみましょう！
https://jwt.io/


<img width="1134" alt="スクリーンショット 2018-06-06 00.39.03.png" src="../../img/qiita-740445d21487dfcb5d9f-3.png">

先ほどheaderやclaimsに込めた情報を確認できます。
このトークンを使って```/private```が叩けます。AuthorizationヘッダにBearerスキームを設定して送信してサーバ側で検証します。

```
Authorization: Bearer {JWT token}
```

実際にはこんな感じ

```bash
$ curl localhost:8080/private -H "Authorization:Bearer <先ほど返ってきたJWT>"
{"title":"VGolangとGoogle Cloud Vision APIで画像から文字認識するCLIを速攻でつくる","tag":"Go","url":"https://qiita.com/po3rin/items/bf439424e38757c1e69b"}
```
先ほどリクエストできなかった`/private`からデータが返ってきました。これでJWTによるセキュアな API ができました。実際にはJWT発行の際にはBasic認証を噛ませたりもします。

以上です。

