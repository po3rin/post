---
title: Go の strings.Index の内部実装と Rabin-Karp アルゴリズム
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/text.jpeg
date: 2019/12/12
id: go-rabin-karp
description: strings.Index 関数の内部実装と Rabin–Karp アルゴリズムが面白かったので解説します。
tags:
    - Go
    - Algorithm
---

本記事は[Go Advent Calendar 2019](https://qiita.com/advent-calendar/2019/go)の12日目の記事です。前回の記事は [Go の命名規則](https://micnncim.com/post/2019/12/11/go-naming-conventions/) でした。

## strings.Index とは

簡単に言うと、任意の文字列の中にある文字列が含まれているかをチェックして、もしあればその始まりのindexの値を返す関数です。
[https://github.com/golang/go/blob/go1.13.1/src/strings/strings.go#L1027](https://github.com/golang/go/blob/go1.13.1/src/strings/strings.go#L1027)

```go
func Index(s, substr string) int
```

```s``` 中に ```substr``` が最初に出現するindexを返します。
公式の例を引用しますが、この関数を使うと下記の結果になります。

```go
fmt.Println(strings.Index("chicken", "ken")) // 4
fmt.Println(strings.Index("chicken", "dmr")) // -1
```

substrが見つからない時は -1 を返します。これをGoがどのように実装しているか不思議じゃないですか？

## Brute-force

```strings.Index```関数の中では```s```や```substr```の長さごとに```switch```で様々な分岐があります。その中で短い文字列なら愚直に総当たり(**Brute-force**)でやっつけています。例えば ```s``` と ```substr```が両方ともある程度小さい場合は ```bytealg.IndexString``` を利用しているケースを見つけることができます。

```go
// in bytealg package
const MaxBruteForce = 64

// in string package
func Index(s, substr string) int {
	n := len(substr)
	switch {
	// ...
	case n <= bytealg.MaxLen:
		// Use brute force when s and substr both are small
		if len(s) <= bytealg.MaxBruteForce {
			return bytealg.IndexString(s, substr)
		}
	// ...
}
```

ここで```bytealg.IndexString```はGoで記述されていない実装を持っている関数です。パッケージの同階層にあるアセンブラによる定義がなされています。

```go
//go:noescape
func IndexString(a, b string) int
```

さて ```bytealg.MaxLen``` という変数が出てきましたが、これはプロセッサによって変わる値です。

```go
// in bytealg package
func init() {
	if cpu.X86.HasAVX2 {
		MaxLen = 63
	} else {
		MaxLen = 31
	}
}
```

そして ```bytealg.MaxBruteForce```は```64```に設定されています。

```go
// in bytealg package
const MaxBruteForce = 64
```

つまり先ほどのコードは ```s``` や ```substr``` の長さが50前後の場合はだいたい総当たりを使っていることを意味します。

実はある程度の長さまでなら総当たりで見つけてしまいます。下記のコードは全部追わなくても大丈夫ですが、```IndexByte```でハズレ(文字列が見つからない)を引き続けたら途中から```IndexString```に切り替えています。このように総当たり戦略を組み合わせて文字列検索をしています。

```go
func Index(s, substr string) int {
	n := len(substr)
	switch {
	// ...
	case n <= bytealg.MaxLen:
		// ...
		c0 := substr[0]
		c1 := substr[1]
		i := 0
		t := len(s) - n + 1
		fails := 0
		for i < t {
			if s[i] != c0 {
				// IndexByte is faster than bytealg.IndexString, so use it as long as
				// we're not getting lots of false positives.
				o := IndexByte(s[i:t], c0)
				if o < 0 {
					return -1
				}
				i += o
			}
			if s[i+1] == c1 && s[i:i+n] == substr {
				return i
			}
			fails++
			i++
			// Switch to bytealg.IndexString when IndexByte produces too many false positives.
			if fails > bytealg.Cutover(i) {
				r := bytealg.IndexString(s[i:], substr)
				if r >= 0 {
					return r + i
				}
				return -1
			}
		}
		return -1
	}
}
```

今までは短い文字列の場合だけでしたがもっと長い文字列の場合 (厳密にいうと```n > bytealg.MaxLen```) は処理が変わります。

## Rabin-karp

長い文字列の時、つまり```n > bytealg.MaxLen:```のケースでも、短い文字列の時と同じようにある程度まで **Brute-force** で頑張りますが、ちょっと厳しそうとなったら**Rabin-Karp**アルゴリズムを利用します。これは文字列から計算されるハッシュを使用してテキスト内に任意の文字列があるかを検索するアルゴリズムです。[Rabin–Karp algorithm](https://en.wikipedia.org/wiki/Rabin%E2%80%93Karp_algorithm)

実際にGoでは```indexRabinKarp```という関数がRabin-karpのアルゴリズムを実装しています。
[https://github.com/golang/go/blob/go1.13.1/src/strings/strings.go#L1107](https://github.com/golang/go/blob/go1.13.1/src/strings/strings.go#L1107)

```go
func indexRabinKarp(s, substr string) int
```

文字列が等しいなら、そこから計算されるハッシュも等しいという事実を利用して、一致する文字列を探します。しかし逆は言えず、ハッシュが一致したからといって文字列が一致するとは限りません。つまり、同じ文字列が出現する箇所をハッシュで当たりをつけていく感覚です。ステップとしては

**1: ハッシュが同じ文字列を見つける**
**2: 実際に文字列が一致しているか調べる**

と言うステップを踏みます。

ハッシュを計算するには様々な方法がありますが、Goでは**ローリングハッシュ関数**を利用します。(なぜこの関数を使っているのかは後で説明します)この関数では下記のようにハッシュを計算します。(以降この記事では分かりやすさのために累乗を```^```で表しています)

```go
hash = substr[0] * prime^(n-1) + substr[1] * prime^(n-2) + ... + substr[n-1] * prime^(0)
```

primeはある任意の**素数**です。例えば文字列```karp```のhashの値は次の式で計算します。

```go
hash = "karp"[0] * prime^3 + "karp"[1] * prime^2 + "karp"[2] * prime^1 + "karp"[3] * prime^0
```

Goを最近書き始めた方は驚くかもしれませんが、Goで文字列に添字アクセス(```"karp"[0]```)すると```k```ではなく```k```の```byte```が取得できます。これをchar固有の値として計算に使っています。

```go
fmt.Println("karp"[0]) // 107
```

そして、Goのローリングハッシュで使う素数は**1677619**が使われています。**値が大きい素数だと違う文字列のハッシュと衝突する確率を減らせるためです。**

```go
const primeRK = 16777619
```

よってこの素数と最初に紹介した式を使うと文字列```karp```のハッシュは

```go
// "karp"[0] * prime^3 + "karp"[1] * prime^2 + "karp"[2] * prime^1 + + "karp"[3] * prime^0
107 * 16777619^3 + 97 * 16777619^2 + 114 * 16777619^1 + 112 * 16777619^0
```

と計算できます。

しかしなぜ大きな素数の中でも```1677619```が使われているのでしょうか。実はこの素数は**2進数展開したときにハミング重みが小さいように選ばれています。**ハミング重みとはビット列の1の数です。Goで確認すると確かに1が少ないですね。

```go
fmt.Println(strconv.FormatInt(16777619, 2))
// 1000000000000000110010011
```

実際にGoで使われているhash計算の関数```hashStr```をみてみましょう。
https://github.com/golang/go/blob/go1.13.1/src/strings/strings.go#L44

```go
func hashStr(sep string) (uint32, uint32) {
	hash := uint32(0)
	// hash計算
	for i := 0; i < len(sep); i++ {
		hash = hash*primeRK + uint32(sep[i])
	}

	// pow計算
	var pow, sq uint32 = 1, primeRK
	for i := len(sep); i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}
```

なぜか```pow```なる値を返していますが、これは後で説明します。注目すべきは```hash```を計算している箇所です。
前に紹介したhash計算の式と少し見た目が違いますが、これは```math.Pow```を使わずに計算する方法を取っているからです。式変形をしているので計算の見た目は違いますが当然計算結果は同じになります。例えば```"hi"```と言う文字列のハッシュ値はこちらになります。

```go
hash, _ := hashStr("hi")
fmt.Println(hash) // 1744872481
```

そもそもですが、なぜ文字列一致を調べるのにローリングハッシュを使うのでしょうか。例で説明します。```karp```の中から```ap```の出現位置を調べたい場合、まず先頭の```ka```のハッシュを計算します。


```go
// k: 107, a:92
h0 = 107 * prime^1 + 92 * prime^0
```

当然 ```ap``` のhash値と合わないので ```ka``` の次の ```ar``` のハッシュを調べます。その際にゼロからまた計算するとまたコストがかかってしまいます。そのため、先ほど計算した値を利用します。つまり、先ほどのハッシュから```ka```の```k```の項を引いて、```r```の項だけを足せばよくなります。下記のような式で計算すれば**文字列の長さに関係なく数ステップで計算が完了してしまいます。**

```go
// k: 107, r: 114
h1 = h0 * prime + 114 - 107 * prime^len(substr)
```

```prime^len(substr)``` をかけているのは素数の次数を合わせるためです。まとめると、もし文字列がマッチしなくても既に計算した値を使って次のhashを計算することで計算を最適化しているのです。このようにローリングしながらhashを計算する手法をローリングハッシュと言います。

では実際に ```indexRabinKarp``` の内部をみてみましょう。
[https://github.com/golang/go/blob/go1.13.1/src/strings/strings.go#L1107](https://github.com/golang/go/blob/go1.13.1/src/strings/strings.go#L1107)

```go
func indexRabinKarp(s, substr string) int {
	// subst の hash の取得
	hashss, pow := hashStr(substr)
	n := len(substr)
	var h uint32

	// substrと比較する最初のパターンのhashの計算
	for i := 0; i < n; i++ {
		h = h*primeRK + uint32(s[i])
	}
	if h == hashss && s[:n] == substr {
		return 0
	}
	for i := n; i < len(s); {
		h *= primeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i-n])
		i++
		if h == hashss && s[i-n:i] == substr {
			return i - n
		}
	}
	return -1
}
```

Goでは最初の```substr```のhash値を計算する時に一緒に計算したpowを使って計算しています。powは先ほどの例のローリングハッシュの計算式で示した```prime^len(substr)```の値に等しくなっています。つまりpowはローリングハッシュで引く項の計算に使っていたわけです。

これでGoのIndex関数の内部アルゴリズムを覗くことができました。

## おまけ

この記事書いてる間にコメントのミスに気づいたのでちゃっかりGoへのコントリビュートに成功しました。

[https://go-review.googlesource.com/c/go/+/210298](https://go-review.googlesource.com/c/go/+/210298)

