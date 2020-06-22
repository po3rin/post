---
title: Goのインターフェース抽象度を美しく保つ為の思考
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/go-interface-co.jpeg
date: 2020/06/22
id: go-interface
description: Goで抽象化を適切に、そして美しく保つ為の自分の考えやTipsを紹介します。
tags:
    - Go
---

## Overview

とある場面でGoの```interface```が持つ振る舞いの抽象度について議論があり、今回はそれをアウトプットしておきます。Go初心者でinterfaceが上手く設計できない人向けです。

## 目次

今回の目次です！下記について自分の考えをお話しします！

* 振る舞いの抽象化の度合いを意識する
* 抽象度をどこまであげるか
* 引数や返り値から発生する「抽象化の漏れ」
* 抽象度をあげる為の統合
* Getter/Setterと抽象度

それではいってみましょう！

## 振る舞いの抽象化の度合いを意識する

振る舞いをinterfaceとして定義していくのがGoの抽象化ですが、そもそも **抽象化は度合いのある概念です** 。この度合いを意識しないと適切なinterfaceの設計は困難です。例えばMySQLにUserを登録する振る舞いがある時、リポジトリパターンによる抽象化を目的にこのようなinterfaceを定義するかもしれません。

```go
// MySQLへの具体的な登録処理を抽象化
type Repository interface {
	RegisterUserToMySQL(user User) error
}
```

上記はMySQLにUserを登録するという振る舞いをinterfaceとして定義していますが、MySQLという具体的な技術に依存しているため抽象度はかなり低いです。MySQLを抽象化するならこうでしょうか？

```go
// DBへの登録を抽象化
type Repository interface {
	RegisterUserToDB(user User) error
}
```

MySQLという具象をデータベースという形で抽象化した為、先ほどより抽象度が上がっています。MySQLだけでなく、PostgreSQLやDynamoDBを使った実装でもこのinterfaceを満たすことができます。しかしまだ抽象度を上げれます。これだとDBというミドルウェアに依存しているからです。更に抽象度をあげるならこうでしょうか？

```go
// ユーザー登録を抽象化
type Repository interface {
	RegisterUser(user User) error
}
```

これでデータベースという具象を更に抽象化し、データベースだけでなく、直接ファイルシステムへの保存や、他のマイクロサービスへの委託も実装として可能になりました。 **言葉にして図にすると抽象は具象をネストしている概念であることが伝わると思います** 。

![抽象度の概念図](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/go-interface1.png)

これを意識できると、抽象度が揃っていないことに気付けるようになります。下記のinterfaceは抽象度が揃っていない例です。

```go
type Repository interface {
	RegisterUser(user User) error
	ResisterGroupToDB(group Group) error
}
```

```RegisterUser```は何にユーザーを保存するかまで抽象化していますが、```ResisterGroupToDB```はDBを使うことを要求しているので抽象度が揃っていません。 **抽象度が違うinterfaceは結局一番抽象度の低い振る舞いと同じ抽象度になります** 。このような抽象度が揃っていないinterfaceがある場合はなるべく抽象度を揃えてあげると良いでしょう。下記はDBへの処理を抽象化している例です。
s
```go
type DBRepository interface {
	RegisterUser(user User) error
	ResisterGroup(group Group) error
}
```

DBという名前に変えてあげることでDBへの保存処理という形で抽象度を合わせています。

## 抽象度をどこまであげるか

**抽象度を高く保つことが常に正義であるという勘違い** をしないことも重要です。抽象化にはデメリットも存在します。下記のコードを見てみましょう。

```go
type Repository interface {
	RegisterUser(user User) error
}
```

将来的にユーザー情報を保持するミドルウェアを変更する可能性を見据えて、ユーザー登録を抽象化したコードです。しかし、開発時にこのコードを読むと「登録処理はDBに直接投げているのか、全文検索エンジンに投げるのか、はたまた別のマイクロサービスに投げているのか」が分かりません。開発者は結局このinterfaceを実装しているコードを逐一見にいってしまうでしょう。つまり **抽象化によって情報が欠落するのです** 。もう1度、抽象度でネストしている図を見てみましょう。抽象度が上がるにつれ文章の情報が欠落していくのがわかると思います。

