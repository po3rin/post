---
title: Go製ダブル配列パッケージと最長一致法を使った形態素解析の実装
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/longest-match-cover.jpeg
date: 2020/07/24
id: longest-match
description: 形態素解析本の4章をGoでなぞってみたので解説と実装を共有します。
tags:
    - Go
    - NLP
---

## Overview

NLP初心者ですが、形態素解析本の4章をGoでなぞってみたので解説と実装を共有します。

[形態素解析の理論と実装 (実践・自然言語処理シリーズ) ](https://www.amazon.co.jp/%E5%BD%A2%E6%85%8B%E7%B4%A0%E8%A7%A3%E6%9E%90%E3%81%AE%E7%90%86%E8%AB%96%E3%81%A8%E5%AE%9F%E8%A3%85-%E5%AE%9F%E8%B7%B5%E3%83%BB%E8%87%AA%E7%84%B6%E8%A8%80%E8%AA%9E%E5%87%A6%E7%90%86%E3%82%B7%E3%83%AA%E3%83%BC%E3%82%BA-%E5%B7%A5%E8%97%A4-%E6%8B%93/dp/4764905779/ref=asc_df_4764905779/?tag=jpgo-22&linkCode=df0&hvadid=295686767484&hvpos=&hvnetw=g&hvrand=12780823995254650343&hvpone=&hvptwo=&hvqmt=&hvdev=c&hvdvcmdl=&hvlocint=&hvlocphy=1009303&hvtargid=pla-530658389412&psc=1&language=ja_JP&th=1&psc=1)

## 超シンプルな形態素解析実装の流れ

1: 辞書からダブル配列の構築
2: ダブル配列に対して共通接頭辞検索を実装する
3: 共通接頭辞検索を使った最長一致法の実装

1と2はGo製のダブル配列パッケージである```ikawaha/dartsclose```が実装しているので今回はこれを使いましょう。

[![img1](https://github-link-card.s3.ap-northeast-1.amazonaws.com/ikawaha/dartsclone.png)](https://github.com/ikawaha/dartsclone)

## Go製ダブル配列パッケージ

ダブル配列の解説は下記のブログが最もわかりやすいと思います。

[情報系修士にもわかるダブル配列](https://takeda25.hatenablog.jp/entry/20120219/1329634865)

Goにもダブル配列のパッケージがあるのでこちらを使います。エラーハンドリングは省略していますが、下記のコードでダブル配列を構築でき、ファイルに保存もできます。

```go
package main

import (
	"os"

	"github.com/ikawaha/dartsclone"
)

func main() {
	keys := []string{
		"データ指向アプリケーションデザイン",
		"データ指向アプリケーション",
		"データ指向",
		"データ",
		"指向",
		"指",
		"アプリケーションデザイン",
		"アプリ",
		"デザイン",
		"本",
		"読む",
		"の",
		"は",
		"を",
	}

	// Build
	builder := dartsclone.NewBuilder(nil)
	_ = builder.Build(keys, nil)

	// Save
	f, _ := os.Create("my-double-array-file")
	defer f.Close()
	builder.WriteTo(f)

	// ...
}
```

## 共通接頭辞検索をやってみる

ダブル配列が構築できたので、次に共通接頭辞検索をやってみましょう。

ちなみに、ダブル配列を利用する際にはメモリに乗っかるので、省メモリ&高速に扱う為にmmapを利用したいところです。ちなみにこのパッケージはすでにmmap対応している為、今回はその機能を使います。ドキュメント通り、mmapを利用するにはGoのビルドタグで指定できます。

```env
export GOFLAGS="-tags=mmap"
```

共通接頭辞検索は```trie.CommonPrefixSearch```メソッドが行います。実際に試してみましょう。

```"データ指向アプリケーションデザイン"```というkeyで共通接頭辞検索をしてみます。

```go
// ...

func main() {
	// ...

	trie, _ := dartsclone.OpenMmaped("my-double-array-file")
	defer trie.Close()

	ret, _ := trie.CommonPrefixSearch("データ指向アプリケーションデザイン", 0)
	for i := 0; i < len(ret); i++ {
		fmt.Printf("id=%d, common prefix=%s\n", ret[i][0], "データ指向アプリケーションデザイン"[0:ret[i][1]])
	}

	// ...
}
```

結果は下記のようになります。ちゃんと共通接頭辞検索ができています。

```bash
go run main.go
id=6, common prefix=データ
id=7, common prefix=データ指向
id=8, common prefix=データ指向アプリケーション
id=9, common prefix=データ指向アプリケーションデザイン
```

## 最長一致法

最長一致方は貪欲的アルゴリズムの一種で、文頭から一文字ずつ共通接頭辞検索を行なっていき、その中で最も長い単語を貪欲に選んでtokenとします。最も長い単語が見つかったらその終わりをoffsetとして再度検索を行なっていきます。

```go
// ...

func main() {
	// ...

	var offset int
	tokens := make([]string, 0)
	var unknown string
	s := "ponはデータ指向アプリケーションデザインの本を読む"

	for {
		ret, err := trie.CommonPrefixSearch(s, offset)
		if err != nil {
			panic(err)
		}

		// 見つからなかったらoffset更新して次へ
		if len(ret) == 0 {
			s := string([]rune(s)[offset])
			unknown += s
			offset += len(s)
			continue
		}

		// 辞書に無い未知語を1単語として処理
		if unknown != "" {
			tokens = append(tokens, unknown)
			unknown = ""
		}

		// 最も長い文字列を探す
		var maxIndex int
		var max int
		for i, r := range ret {
			if max < r[1] {
				max = r[1]
				maxIndex = i
			}
		}

		// token追加
		token := s[offset:ret[maxIndex][1]]
		tokens = append(tokens, token)

		// offset更新
		id := ret[maxIndex][0]
		k := keys[id]
		offset += len(k)

		// 脱出条件
		if len(s) <= offset {
			break
		}
	}

	fmt.Printf("%#v", tokens)
}
```

出力はこちらです。こんなシンプルな最長一致法でもそれっぽく形態素解析ができています。

```json
[]string{
	"pon",
	"は",
	"データ指向アプリケーションデザイン",
	"の",
	"本",
	"を",
	"読む",
}
```

形態素解析の本によると、単純なアルゴリズムにも関わらず90%以上の分割精度が得られるそうです。また、辞書引き回数が少ないのも特徴のようです。

## まとめ

今回は最長一致法を使った辞書ベースの形態素解析の作り方を紹介しました。今後はGoでラティス&ビタビアルゴリズムをやってみようと思います。もし時間があればダブル配列も実装したい。。
