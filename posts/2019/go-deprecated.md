---
title: Goのパッケージ開発者がDeprecatedを利用者に伝える & 利用者が検知する方法
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/old.jpeg
date: 2019/05/29
id: go-deprecated
description: 今回の記事は僕のような初心者向けですが、Goのパッケージを開発して公開している方達は特に必見です。
tags:
    - Go
---

## Deprecated(非推奨)を利用者に伝える

全てはGoのwikiに書いてありますが、 Deprecated を利用者に伝える方法はコメントで```Deprecated:```と記述することです。https://github.com/golang/go/wiki/Deprecated

こんな感じです。

```go
// Deprecated: should not be used
func Add(a, b int) int {
	return a + b
}
```

例えば olivere/elastic (GoでElasticsearchを扱うパッケージ) では下記のように書かれています。

```go
// SetMaxRetries sets the maximum number of retries before giving up when
// performing a HTTP request to Elasticsearch.
//
// Deprecated: Replace with a Retry implementation.
func SetMaxRetries(maxRetries int) ClientOptionFunc
```

```Deprecated:``` の後には非推奨の理由や、代わりに何を使えば良いかなどを記載すると良いでしょう。

## Deprecated(非推奨)を検知する

Go の 公式のlintツールでは教えてくれないので、静的解析ツールや他のりんとツールを使うと良いでしょう。例えば https://github.com/dominikh/go-tools の静的解析ツールセットを使うと下記のように利用パッケージの非推奨を検知できます。

```bash
$ staticcheck ./...
file/path:132:3: elastic.SetMaxRetries is deprecated: Replace with a Retry implementation.  (SA1019)
```

## まとめ

使っている全てのパッケージの更新を追うのは辛いので、パッケージの利用者は静的解析ツールで非推奨な機能に気づけるようにしておきましょう。パッケージの開発者の方は非推奨機能を置いておく場合は、利用者が非推奨に気づけるように```Deprecated:```は付けておきましょう。

