# Go言語の golang/go パッケージで初めての構文解析

この記事は、Go Advent Calendar 2018の5日目の記事です。

「Go言語でつくるインタプリタ」を読んで、プログラミング言語の「仕組み」に興味がでてきた。そして、Go言語だと構文解析が簡単に出来るとの噂が！ということで ```golang/go``` パッケージを触ってみると、Go言語で出来る事のイメージが更に広がった。せっかくなので自分のような初心者向けにハンズオン形式で紹介していきます。最終的にGo言語のソースコードから抽象構文木を所得して、そこから更に、抽象構文木を書き換えてGo言語のソースコードに変換するところまでやります。

## とりあえず抽象構文木を手に入れてみる

抽象構文木 (abstract syntax tree、AST) とは言語の意味に関係ない情報を取り除き、意味に関係ある情報のみを取り出した（抽象した）木構造のデータ構造です。よってGo言語においてはスペース、括弧、改行文字などが省かれた木構造のデータ構造になる。これは見てみないと分からないと思うので、まずは抽象構文木を所得する術を確認し、早速、抽象構文木を確認しましょう。まずは下記を実行してみてください。

```go
package main

import (
    "go/ast"
    "go/parser"
)

func main() {
    // ASTを所得
    expr, _ := parser.ParseExpr("A + 1")

    // AST をフォーマットして出力
    ast.Print(nil, expr)
}
```

これを実行すると、速攻で抽象構文木が手に入る。

```bash
$ go run main.go

 0  *ast.BinaryExpr {
 1  .  X: *ast.Ident {
 2  .  .  NamePos: 1
 3  .  .  Name: "A"
 4  .  .  Obj: *ast.Object {
 5  .  .  .  Kind: bad
 6  .  .  .  Name: ""
 7  .  .  }
 8  .  }
 9  .  OpPos: 3
10  .  Op: +
11  .  Y: *ast.BasicLit {
12  .  .  ValuePos: 5
13  .  .  Kind: INT
14  .  .  Value: "1"
15  .  }
16  }
```

ここでは2つのパッケージが使われている。

#### go/parser
```go/parser``` はGo言語の構文解析を行う為のパッケージです。名前の通りGo言語用のParser(テキストをプログラムで扱えるようなデータ構造に変換する)を実装しています。出力はGo言語の抽象構文木（AST）になります。今回使ったメソッドは下記になります。

