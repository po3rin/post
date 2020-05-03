---
title: Go の -race option は内部で何をしているのか。何を検知しないのか。
cover: img/gopher.png
date: 2019/09/27
id: go-race
description: -race をつけてCIを通しているのにAPIがデータ競合で落ちてしまいました。調べていたら -race がそもそも何をしているかに行き着いたので簡単に共有します。
tags:
    - Go
---

## -race とは

```-race```はコンパイラフラッグの一種で競合を検知するのに便利です。例えば下記のコードはmapへの読み書きが同時に起こってパニックするコードです。これを```go run main.go -race```のように実行するとwarningを出してくれます。

```go
package main

import (
	"fmt"
	"strconv"
	"time"
)

func main() {

	m := make(map[string]int)

	go func() {
		for i := 0; i < 1000; i++ {
			m[strconv.Itoa(i)] = i // write
		}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			fmt.Println(i, m[strconv.Itoa(i)]) // read
		}
	}()

	time.Sleep(time.Second * 5)

}
```

下記のように実行するとWarningを出します。

```
$ go run -race main.go
```

## -race はそもそも何をしているのか

内部では```C/C++```用の競合検出ライブラリが使われています。簡単に言うと、競合を実行時に検出するコードを出力することができるライブラリです。
https://github.com/google/sanitizers/wiki/ThreadSanitizerCppManual

もちろん```Go```のコンパイラはこれをサポートしており、実際に```runtime/race```の```README.md```には```ThreadSanitizer```が使われていることが明記されています。

https://github.com/golang/go/tree/master/src/runtime/race

> untime/race package contains the data race detector runtime library. It is based on ThreadSanitizer race detector, that is currently a part of the LLVM project (http://llvm.org/git/compiler-rt.git).

コンパイル時に実際にどのようなコードが仕込まれているかを見てみましょう。例えば下記のコードをビルドした場合をみてみます。

```go
package main

func Inc(x *int) {
    *x++
}
```

普通にビルドすると下記のようにコンパイルされます。

```
...

pcdata  $2, $1
pcdata  $0, $1
movq    "".x+8(SP), AX
pcdata  $2, $0
incq    (AX)

...
```

```-race``` をつけると下記のようにコンパイルされます。

```
...

pcdata  $2, $1
movq    "".x+32(SP), AX
testb   AL, (AX)
pcdata  $2, $0
movq    AX, (SP)
call    runtime.raceread(SB)
pcdata  $2, $1
movq    "".x+32(SP), AX
movq    (AX), CX
movq    CX, ""..autotmp_4+8(SP)
pcdata  $2, $0
movq    AX, (SP)
call    runtime.racewrite(SB)
movq    ""..autotmp_4+8(SP), AX
incq    AX
pcdata  $2, $2
pcdata  $0, $1
movq    "".x+32(SP), CX

...
```

コンパイラーが```call runtime.raceread(SB)```のように同時に到達可能な各メモリー位置に読み取りおよび書き込みを検知する命令を追加しています。ご覧の通り、```-race```をつけると命令が増えてパフォーマンスが落ちるのでビルドしたバイナリを本番に乗っけるのはやめましょう。[go build -race でプログラムが遅くなってた話](https://qiita.com/smith-30/items/be4d92c251d2b2b39bd3)

ちなみにGoコンパイラが吐き出すアセンブリは```Compiler Explorer```というサービスで簡単に確認できて便利です。

[Compiler Explorer](https://go.godbolt.org/z/IWw4hk)

## -raceを使ってるのに競合が起きる理由

-raceでは競合を検知するための命令を追加しているので、そこに到達しないと競合を検知しません。つまりテスト時に-raceをつけてPASSしたからと言って競合が発生しないとは言えません。

Goのドキュメントにも明示されていました。
https://golang.org/doc/articles/race_detector.html#How_To_Use

> To start, run your tests using the race detector (go test -race). The race detector only finds races that happen at runtime, so it can't find races in code paths that are not executed. If your tests have incomplete coverage, you may find more races by running a binary built with -race under a realistic workload.

テストカバレッジが低い場合はテストだけで全ての競合が消せたと判断せずに、実際に```-race``` をつけてビルドしてみて動かしてみることが大切です。

## Referece

Golang race detection
https://krakensystems.co/blog/2019/golang-race-detection

