---
title: Goを読んでDockerの抽象構文木の構造をサクッと理解する
cover: img/gopher.png
date: 2019/03/05
id: dockerfile-ast
description: Dockerfile の抽象構文木ってどうなっているんだろうと思い調べてみました。
---

こんにちはpo3rinです。Dockerfile の抽象構文木(以降 AST と呼ぶ)ってどうなっているんだろうと思い調べてみました。

## Dockerfile の AST を所得する

下記の Dockerfile の AST をみてみます。

```Dockerfile
FROM golang:latest

WORKDIR /go
ADD . /go

CMD ["go", "run", "main.go"]
```

moby の buildkit が Dockerfile の Parser を提供しているのでそれを使います。

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func main() {
	f, _ := os.Open("./Dockerfile")
	r, _ := parser.Parse(f)
	ast := r.AST.Dump()
	fmt.Printf("%+v", ast)
}
```

これで文字列化された Dockerfile の AST の Dump がみれます。

```cmd
$ go run main.go
(from "golang:latest")
(workdir "/go")
(add "." "/go")
(cmd "go" "run" "main.go")
```

かなり、簡略化して表示されるので、AST の構造までは覗けません。そこはコードを読んでいく必要がありそです。

## DockerfileのASTの構造を知る

github.com/moby/buildkit/frontend/dockerfile/parser を読むと下記の構造体があります。

```go
// Node is a structure used to represent a parse tree.
type Node struct {
	Value      string          // actual content
	Next       *Node           // the next item in the current sexp
	Children   []*Node         // the children of this sexp
	Attributes map[string]bool // special attributes for this node
	Original   string          // original line used before parsing
	Flags      []string        // only top Node should have this set
	StartLine  int             // the line in the original dockerfile where the node begins
	endLine    int             // the line in the original dockerfile where the node ends
}
```

Node は解析木を表すために使用される構文コードです。基本的に Value、 Next、 Children の3つのフィールドを使います。Value は現在のトークンの文字列値です。 Next は次のトークンで、Children はすべての子Nodeのスライスになっています。

他の言語のASTをみたことある人はかなりシンプルな構造だと感じると思いますが、実は公式も「素直に言ってかなりお粗末」だと言っています。しかし、Dockerfile はプログラミング言語よりもシンプルなので、Node もシンプルであることはむしろ効果的だと言っています。

> This data structure is frankly pretty lousy for handling complex languages, but lucky for us the Dockerfile isn't very complicated. This structure works a little more effectively than a "proper" parse tree for our needs.

さてNodeが実際にどのように構成されているのか見てみましょう。ネストした構造体を覗くときは```github.com/kr/pretty```が便利なのでこれを使いましょう。まずは大枠を掴みます。

```go
package main