![抽象度の概念図](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/go-interface1.png)

抽象化の目的がミドルウェアの変更容易性の向上であり、実は変更するミドルウェアをDBのみに想定しているなら(MySQL -> PostgreSQL など)下記のようにDBの実装を抽象化するだけで十分です。

```go
type DBRepository interface {
	RegisterUser(user User) error
}
```

これでコードを見ただけで内部でDBへの操作を行っていることが分かります。ここまで抽象度を下げれば開発者がDBを意識できるので、関数名を見るだけで多くの情報を得られます。

過度な抽象化は避け、**将来の変更は本当に起こり得るのか、起こるとしたらどの範囲までの変更なのかを考えて抽象化するようにしましょう。** 抽象化の目的を自信を持って答えれないようなら、抽象化をそもそも取りやめるべきです。

一方で駆け出しGopherの多い現場では十分に抽象化しきれず失敗するパターンの方が多い為、今後は抽象化の目的を一旦脇に置き、抽象度を上げれずに失敗するパターンを見ていきます。

## 引数や返り値から発生する「抽象化の漏れ」

interfaceで振る舞いを抽象化したのにも関わらず、**具象が引数や返り値から溢れ出てしまうケース** があることに気づきました。実は少し前まで弊社のGoアプリケーションのinterfaceではこのような問題のあるコードが散見されました。わかりやすいようにメソッドは省略してありますが、下記は本当にあった怖いinterfaceです。

```go
type Store interface {
	// ...
	SimilarItems(p elasticsearch.SearchParams) ([]Item, error)
}
```

類似のアイテムをElasticsearchから取得する操作を抽象化したはずが、独自定義のelasticsearchパッケージの構造体```elasticsearch.SearchParams```が漏れ出しています。これではElasticsearchへの操作を全く抽象化できていません。

元々は、大量の引数を1つの構造体として受けたかっただけなのですが、そのメソッドをそのままinterfaceとして定義してしまったようです。個人的に引数や返り値に抽象化したかったはずの具象が残ってしまうことを **抽象化の漏れ** と呼んでいます。自分は上記のinterfaceを下記のように修正しました。

```go
type Store interface {
	// ...
	SimilarItems(item Item) ([]Item, error)
}
```

呼び出し側が作った```elasticsearch.SearchParams```から```Item```を生成していた処理を```SimilarItems```メソッドの外へ切り出しました。このようにinterfaceの抽象度を適切に保つためのリファクタリングでは、処理の切り出しや、処理の統合が行われます。今回の例では処理の切り出しにより抽象度を保っています。

## 抽象度をあげる為の統合

抽象度をあげる為の統合のパターンにおいては下記のようなinterfaceが題材として考えられます。

```go
type Store interface {
	CreateQuery(words []string) Query
	SearchItems(q Query) ([]Item, error)
}
```

```CreateQuery```は単語から検索クエリを生成し、```SearchItems```はQueryを使ってアイテムを検索します。これは **クエリという概念を使うミドルウェアを利用することを要求してしまう** ので抽象度は低いです。この例ではクエリの生成を```SearchItems```の内部で呼ぶようにして抽象度をあげることが可能です。

```go
type Store interface {
	SearchItems(words []string) ([]Item, error)
}
```

これで振る舞いを統合することで抽象度を上げました。また、メソッドが１つ減り、interfaceを小さくできたので、interfaceを実装する側が楽にinterfaceを満たせるようになりました。

難しい言葉を使うと **逐次的凝縮、手続き的凝縮で利用されている振る舞いは統合できる可能性があります**。先ほどの例ではinterfaceが逐次的凝縮を要求しています(CreateQueryを呼んでからSearchItemsを呼ぶ必要がある)。逐次的な振る舞いを１つに統合してinterfaceの振る舞いを１つ落とせました。

以下では逐次的凝縮を見つけて振る舞いを統合する例をもう１つ紹介します。

