---
title: Goを始めて最高にお世話になったGo関連ブックマークを晒します
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/bookmark.jpeg
date: 2019/10/09
id: go-bookmark
description: ブックマークを整理したのですが、せっかくなのでGoを1年間書いてきてお世話になったブックマーク記事を晒します。
tags:
    - Go
---

## Blog & Serial

#### The Go Blog
Goの公式ブログ。深いところまでしっかり書かれているので、調べたいトピックはまずはここで調べたい。
[https://blog.golang.org/](https://blog.golang.org/)

#### Practical Go
GoのcontributorであるDave Cheneyさんのブログです。Goで開発&運用する上でのアドバイスが書かれており、入門記事だけでは得られないノウハウがふんだんにまとめられています。
[https://dave.cheney.net/practical-go](https://dave.cheney.net/practical-go)

#### Goならわかるシステムプログラミング
@shibukawaさんの連載です。Goで低レイヤーを学んでいきます。根底の仕組みがガッツリ学べます。あまりに良すぎて僕は書籍で購入してます。書籍の方は色々加筆されているので皆さんも書籍で書いましょう。
[https://ascii.jp/elem/000/001/235/1235262/](https://ascii.jp/elem/000/001/235/1235262/)

#### A Journey With Go
Goに関する面白いトピックが多いです。Goの内部の話などGo初心者が中級者になるための記事が多い印象です。
[https://medium.com/a-journey-with-go](https://medium.com/a-journey-with-go)

## Tips

#### Go CodeReviewComments 日本語翻訳 #golang
Goを書く上で気をつけるべき&意識すべきポイントが網羅された「Go CodeReviewComments」の日本語訳記事
[https://qiita.com/knsh14/items/8b73b31822c109d4c497](https://qiita.com/knsh14/items/8b73b31822c109d4c497)

#### High Performance Go Workshop
Goアプリケーションのパフォーマンスに関するトピックほぼ全てが網羅された記事。ベンチマーク、プロファイリングはもちろん、コンパイラ最適化、メモリやガベージコレクタについてなどについて書かれている。はっきり言って最高。
[https://dave.cheney.net/high-performance-go-workshop/gopherchina-2019.html](https://dave.cheney.net/high-performance-go-workshop/gopherchina-2019.html)

#### Profiling Go Programs
Goの公式ブログによるプロファイリングツールpprofの使い方です。使い方忘れたらよくここにきます。
[https://blog.golang.org/profiling-go-programs](https://blog.golang.org/profiling-go-programs)

#### SliceTricks
Goの公式WikiのSlice操作の実装パターン集です。いい感じのSlice操作実装を忘れた時にめちゃ見返す。
[https://github.com/golang/go/wiki/SliceTricks](https://github.com/golang/go/wiki/SliceTricks)

#### インタフェースの実装パターン #golang
インターフェースの実装パターンがコード付きで丁寧に解説されています。
この記事を書いている @tenntenn さんの記事はどれもGo初心者にとって知っておいたほうがいい事ばかりなので他の記事も読んでみることをオススメします。
[https://qiita.com/tenntenn/items/eac962a49c56b2b15ee8](https://qiita.com/tenntenn/items/eac962a49c56b2b15ee8)

## Architecture

#### struct に依存しない処理は function に切り出すのか、method に切り出すのか
関数にするかメソッドにするかはGopherたちが悩む部分です。迷ったら読み返します。
[https://www.pospome.work/entry/2017/01/16/233351](https://www.pospome.work/entry/2017/01/16/233351)

#### Golang Receiver vs Function Argument
こちらも処理を関数にするかメソッドにするか迷った時に読み返します。
[https://grisha.org/blog/2016/09/22/golang-receiver-vs-function/](https://grisha.org/blog/2016/09/22/golang-receiver-vs-function/)

#### Functional Option Pattern
構造体の初期化時にオプション引数を与えるためのデザインパターンである「Functional Option Pattern」の日本語解説です。
[https://blog.web-apps.tech/go-functional-option-pattern/](https://blog.web-apps.tech/go-functional-option-pattern/)

## Analysis

#### GoのためのGo
Goで静的解析する際の基本が体系的にまとまっています。
[https://motemen.github.io/go-for-go-book/](https://motemen.github.io/go-for-go-book/)

## Handson

#### Write a Kubernetes-ready service from zero step-by-step
Goで作ったAPIをKubernetesで動かす記事ですが、Kubernetes動かすまでにGoによるAPI開発の基本的なポイントがしっかり抑えられていてGoを触り始めた頃にすごく勉強になったハンズオンです。
[https://blog.gopheracademy.com/advent-2017/kubernetes-ready-service/](https://blog.gopheracademy.com/advent-2017/kubernetes-ready-service/)

## Test

#### Goのtestを理解する in 2018 #go
テストの書き方でここってどうやって書いたっけ？となったら必ず訪れるブログ。ほぼテストに関する全てが書いてある。
[https://budougumi0617.github.io/2018/08/19/go-testing2018/](https://budougumi0617.github.io/2018/08/19/go-testing2018/)

#### Go Fridayこぼれ話：非公開（unexported）な機能を使ったテスト #golang
非公開の機能をテストしたい時のパターンが網羅されている。
[https://tech.mercari.com/entry/2018/08/08/080000](https://tech.mercari.com/entry/2018/08/08/080000)

## Packages

#### Awesome Go : 素晴らしい Go のフレームワーク・ライブラリ・ソフトウェアの数々
こんなパッケージないかな？？って時に開く記事。
[https://qiita.com/hatai/items/f31914f37dc6c53b2bce](https://qiita.com/hatai/items/f31914f37dc6c53b2bce)

## Playground

#### GoDoc Playground
GoDocのPlaygroundです。GoDocを書く際にコメントアウトで記載していきますがそれをリアルタイムで確認できます。
[https://bradleyjkemp.dev/godoc-playground/](https://bradleyjkemp.dev/godoc-playground/)

## Editor

#### vim-go チュートリアル
vim-goのチュートリアルです。GoをVimで操りたくなったらまずここ。
[https://github.com/hnakamur/vim-go-tutorial-ja](https://github.com/hnakamur/vim-go-tutorial-ja)

#### vim-goを使うなら使用したいコマンド集と設定

VimでGoを触るためのノウハウが体系的にまとまっています。
[https://qiita.com/gorilla0513/items/a027885d03af0d6d5863](https://qiita.com/gorilla0513/items/a027885d03af0d6d5863)

## Detects Issue

### Go Report Card

[![Go Report Card](https://goreportcard.com/badge/github.com/po3rin/gonnp)](https://goreportcard.com/report/github.com/po3rin/gonnp)

GoのリポジトリのURLを入れるだけでgolintやgofmtでコードをチェックしたりmisspellを見つけたりしてくれます。最終的にREADME.mdに貼れる上記のようなバッジがすぐに作れます。

[https://goreportcard.com/](https://goreportcard.com/)

### GolangCI

[![GolangCI](https://golangci.com/badges/github.com/po3rin/gonnp.svg)](https://golangci.com)

~~こちらもGoのリポジトリに対して様々な問題を指摘してくれます。structcheck、errcheck、gosimpleなどのツールで細かい問題点も指摘してくれます。こちらもバッジが作れます。https://golangci.com/~~

2020年4月にサービスが終了するようです。。
詳しくは作者のブログをご覧ください
[https://medium.com/golangci/golangci-com-is-closing-d1fc1bd30e0e](https://medium.com/golangci/golangci-com-is-closing-d1fc1bd30e0e)
