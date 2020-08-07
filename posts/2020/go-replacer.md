---
title: Goの文字列置換とその内部実装
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/replace-go-cover.jpeg
date: 2020/07/14
id: go-replacer
description: Goの文字列置換で利用されているアルゴリズム、データ構造を探りました。
tags:
    - Go
---

## Overview

こんにちはponです。今回は、Go の文字列置換周りのコードを読んだので共有します。Goには正規表現を除くと2つの文字列置換が用意されています。```func (*Replacer) Replacer``` と ```func Replace``` です。今回はその2つの文字列置換のコードリーディングをしたのでその知見を共有します。

## Replace と Replacer の基本

[Replace](https://golang.org/pkg/strings/#Replace)は strings パッケージに定義されている関数です。文字列の置換をしてくれます。

```go
func main() {
	s := "hello world"
	ns := strings.Replace(s, "hello", "bye", -1) // or ReplaceAll
	fmt.Println(ns)                              // bye world
}
```

一方で strings パッケージには [Replacer](https://golang.org/pkg/strings/#Replacer) という構造体が定義されているのをご存知でしょうか？```Replacer```を使うと複数の置換パターンをまとめて実行できます。


```go
func main() {
	re := strings.NewReplacer("Golang", "Go", "Clang", "C")
	s := "writes Golang and Clang"
	res := re.Replace(s)
	fmt.Println(res) // writes Go and C
}
```

ちなみに```Replacer```は [ドキュメント](https://pkg.go.dev/strings?tab=doc#Replacer)に記載があるようにgoroutine safe です。

> It is safe for concurrent use by multiple goroutines.

## Replace Internal

**実は strings.Replace は strings.Index があれば構築できます**。 ```strings.Index```は任意の文字列の中にある文字列が含まれているかをチェックして、もしあればその始まりのindexの値を返す関数です。```strings.Index```の内部実装については過去に記事にしたので[こちら](https://po3rin.com/blog/go-rabin-karp)を参照ください。



実際に```strings.Replace```の中で```strigs.Index```が利用されているのを確認しましょう。少しコードは省略しています。

```go
func Replace(s, old, new string, n int) string {
	//...

	// 置換するべき箇所の数を取得
	if m := Count(s, old); m == 0 {
		return s // avoid allocation
	} else if n < 0 || m < n {
		n = m
	}

	// 置換実行
	t := make([]byte, len(s)+n*(len(new)-len(old)))
	w := 0
	start := 0
	for i := 0; i < n; i++ {
		j := start
		if len(old) == 0 {
			// ...
		} else {
			j += Index(s[start:], old)
		}
		w += copy(t[w:], s[start:j])
		w += copy(t[w:], new)
		start = j + len(old)
	}
	w += copy(t[w:], s[start:])
	return string(t[0:w])
}
```

```strings.Replace```の最初の方で```strings.Count```を読んでいます。これは一致する文字列の数を返す関数です。この```strings.Count```の中でも```strings.Index```が利用されています。

```go
func Count(s, substr string) int {
	// ...

	n := 0
	for {
		i := Index(s, substr)
		if i == -1 {
			return n
		}
		n++
		s = s[i+len(substr):]
	}
}
```

```func Replace```の処理はstrings.Indexに依存しているが確認できました。**strings.Index は Brute-force もしくは Rabin-karpアルゴリズムを利用しています**。```strigns.Index```の内部については[過去記事](https://po3rin.com/blog/go-rabin-karp)をご覧ください。

ここまでからわかる通り、Replaceで基本的な置換する為には```strings.Index```を最低でも2回は呼ぶ必要が出てきます。

## Replacer

Replacerの定義を確認しましょう。

```go
type Replacer struct {
	once   sync.Once // guards buildOnce method
	r      replacer
	oldnew []string
}

// replacer is the interface that a replacement algorithm needs to implement.
type replacer interface {
	Replace(s string) string
	WriteString(w io.Writer, s string) (n int, err error)
}
```

```once```フィールドでは何かをbuildする為の関数を一回だけ呼ぶ為の ```sync.Once``` が定義されています。何をbuildするのかは後ほど確認しましょう。また、**rでは内部実装を入れ替えるためにreplacer interfaceで抽象化しています**。```oldnew```は後でも詳しく確認しますが、置換したい文字列のペアのリストを保持します。

それでは```NewReplacer```を見ていきます。これは先ほど見た```Replacer```を初期化する関数です。

```go
func NewReplacer(oldnew ...string) *Replacer {
	if len(oldnew)%2 == 1 {
		panic("strings.NewReplacer: odd argument count")
	}
	return &Replacer{oldnew: append([]string(nil), oldnew...)}
}
```

sync.Onceはゼロ値で初期化できるので明示的な初期化は必要ありません。引数として渡された置換リストを保持して返すだけなので簡単ですね。

では早速、お目当ての```func (*Replacer) Replace```を見ていきましょう。

```go
// Replace returns a copy of s with all replacements performed.
func (r *Replacer) Replace(s string) string {
	r.once.Do(r.buildOnce)
	return r.r.Replace(s)
}
```

何かを一回だけbuildしてますね。この関数を呼ぶ前は```r.r```はまだ```nil```なのにも関わらず```r.r.Replace```を読んでいるので、```r.buildOnce```の中で```r.r```を初期化しているのが推測できます。

それではr.buildOnceを見ていきましょう。

```go
func (r *Replacer) buildOnce() {
	r.r = r.build()
	r.oldnew = nil
}
```

やはり```r.r```を初期化しています。```r.oldnew = nil``` から ```r.build```でしか置換するリストは必要ないことがわかります。

```Replacer.build```関数の実装をは少し長いのでお見せしませんが、内部では```replacer``` interfaceを実装している4種類のコンポジット型をだし分けています。

```go
type genericReplacer struct {
	root      trieNode
	tableSize int
	mapping   [256]byte
}

type singleStringReplacer struct {
	finder *stringFinder
	value  string
}

type byteReplacer [256]byte

type byteStringReplacer struct {
	replacements [256][]byte
	toReplace    []string
}
```

どのコンポジット型を使うかは```NewReplacer```に引数で渡す```oldnew```の値に依存します。例えば```oldnew```の値によって下記のように分岐します。

```go
[]string{"Golang", "Go"}               // singleStringReplacer
[]string{"Golang", "Go", "Clang", "C"} // genericReplacer
[]string{"a", "b"}                     // byteReplacer
[]string{"a", "bb"}                    // byteStringReplacer
```

任意の置換リストでどのreplacerが使われるかを確認できる単純化したコードを用意したので、もし興味があれば確かめてみてください。
[Go Playground](https://play.golang.org/p/x1s5N1kiFD-)

今回の探究では、最もよく使うと思われる```genericReplacer```を深ぼっていきましょう。もう一度```genericReplacer```のフィールドを確認します。

```go
type genericReplacer struct {
	root      trieNode
	tableSize int
	mapping   [256]byte
}
```

```trieNode```という型から内部で**Trie**が利用されていることが分かります。GoのTrieは下記のような構造になっています。

![Trieの例](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/trie1.png)

このTrieは置換用に実装されているのでvalueとして置換先のstringを格納します。

![Trieの例2](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/trie2.png)

TrieのNodeは下記のデータ構造になっています。

```go
type trieNode struct {
	value    string
	priority int
	prefix   string
	next     *trieNode
	table    []*trieNode
}
```

complete key の Node　であれば value に置換先の文字列が入ります。nextで次のNode、複数ならtableに格納されます。例えば置換リストが ```oldnew = []string{"Golang", "Go", "Clang", "C"}``` であれば下記のようなTrieが生成されます。

![Trieの例](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/trie3.png)

構造体で見るとこうなります。

```go
main.trieNode{
	value:    "",
	priority: 0,
	prefix:   "",
	next:     (*main.trieNode)(nil),
	table: []*main.trieNode{
		&main.trieNode{
			value:    "",
			priority: 0,
			prefix:   "lang",
			next: &main.trieNode{
				value:    "C",
				priority: 2,
				prefix:   "",
				next:     (*main.trieNode)(nil),
				table:    []*main.trieNode{},
			},
			table: []*main.trieNode{},
		},
		&main.trieNode{
			value:    "",
			priority: 0,
			prefix:   "olang",
			next: &main.trieNode{
				value:    "Go",
				priority: 4,
				prefix:   "",
				next:     (*main.trieNode)(nil),
				table:    []*main.trieNode{},
			},
			table: []*main.trieNode{},
		},
		(*main.trieNode)(nil),
		(*main.trieNode)(nil),
		(*main.trieNode)(nil),
		(*main.trieNode)(nil),
		(*main.trieNode)(nil),
	},
}
```

ちなみに深い構造体のデバッグでは下記のprint用パッケージが最高でおすすめです。

[![img1](https://github-link-card.s3.ap-northeast-1.amazonaws.com/k0kubun/pp.png)](https://github.com/k0kubun/pp)

Trieの構築は```GenericReplacer```の生成時に合わせて行なっています。

```go
func makeGenericReplacer(oldnew []string) *genericReplacer {
	r := new(genericReplacer)

	// oldの各byteをindexとしてmappingに1を立てる
	for i := 0; i < len(oldnew); i += 2 {
		key := oldnew[i]
		for j := 0; j < len(key); j++ {
			r.mapping[key[j]] = 1
		}
	}

	// tableSizeを計算これはmappingのビットカウントしてあげるだけ
	for _, b := range r.mapping {
		r.tableSize += int(b)
	}

	// mappingの0の箇所をtablesizeと同じbyteに
	// mappingの1の箇所は出現順に0からインクリエントした数を与える
	var index byte
	for i, b := range r.mapping {
		if b == 0 {
			r.mapping[i] = byte(r.tableSize)
		} else {
			r.mapping[i] = index
			index++
		}
	}

	r.root.table = make([]*trieNode, r.tableSize)

	// Trie構築
	for i := 0; i < len(oldnew); i += 2 {
		r.root.add(oldnew[i], oldnew[i+1], len(oldnew)-i, r)
	}
	return r
}
```

上のコードからわかるように実際にTrieに対するNodeの追加は```func (*trieNode) add```が行なっています。少し長いのでコードは載せませんが、上のコードで作った```t.mapping```と```t.tableSize```も利用してNodeを構築します。

ここまでで```func (*Replacer) Replace```が内部でTrieを利用していることは分かりました。実際に構築したTrieを使っている箇所をみていきましょう。Replaceの内部では下記のようにpublicなメソッドである```func (*genericReplacer)WriteString```を読んでいるだけです。

```go
func (r *genericReplacer) Replace(s string) string {
	buf := make(appendSliceWriter, 0, len(s))
	r.WriteString(&buf, s)
	return string(buf)
}
```

続いて```func (*genericReplacer)WriteString```で呼ばれている ```func (*genericReplacer) WriteString``` をみていきましょう。

```go
func (r *genericReplacer) WriteString(w io.Writer, s string) (n int, err error) {
	// ...

	// 文字列の長さだけぶん回す
	for i := 0; i <= len(s); {
		// ...

		// Trieに対するlookup
		val, keylen, match := r.lookup(s[i:], prevMatchEmpty)
		prevMatchEmpty = match && keylen == 0
		if match {
			// Trieで文字列がマッチしたときの処理
		}
		i++
	}
	// ...
}
```

対象の文字列に対して走査しながらTrieで探索していきます。なので、**```func (*Replacer) Replace``` ではTrieを使って複数の置換を1回の走査で完了することができます**。

## 内部構造を知ると仮説が立てれる

ここで Replace と Replacer のアルゴリズム、データ構造を紹介してきました。ここまでの知識があればどちらを使えば良いかの仮説が立てれます。

* 置換ペアが１つならTrieを構築するコストがあるのでReplaceで十分ではないか？
* 置換ペアが多いなら文字列全体の走査が1回でおわるReplacerを使うべきでは？

このような仮説が立てれればどのようなパフォーマンステストを実施すれば良いかも分かります。

## まとめ

```func Replace``` と ```func (*Replacer) Replace``` の内部で利用されているアルゴリズム&データ構造を紹介しました。今回はコードを詳細に追わず、流れだけを追ったので、もっと詳しく知りたい方(Nodeにkeyの値がないけどなんで？？とか)は実際にコードを読んでみてください。