```go
// interface定義側
type Repository interface {
	CreateUser(name string) User //ユーザー生成
	RegisterUser(u User)         //ユーザーを登録
}

// interface利用側。動作が逐次的凝縮になっている。
func XXX(r Repository) {
	// ...
	user := r.CreateUser("pon")
	r.RegisterUser(user)
	// ...
}
```

ユーザー登録をする際にこの２つのメソッドが手続き的凝縮になって利用されているので、振る舞いを１つにまとめることができます。

```go
type Repository interface {
	// 内部で CreateUser と同等の処理を持つことを期待する。これで十分では？
	RegisterUser(name string) error
}
```

interfaceが巨大になったら、凝縮性の観点からも振る舞いの統合を検討してみましょう。凝縮性に関しては１つの大きなテーマなので他の最高の資料にお任せします。

[オブジェクト指向のその前に-凝集度と結合度/Coheision-Coupling](https://speakerdeck.com/sonatard/coheision-coupling)

## Getter/Setterと抽象度

これも統合で解決できる例ですが頻出するので言及します。皆さんもこんなinterfaceを定義したことがあるかもしれません。これはデータ構造をもつモデルを抽象化しようと思った時に発生しがちなinterfaceです。

```go
// とあるURLに対してメッセージを送信する振る舞いを抽象化
type Messenger interface {
	GetURL() string
	SetURL(url string)
	GetMessage() string
	SetMessage(msg string)
	Send() error
}
```

とあるURLに対してメッセージを送信する振る舞いを持つinterfaceです。これだと```URL```や```Message```というデータ構造を持つ具象を要求するので抽象度は低いです。このようにフィールド毎にGetter/Setterを定義すると抽象度が大きく下がる可能性があります。

また抽象度だけでなく別の問題も発生します。それは **interfaceの肥大化** です。interfaceを小さく保つことは、interfaceを実装する側を楽にするのでなるべくミニマムに保ちたいところです。弊社ではGetter/Setterの定義で10個以上のメソッドが定義された巨大interfaceがそびえ立っていたこともあります。

interfaceにフィールド毎のGetter/Setterを定義してしまう問題の解決の指針としては下記が考えられます。
* そもそも使っていないメソッドは削除する(当然)
* Getterをコールしている処理も含めて抽象化する
* Setterは関数の引数で渡せないか？もしくは初期化関数で渡せないか？

上記のinterfaceはこれで十分でしょう。Setterで渡していた```url```と```message```を引数でうけるようにしています。

```go
type Messenger interface {
	Send(url string, message string)
}
```

「実はURLに対してPingするのにGetURLというGetterが必要だったんです。。」ということならそれも含めて抽象化しましょう。下記で十分です。

```go
type Messenger interface {
	Send(url string, message string) error
	Ping(url string) error
}
```

これでもまだ引数でurlを必ず要求するので初期化関数でURLを渡せるようにしとくとinterfaceの定義内ではURLという存在すら抽象化できます。

```go
// 引数からURLを消せたので、もはやURLという具象すら抽象化できている。
type Messenger interface {
	Send(message string) error
	Ping() error
}

// ...

// Messengerを実装する具象を返す。引数にURLを要求する。
func NewMessenger(url string) *messenger
```

これでGetter/Setterを削除してinterfaceに定義するのを振る舞いのみに限定することで抽象度を保ち、interfaceを小さく保てました。当然Getter/Setterが必要なinterfaceもあります。

## まとめ

Goのinterfaceから抽象度について考察し、抽象度を適切に保つ為の思考やTipsを紹介しました。過度な抽象化はせず、適切に美しい抽象化を目指していきましょう！

## 参考文献
[@sonatard](https://twitter.com/sonatard)さんの資料が最高です！

[オブジェクト指向のその前に-凝集度と結合度/Coheision-Coupling](https://speakerdeck.com/sonatard/coheision-coupling)

[Repositoryによる抽象化の理想と現実/Ideal and reality of abstraction by Repository](https://speakerdeck.com/sonatard/ideal-and-reality-of-abstraction-by-repository)

[Wiki:凝集度](https://ja.wikipedia.org/wiki/%E5%87%9D%E9%9B%86%E5%BA%A6)