[go/parser.ParseExpr](https://godoc.org/go/parser#ParseExpr)

```go
func ParseExpr(x string) (ast.Expr, error)
```

go/parser ドキュメントはこちら
https://godoc.org/go/parser

#### go/ast
go/ast はGo言語の抽象構文木 (AST) を表現する為の型が定義されているパッケージです。

先ほど見た ```go/parser.ParserExpr``` はGo言語の式(Expression) を構文解析し、式の抽象構文木を表現する ```ast.Expr``` インターフェースを返しています。```ast.Expr``` の構造は下のようになります。

[go/ast.Expr](https://godoc.org/go/ast#Expr)

```go
type Expr interface {
	Node
	exprNode()
}
```

main関数で使った ```ast.Print``` は抽象構文木を読みやすい形で出力してくれます。

go/ast のドキュメントはこちら
https://godoc.org/go/ast

### Go言語のファイルから抽象構文木を手にいれる

上で実装したように毎回string型でコードを渡すのはダルいので、Go言語で書かれたファイルの入力からASTを所得しましょう。

まずは構文解析対象となる example/example.go を作りましょう。

```go
package example

import "log"

func add(n, m int) {
	log.Println(n + m)
}
```

ではこのファイルを入力として構文木を出してみましょう。
自分は下記のディレクトリ構造をとりますが皆さんお好みで！

```
.
├── example
│   └── example.go
└── main.go
```

まずは構文解析対象となる ```example.go``` を実装します。

```go
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

func main() {
    fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "./example/example.go", nil, parser.Mode(0))

	for _, d := range f.Decls {
		ast.Print(fset, d)
	}
}
```

早速実行してみましょう！example.go の抽象構文木が所得できます。(結果が長いので省略します。実行して確認してみてください)

token.NewFileSet()は構文解析によって得られたASTのノードの詳細な位置情報(ファイル名と行番号、カラム位置など)を保持する ```token.FileSet``` 構造体のポインタを返しています。なぜこれを生成する必要があるのかは後々説明します。

```parser.ParseFile``` の実装の詳細は省略しますが(ドキュメントは https://godoc.org/go/parser#ParseFile )、src が nil であるときは filename に指定されたファイルパスの内容を読み込みます。返ってくるのは ast.File 構造体です。

[go/ast.File](https://godoc.org/go/ast#File)

```go
type File struct {
    Doc        *CommentGroup   // associated documentation; or nil
    Package    token.Pos       // position of "package" keyword
    Name       *Ident          // package name
    Decls      []Decl          // top-level declarations; or nil
    Scope      *Scope          // package scope (this file only)
    Imports    []*ImportSpec   // imports in this file
    Unresolved []*Ident        // unresolved identifiers in this file
    Comments   []*CommentGroup // list of all comments in the source file
}
```

その中で先ほど実装に使った ```ast.Decls``` (Declarations の略)はトップレベルで宣言されたノードが返ってきます。

他のフィールドに目を当てると、例えば ```ast.File.Name``` はパッケージ名を格納し、```ast.File.Imports``` はこのファイルで読み込んでいるパッケージのノードを表現します。

例えば下記では、パッケージのノードを所得して出力しています。

```go
func main() {
    fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "./example/example.go", nil, parser.Mode(0))

	for _, d := range f.Imports {
		ast.Print(fset, d)
	}
}
```

これを実行してみます。

```bash
$ go run main.go

 0  *ast.ImportSpec {
 1  .  Path: *ast.BasicLit {
 2  .  .  ValuePos: ./example/example.go:5:2
 3  .  .  Kind: STRING
 4  .  .  Value: "\"log\""
 5  .  }
 6  .  EndPos: -
 7  }
```

このように 構文解析することでGo言語のソースコードから必要な情報を所得しています。

## 抽象構文木(AST)の中をのぞいてみる

先ほど見た ```ast.File``` 構造体のなかにある ```ast.Decl``` の存在に思いを馳せることから始めましょう！
みてみると ```Node``` インターフェースが実装されています。

[go/ast.Decl](https://godoc.org/go/ast#Decl)

```go
type Decl interface {
	Node
	declNode()
}
```

実は抽象構文木のノードに対応する構造体は、全てこちらの ast.Node インタフェースを実装しています。

それでは、そもそもの Node インターフェースの中身はどうなっているのでしょうか。

[go/ast.Node](https://godoc.org/go/ast#Node)

```go
type Node interface {
    Pos() token.Pos // position of first character belonging to the node
    End() token.Pos // position of first character immediately after the node
}
```

こちらも インターフェース型です。そのノードのソースコード上での位置を表現します。```Node.Pos()```や```Node.End()```については後にみていきます。今はソースコード上の位置を返してくれるんだなくらいに覚えておいてください。

```Decl``` ノードが ```ast.Node``` インターフェース を実装しているのを確認しましたが、他にも ast.Node インターフェースを実装しているものがあります。ここでは前に紹介したノードも含めて3つの主なサブインターフェースを紹介します。

① [go/ast.Decl](https://godoc.org/go/ast#Decl)

```go
type Decl interface {
	Node
	declNode()
}
```

宣言に関するノード（declaration）。import や type や func がここに大別される。先ほど触れた```ast.File``` にも実装されています。

② [go/ast.Expr](https://godoc.org/go/ast#Expr)

```go
type Expr interface {
	Node
	exprNode()
}
```

式に関するノード（expression） 識別子や演算、型など。この記事の初めに ```parser.ParseExpr``` でGo言語の式(```A + 1```)をstring型で渡してexpressionノードに変換する例を紹介しました。

③ [go/ast.Stmt](https://godoc.org/go/ast#Stmt)

文に関するノード（statement） if や for、switch など

```go
type Stmt interface {
	Node
	stmtNode()
}
```

これ以外にもファイルやコメントなど、これらに分類されない構文ノードも存在します。
[参考 Nodeの構成](https://qiita.com/po3rin/items/a19d96d29284108ad442#%E3%81%8A%E3%81%BE%E3%81%91-node%E3%81%AE%E6%A7%8B%E6%88%90)

ここまで紹介すれば、あとはGo言語のNodeの構造を参考にすれば、Go言語のソースコードから好きなものを取り出すことができます。

## 抽象構文木(AST)のトラバース

さて、抽象構文木が所得できたら、木構造やグラフの全てのノードを辿り(トラバース)し、再帰的に処理したくなってきます。なぜならフィールド名等を一個づつ指定して目的のノードにアクセスするのは type assertion や type switch が多発する為、非常にめんどくさいです。しかし、ご安心を。astパッケージには抽象構文木をトラバースする便利な関数が提供されています。

まずは使い方をみてみましょう。

```go
func main() {
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "./example/example.go", nil, parser.Mode(0))

	ast.Inspect(f, func(n ast.Node) bool {
		if v, ok := n.(*ast.FuncDecl); ok {
            fmt.Println(v.Name)
		}
		return true
	})
}
```

上はソースコードから関数名だけを引っこ抜いてくる処理です。実行してみてください。add関数の名前だけが所得できています。

```bash
$ go run main.go
add
```

```ast.Inspect``` は、ASTを深さ優先でトラバースする関数です。ASTの任意のNodeを渡せばトラバースできます。そして、```ast.FuncDecl``` は ```ast.Decl```インターフェースを実装しており、関数の宣言に関するノードを担当しています。[参考 Nodeの構成](https://qiita.com/po3rin/items/a19d96d29284108ad442#%E3%81%8A%E3%81%BE%E3%81%91-node%E3%81%AE%E6%A7%8B%E6%88%90)

## ソースコードの位置を所得する

さて、静的解析ではファイル名や行番号などを返したい場合があります。位置の所得についてみていきます。そういえば全ての Node には位置情報を返すメソッドが定義されていました。

```go
type Node interface {
    Pos() token.Pos // position of first character belonging to the node
    End() token.Pos // position of first character immediately after the node
}
```

これを先ほどのコードに入れてファイル上の位置が所得できるか試してみましょう。

```go
func main() {
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "./example/example.go", nil, parser.Mode(0))

	ast.Inspect(f, func(n ast.Node) bool {
		if v, ok := n.(*ast.FuncDecl); ok {
            fmt.Println(v.Name)
            fmt.Println(v.Pos())
		}
		return true
	})
}
```

そして実行します。

```bash
$ go run main.go
add
32
```

32！？なんの数字？？となります。```ast.Node.Pos()``` は実はノードに属する最初の文字の位置からのbyte数を返してしまします。

ではどうやって行番号やカラム位置などに変換するのでしょうか。ここで go/token パッケージに注目する時がきました。もう一度先ほどのソースコードをみてみましょう。

```go
func main() {
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "./example/example.go", nil, parser.Mode(0))

	ast.Inspect(f, func(n ast.Node) bool {
		if v, ok := n.(*ast.FuncDecl); ok {
            fmt.Println(v.Name)
            fmt.Println(v.Pos())
		}
		return true
	})
}
```

実は今まで使っていた ```token.NewFileSet()``` は構文解析によって得られたASTのノードの詳細な位置情報(ファイル名と行番号、カラム位置など)を保持する ```token.FileSet``` 構造体のポインタを返しています。

これを使ってノードの詳細な位置を復元できます。

```go
func main() {
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "./example/example.go", nil, parser.Mode(0))

	ast.Inspect(f, func(n ast.Node) bool {
		if v, ok := n.(*ast.FuncDecl); ok {
            fmt.Println(v.Name)
            // v.Pos() から 詳細な位置情報(ファイル名と行番号、カラム位置)を復元
            fmt.Println(fset.Position(v.Pos()))
		}
		return true
	})
}
```

これを実行してみましょう。

```bash
go run main.go
add
./example/example.go:5:1
```

さっきの 32 という整数から　詳細な位置情報が復元できました。```token.FileSet``` は抽象構文木のノードの位置情報を保持するので、これの情報を使って、```token.FileSet``` から生えている Positionメソッドで、Pos値を詳細な位置情報Position値(ファイル名と行番号、カラム位置)に変換しています。上の例のように1度生成した```token.FileSet```は他で使いまわすのでどっかで保持しておくと良いですね。

[go/token.FileSet.Positon](https://godoc.org/go/token#FileSet.Position)

```go
func (s *FileSet) Position(p Pos) (pos Position)
```

## コードから得たASTを書き換えてファイルに出力する

さて、ここまででASTの構造をのぞいてきました。ここから独自のツールを作るとなると、ASTを書き換えてソースコードに出力するという流れが出てくる(例えば、関数名を書き換えたい、Field名を書き換えたい、コードをASTから自動生成したい等)。そのために、ASTを実際に書き換えて、それをGo言語のコードに変換して出力する流れをみてみましょう。まずは```example/example.go```の関数名"add"を"plus"に書き換えてみます。

```go
func main() {
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "./example/example.go", nil, parser.Mode(0))

	ast.Inspect(f, func(n ast.Node) bool {
		if v, ok := n.(*ast.FuncDecl); ok {
            // ノードを直で書き換える
			v.Name = &ast.Ident{
				Name: "plus",
			}
		}
		return true
    })

    // 指定したFileがあったら開く、なかったら作る。
    file, err := os.OpenFile("example/result.go", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

    // go/printer パッケージの機能でASTからソースコードを作る。
	pp := &printer.Config{Tabwidth: 8, Mode: printer.UseSpaces | printer.TabIndent}
	pp.Fprint(file, fset, f)
}
```

今回は```ast.Inspect```で探索して得たノードをそのまま書き換えています。（実は [golang.org/x/tools/go/ast/astutil](https://godoc.org/golang.org/x/tools/go/ast/astutil) パッケージにはAST書き換えの便利な機能が提供されていますが、今回は使わずに書き換えてみます。）

そして今回は ```go/printer``` パッケージを使っています。こちらは AST からコードを生成する便利な関数が提供されている。今回使ったのは二つ。

[go/printer.Config](https://godoc.org/go/printer#Config)

```go
type Config struct {
	Mode     Mode // default: 0
	Tabwidth int  // default: 8
	Indent   int  // default: 0 (all code is indented at least by this much)
}
```

こちらでprinterの出力時の設定が出来る。そして実際の出力は下のメソッドでイケる。

[go/printer.Fprint](https://godoc.org/go/printer#Config.Fprint)

```go
func (cfg *Config) Fprint(output io.Writer, fset *token.FileSet, node interface{}) error
```

output に io.Writer を実装しているもの (今回は```os.File```を渡している) を渡す。```fset``` は最初に作った ```token.FileSet```を渡している。そして、```node``` には実際に書き換えたASTを渡している。

これを実行すると```example/example.go```の関数名"add"を"plus"に書き換えて、新しいファイル```example.result.go```を作成します。

```go
package example

import "log"

func plus(n, m int) {
	log.Println(n + m)
}
```

これで AST を書き換えて、ASTからGo言語のソースコードを出力する一連の流れができました！

## まとめ

golang/go には今回使ったパッケージ以外にも 様々なパッケージやインターフェース、構造体があります。興味持った方で、もっと詳しく知りたいという方は下記の記事が参考になるでしょう。

[goパッケージで簡単に静的解析して世界を広げよう #golang](https://qiita.com/tenntenn/items/868704380455c5090d4)
静的解析を学ぶ際にどのような時にどのような記事を読めば良いのかをまとめてくれています！

[https://qiita.com/tenntenn/items/beea3bd019ba92b4d62a](https://qiita.com/tenntenn/items/beea3bd019ba92b4d62a)
こちらは AST の解析時に型をチェックしたりする方法を紹介している。今回のハンズオンでは紹介してないですが、```go/types``` は構文解析では結構使います。

typesパッケージ使った構文解析も余力あれば書きたい。

### 参考 Nodeの構成

```
Node
  Decl
    *BadDecl
    *FuncDecl
    *GenDecl
  Expr
    *ArrayType
    *BadExpr
    *BasicLit
    *BinaryExpr
    *CallExpr
    *ChanType
    *CompositeLit
    *Ellipsis
    *FuncLit
    *FuncType
    *Ident
    *IndexExpr
    *InterfaceType
    *KeyValueExpr
    *MapType
    *ParenExpr
    *SelectorExpr
    *SliceExpr
    *StarExpr
    *StructType
    *TypeAssertExpr
    *UnaryExpr
  Spec
    *ImportSpec
    *TypeSpec
    *ValueSpec
  Stmt
    *AssignStmt
    *BadStmt
    *BlockStmt
    *BranchStmt
    *CaseClause
    *CommClause
    *DeclStmt
    *DeferStmt
    *EmptyStmt
    *ExprStmt
    *ForStmt
    *GoStmt
    *IfStmt
    *IncDecStmt
    *LabeledStmt
    *RangeStmt
    *ReturnStmt
    *SelectStmt
    *SendStmt
    *SwitchStmt
    *TypeSwitchStmt
  *Comment
  *CommentGroup
  *Field
  *FieldList
  *File
  *Package
```