import (
	"os"

	"github.com/kr/pretty"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func main() {
	f, _ := os.Open("./Dockerfile")
	r, _ := parser.Parse(f)
	pretty.Print(r.AST)
}
```

出力はこうなります。

```go
&parser.Node{
    Value:    "",
    Next:     (*parser.Node)(nil),
    Children: {
        &parser.Node{
            Value: "from",
            Next:  &parser.Node{
                Value:      "golang:latest",
                Next:       (*parser.Node)(nil),
                Children:   nil,
                Attributes: {},
                Original:   "",
                Flags:      nil,
                StartLine:  0,
                endLine:    0,
            },
            Children:   nil,
            Attributes: {},
            Original:   "FROM golang:latest",
            Flags:      {},
            StartLine:  1,
            endLine:    1,
        },
        &parser.Node{
            Value: "workdir",
            Next:  &parser.Node{
                Value:      "/go",
                Next:       (*parser.Node)(nil),
                Children:   nil,
                Attributes: {},
                Original:   "",
                Flags:      nil,
                StartLine:  0,
                endLine:    0,
            },
            Children:   nil,
            Attributes: {},
            Original:   "WORKDIR /go",
            Flags:      {},
            StartLine:  3,
            endLine:    3,
        },
        &parser.Node{
            Value: "add",
            Next:  &parser.Node{
                Value: ".",
                Next:  &parser.Node{
                    Value:      "/go",
                    Next:       (*parser.Node)(nil),
                    Children:   nil,
                    Attributes: {},
                    Original:   "",
                    Flags:      nil,
                    StartLine:  0,
                    endLine:    0,
                },
                Children:   nil,
                Attributes: {},
                Original:   "",
                Flags:      nil,
                StartLine:  0,
                endLine:    0,
            },
            Children:   nil,
            Attributes: {},
            Original:   "ADD . /go",
            Flags:      {},
            StartLine:  4,
            endLine:    4,
        },
        &parser.Node{
            Value: "cmd",
            Next:  &parser.Node{
                Value: "go",
                Next:  &parser.Node{
                    Value: "run",
                    Next:  &parser.Node{
                        Value:      "main.go",
                        Next:       (*parser.Node)(nil),
                        Children:   nil,
                        Attributes: {},
                        Original:   "",
                        Flags:      nil,
                        StartLine:  0,
                        endLine:    0,
                    },
                    Children:   nil,
                    Attributes: {},
                    Original:   "",
                    Flags:      nil,
                    StartLine:  0,
                    endLine:    0,
                },
                Children:   nil,
                Attributes: {},
                Original:   "",
                Flags:      nil,
                StartLine:  0,
                endLine:    0,
            },
            Children:   nil,
            Attributes: {"json":true},
            Original:   "CMD [\"go\", \"run\", \"main.go\"]",
            Flags:      {},
            StartLine:  6,
            endLine:    6,
        },
    },
    Attributes: {},
    Original:   "",
    Flags:      nil,
    StartLine:  1,
    endLine:    6,
}
```

全体像は下記のようになります。

<img width="689" alt="node.png" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2019/1551744000/qiita-a3934f47b5e390acfdfd-1.png">

1行の中でもtokenごとにNodeに分けられます。一番上のNodeがルートNodeと呼ばれます。ルートNode自身はValueを持たずChildren Nodeの一覧を保持します。こう見るとルートNodeのChildrenの数はイメージのレイヤ数と基本的に一致します。Next Nodeは同じ行の中の次のtokenのNodeです。Flagsは ```--from=builder```などのDockerfile上で使われるFlagか格納されます。StartLineとendlineは文字通り、そのノードのDockerfileにおける行数です。Originalは解析前に使用された元の行を格納しています。

## PrintWarnings を使って Dockerfile に対する Warning を見る

Dockerfile をビルドする際に稀に出る Warning は Dockerfile の parse 時に検知しています。moby は下記のようなparseした後のASTに対してWarningをだすメソッドもあります。

```go
func (r *Result) PrintWarnings(out io.Writer)
```

このような空白行があると記載がDockerfileにあると

```Dockerfile
RUN echo "Hello" && \
    # (空行)
    echo "Docker AST"
```

このようなDockerfileに対して PrintWarnings メソッドを使うと下記のような Warning を吐きます。

```
[WARNING]: Empty continuation line found in:
    RUN echo "Hello" &&     echo "Docker AST"
[WARNING]: Empty continuation lines will become errors in a future release.
```

## おまけ : ASTを使ったDockerfileのLint

Dockerfile の AST が手に入ったので簡単なlintツールも作れそうです。Dockerfile におけるベストプラクティスの一つはレイヤの数を最小にすることです。つまり、二回連続でRUNを使っている Dockerfile は一つのコマンドに統合すべきです。下のコードは RUN を二連続で呼び出している部分を検知します。

```go
package main

import (
	"log"
	"os"

	"github.com/kr/pretty"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func main() {
	f, _ := os.Open("./Dockerfile")
	r, _ := parser.Parse(f)
	pretty.Print(r.AST)

	var valueToken string
	for _, child := range r.AST.Children {
		if valueToken == child.Value {
			log.Fatal("RUN is used in two consecutive layers")
		}
		valueToken = child.Value
	}
}
```

こんな Dockerfile は

```Dockerfile
FROM golang:latest

WORKDIR /go
ADD . /go

RUN echo "Hello"
RUN echo "Docker AST"

CMD ["go", "run", "main.go"]
```

こういう風に検知できますね。

```console
$ go run main.go
2019/03/04 23:36:10 RUN is used in two consecutive layers
```

Dockerfile の AST はシンプルゆえ複雑なことはできませんが、これくらいの検知なら十分です。

## 終わりに

簡単に Dockerfile の AST を追ってみました。今後このASTを使ってどのようにビルドしているのか追ってみて、暇ならまた記事にします。

##追記
暇だったので、Dockerfile 抽象構文木から LLB を生成するフローを追う記事を書きました。ほぼこの記事の続編です。
https://qiita.com/po3rin/items/f414660bd2a6173c587a

