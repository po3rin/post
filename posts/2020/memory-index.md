---
title: Goで実装するインメモリ静的転置インデックス
cover: img/memory-index.jpeg
date: 2020/12/08
id: memory-index
description: 「情報検索」の第4章を読みながらインメモリ転置インデックスをGoで実装してきます。
tags:
    - Go
    - Search Engine
draft: true
---

## Overview

こんにちは [pon](https://twitter.com/po3rin) です。これは[「情報検索：検索エンジンの実装と評価」（Buttcher本） Advent Calendar 2020](https://adventar.org/calendars/4968) の記事です。


仕事では主に検索基盤を担当しているので、検索エンジンの内部を知りたくなるのは仕方のないことです。そこでいわゆるButtcher本を読み始めました。

![buttcher](../../img/buttcher.jpeg)

そこで学んだインメモリインデックスは検索に使われる転置インデックスを理解するのにはシンプルである為、情報検索データ構造に関してあまり詳しくない僕のような人間が情報検索を理解する足がかりになるでしょう。

今回はButtcher本で解説されているインメモリインデックスをGoで実装しながら理解していきます。

## 転置インデックスとは

転置インデックスに関しては同じアドカレである「情報検索：検索エンジンの実装と評価」（Buttcher本） Advent Calendar 2020 の2日目の記事にて解説されています。この記事では今回実装する ***スキーマ非依存型インデックス*** についての解説もあるので、転置インデックスの節だけでも目を通して置くと良いでしょう。

[「情報検索：検索エンジンの実装と評価」2章 基本技術 - 人間だったら考えて](https://www.szdrblog.info/entry/2020/12/02/001416#%E8%BB%A2%E7%BD%AE%E3%82%A4%E3%83%B3%E3%83%87%E3%83%83%E3%82%AF%E3%82%B9)

## 転置インデックス実装方針

今回はButtcher本で解説されている手法からなるべくシンプルなデータ構造で転置インデックスを実装していきます。言語は自分が一番慣れているGoで実装していきます。転置インデックスには様々な種類があり、今回は ***スキーマ非依存型のインメモリな静的転置インデックス*** を実装していきます。

### 静的転置インデックス

静的とは途中で内容が変わらないことを意味します。シンプル故に、初めて転置インデックスを勉強する人にとっては良い題材です。

### インメモリインデックス

今回はテキストコレクションが全てメモリに収まるとします。検索対象のテキストコレクションがメモリに収まる範囲の場合、インメモリインデックスを実装することができるので、今回はインメモリインデックスを実装していきます。

TODO: インメモリインデックス

TODO: 逆にテキストコレクションがメモリに収まらない場合は

### スキーマ非依存型インデックス

また、今回はもっともシンプルな ***スキーマ非依存型インデックス*** を実装していきます。これは第2章で出てくるようにドキュメント志向の最適化をせずに、テキストコレクションでの出現位置だけを保持します。

再掲ですが下記の記事でスキーマ非依存型インデックスが解説されています。
[「情報検索：検索エンジンの実装と評価」2章 基本技術 - 人間だったら考えて](https://www.szdrblog.info/entry/2020/12/02/001416#%E8%BB%A2%E7%BD%AE%E3%82%A4%E3%83%B3%E3%83%87%E3%83%83%E3%82%AF%E3%82%B9)


## インデックスを構成するコンポーネント

インデックスを構築するための基本的なコンポーネントは辞書とポスティングリストの二つです。今回はGoで実装していくにあたり辞書とポスティングリストのデータ構造を検討します。

各データ構造は順に解説していきます。全体像はこちらになります。

![index data structure](../../img/inmem-index-archi.png)

### 辞書

辞書は検索時にtermの検索に利用されるデータ構造です。本では ***ソート配列辞書*** と ***ハッシュ配列辞書*** の説明がありますが、ハッシュ配列辞書の方がソート配列辞書よりも探索が高速なので(ハッシュ配列辞書はバイナリサーチなどの調査が不要な為)今回はハッシュ配列辞書を使って実装します。

ハッシュ配列辞書はハッシュテーブルを用いて辞書を構築するデータ構造です。ハッシュが衝突した場合は衝突チェインで繋いでいきます。下図はハッシュ配列辞書を表したものです。

![index data structure](../../img/dict-hash.png)

TODO: 後方挿入

### ポスティングリスト

ポスティングリストはコレクション中の位置情報を保持し、この情報は辞書を介してアクセスします。インデックスの構築中は当然ポスティングリストのデータ構造は拡張可能になっていなければいけません。

TODO: 配列でもつ場合を考えると

例えばポスティングリストを ***linked list*** として実装すると簡単にポスティングリストは拡張可能ですが、ポインタ分のスペースが余分に必要になるというデメリットがあります。下図はlinked listを使った時のポスティングリストを表しています。

![linked list を使った辞書の実装](../../img/linkeddict.png)

そこでこのデメリットを解消しつつ、CPUキャッシュヒット率もあげられるという ***(展開リンクリスト)unrolled linked list*** というデータ構造で実装します。下図は展開リンクリストを使ったポスティングリストを表したものです。

![index data structure](../../img/postings-unrolled.png)

TODO: unrolled linked list の説明

## Goによるインメモリインデックス実装

それでは前節で検討したインデックスを実装していきます。Goのバージョンは執筆時点で最新の1.15.6を使います。

### テキストコレクション

今回はスキーマ非依存型インデックスを実装する為、テキスト全体を一つのテキストとみなして処理していきます。

```data```ディレクトリにいつくかの英語のテキストファイルを置き、そこからテキストコレクションを作成します。デバッグ用のシンプルなテキストファイルを用意しています。

```bash
# hello1.txt
Hello inverted index.

# hello2.txt
Hello search engines.

# hello3.txt
Many search engines incorporate an inverted index.
```

テキストコレクションはtermとその出現位置のペアの集合をみなせるので、まずは(term, position)のペアのスライス```terms```を作成します、

```go
package main

type term struct {
	term string
	pos  int
}

type terms []term

var re = regexp.MustCompile("[^a-zA-Z 0-9]+")

func clean(document string) string {
	return re.ReplaceAllString(strings.ToLower(document), "")
}

func main() {
	datadir := "data"

	files, err := ioutil.ReadDir(datadir)
	if err != nil {
		log.Fatal(err)
	}

	terms := make(terms, 0)
	var pos int
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		b, err := ioutil.ReadFile(filepath.Join(datadir, file.Name()))
		if err != nil {
			log.Fatal(err)
		}
		tokens := strings.Split(string(b), " ") //　tokenize
		for _, t := range tokens {
			pos++
			t = clean(t) // テキストの簡単なクリーニング
			terms = append(terms, term{
				term: t,
				pos:  pos,
			})
		}
    }
    
    fmt.Println(terms)
    
    // ...
}
```

今回のtokenizeはSpaceでのスプリットで行なっています。日本語をサポートする場合は、形態素解析ライブラリを移植すれば良いでしょう。

ファイルの中身を見ていき、正規表現を使う```clean```関数で不要な記号などを取り除いてから```terms```に格納していきます。これを実行すると下記の結果が得られます。

```
# go run main.go
[{hello 1} {inverted 2} {index 3} {hello 4} {search 5} {engines 6} {many 7} {search 8} {engines 9} {incorporate 10} {an 11} {inverted 12} {index 13}]
```

テキストコレクションをtermとpositionのペアの集合に変換できました。ここまできたら次はデータ構造を用いてインデックスを構築していきます。

###　データ構造の実装

前節で説明したように辞書にはハッシュテーブルを、ポスティングリストには展開リンクリストを利用します。
辞書のハッシュテーブルに関してはGoの```map```がハッシュテーブルの実装なのでこいつを使いましょう。```postings```型はポスティングリストの展開リンクリストを表現する型です。

```go
type dict map[string]*postings
```

続いて```postings```型を実装していきます。```postings.next```が次のgroupへのポインタを格納し、```ps```フィールドにはテキストコレクション内での位置情報を保持します。

```go
const limit = 16

// schema-independent index
type postings struct {
	ps   []int // テキストコレクションでの位置
	next *postings //　次のgroupへのポインタ
}

func (p *postings) put(pos int) {
    // nextがnilじゃないなら今のgroupは充足しているので次のgroupへ
	if p.next != nil {
		p.next.put(pos)
		return
	}

    // groupが一杯になったら新しくgroupを作成しそこに追加しにいく
	if cap(p.ps) <= len(p.ps) {
		p.next = &postings{
			ps: make([]int, 0, min(limit, 8)),
		}
		p.next.put(pos)
    }

	p.ps = append(p.ps, pos)
}

func min(i, j int) int {
	if i > j {
		return i
	}
	return j
}
```

これで基本的なデータ構造の実装はできました。実際にこれらを使ってインデックスを構築していきます。debug用に```*postings```に```get```メソッドを追加した後に ````main```関数の続きを書いていきます。

### インデックス構築

```go
func (p *postings) get() []int {
	list := make([]int, 0)
	return p.getTraverse(list)
}

func (p *postings) getTraverse(list []int) []int {
	list = append(list, p.ps...)
	if p.next != nil {
		return p.next.getTraverse(list)
	}
	return list
}

func main() {
	// 続き ...

	dict := make(dict, 0)
	for _, t := range terms {
		var e *postings
		e, ok := dict[t.term]
		if !ok {
            // 辞書にないなら新しくエントリ追加
			e = &postings{
				ps: make([]int, 0, 4),
			}
			dict[t.term] = e
		}
		e.put(t.pos)
	}

	for t, e := range dict {
		fmt.Println("--------------")
		fmt.Printf("term: %v\n", t)
		ps := e.get()
		fmt.Printf("pos: %v\n", ps)
	}
}

```

これを実行してみましょう。狙った通りのスキーマ非依存型の静的インデックスが作成できました。

```bash
$ go run main.go
--------------
term: engines
pos: [6 9]
--------------
term: many
pos: [7]
--------------
term: incorporate
pos: [10]
--------------
term: an
pos: [11]
--------------
term: hello
pos: [1 4]
--------------
term: inverted
pos: [2 12]
--------------
term: index
pos: [3 13]
--------------
term: search
pos: [5 8]
```

## まとめ

本記事では転置インデックスの実装を通してそこで利用できるデータ構造の一部をご紹介しました。Buttcher本の4章ではこの他にもたくさんのデータ構造を紹介しているので、興味ある方はぜひ読んでみてください。
