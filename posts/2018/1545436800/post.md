# Go 1.12 beta 1 の標準パッケージの中で気になったマイナーチェンジを Docker で試したので紹介する。

Gopher道場 Advent Calendar 2018 の 22日目のエントリです。
昨日は [hiokidaichi](https://qiita.com/hiokidaichi) さんの [Mac の Go で birth time を取得する](https://qiita.com/hiokidaichi/items/26890e7da566cb3a4121) でした！！

## 追記 2019/2/26
*[go1.12](https://golang.org/doc/go1.12)がリリースされました！！*


## はじめに

きたる 2019/2 に Go 1.12 がリリースされる予定です。一方で 2018/12 現在、Go 1.12 の beta版の機能を試すことができます。Docker環境立てて、Go 1.12 の中で気になった機能を試して見たので紹介します。せっかくなので Go 1.12 の Docker環境の立て方も紹介します。ここで紹介する機能は現在WIPなので注意！！

## Go 1.12 beta 1 の Docker 環境を準備

もっと最適化はできると思いますが、下の Dockerfile を準備したら終わり。

```Dockerfile
FROM centos:centos7

WORKDIR /home/go
ENV PATH $PATH:/usr/local/bin/go/bin
RUN curl -O https://storage.googleapis.com/golang/go1.12beta1.linux-amd64.tar.gz \
    && tar -C /usr/local/bin -xzf go1.12beta1.linux-amd64.tar.gz \
    && rm go1.12beta1.linux-amd64.tar.gz \
    && yum -y update && yum clean all \
    && yum install -y vim && yum clean all
```

```go1.12beta1.linux-amd64.tar.gz``` をダウンロード & 解凍して ```go``` にパスを通して エディターの ```vim``` を追加しています。これを走らせましょう。

```bash
$ docker build ./ -t go1.12
$ docker run -it --name go1.12 go1.12 /bin/bash
```

docker run でコンテナに入ったら確認です。

```bash
$ go version
go1.12beta1 linux/amd64
```

go1.12beta1 が入りました。

## bytes.ReplaceAll

bytes パッケージの新しい関数。

```go
func ReplaceAll(s, old, new []byte) []byte
```

```bytes.ReplaceAll```は、```s``` の ```old``` を ```new``` に置き換えたバイトスライスのコピーを返す。

```go
package main // import "try-go112"

import (
    "fmt"
    "bytes"
)

func main(){
    // qux qux bar bar baz baz
    fmt.Printf("%s\n", bytes.ReplaceAll([]byte("foo foo bar bar baz baz"), []byte("foo"), []byte("qux")))
}
```

## strings.ReplaceAll

先ほどの ```bytes.ReplaceAll``` の string バージョンも追加される。

```go
func ReplaceAll(s, old, new string) string
```

使用例はこちら

```go
package main // import "try-go112"

import (
        "fmt"
        "strings"
)

func main(){
        // moo moo moo
        fmt.Println(strings.ReplaceAll("oink oink oink", "oink", "moo"))
}
```

## expvar.Delete

expvar パッケージは メトリクスを簡単に所得できる便利な標準パッケージ。expvar パッケージに関しては下記の記事が参考になりそう。
https://qiita.com/methane/items/8f56f663d6da4dee9f64

そして、Go 1.12 からの新しい関数が追加。メトリクスのマップからキーを指定して削除できます。

```go
func (v *Map) Delete(key string)
```

このパッケージは使ったことないのですが、そもそも今まで削除どうやってたんだろう。下が使用例。

```go
package main // import "try-go112"

import (
    "expvar"
    "fmt"
)

func main() {
    Map := expvar.NewMap("myapp")

    Conns := new(expvar.Int)
    MessageRecv := new(expvar.Int)
    MessageSent := new(expvar.Int)

    Map.Set("conns", Conns)
    Map.Set("msg_recv", MessageRecv)
    Map.Set("msg_sent", MessageSent)

    Map.Delete("msg_recv")
    str := Map.String()

    // {"conns": 0, "msg_sent": 0}
    fmt.Println(str)
}
```

## fmt

map を print する際に key でソートされて出力されるようになる。

```go
package main // import "try-go112"

import "fmt"

func main() {
    m := map[interface{}]interface{}{
        "go":  "golang",
        "rb":  "ruby",
        "js":  "javascript",
    }
    // map[go:golang js:javascript rb:ruby]
    fmt.Println(m)
}
```

ソートのルールは https://tip.golang.org/doc/go1.12#fmt で確認できる。

## io.StringWritter

新しいインターフェースが公開される。今まで ```io.stringWriter``` インターフェースがありましたが(実装は全く同じ)、このインターフェースは外部にエクスポートされていませんでした。しかし、Go 1.12 から 外から使える ```io.WriteStringer``` インターフェースが追加されています。

```go
type StringWriter interface {
    WriteString(s string) (n int, err error)
}
```

```go
package main // import "try-go112"

import (
        "fmt"
        "io"
)


type User struct {
        Name string
}

func (u *User) WriteString(s string) (n int, err error) {
        // 実装は適当
        fmt.Printf("to %v, %v \n", u.Name, s)
        return 1, nil
}

func main() {
        u := &User{
                Name: "po3rin",
        }
        checkStringWriter(u)
}

func checkStringWriter(v interface{}) {
        // io.StringWriterの実装を確認。
        if _, ok := v.(io.StringWriter); ok {
                fmt.Println("impliments StringWriter")
        }
}
```

## os.ProcessState.ExitCode

os.ProcessState から  新しいメソッドが生える。

```go
func (p *ProcessState) ExitCode() int
```

終了したプロセスの終了コードを返してくれる。プロセスが終了していないかシグナルによって終了した場合は-1を返します。

```go
package main // import "try-go112"

import (
        "fmt"
        "os"
        "os/exec"
)

func main() {
        if len(os.Args) == 1 {
                return
        }
        cmd := exec.Command(os.Args[1], os.Args[2:]...)
        cmd.Run()
        state := cmd.ProcessState
        fmt.Println(state.ExitCode())
}
```

## reflect.MapIter

新しく ```reflect.MapIter``` が実装される。

```go
type MapIter struct {
    // contains filtered or unexported fields
}
```

MapIter は下記のメソッドで初期化する。

```go
func (v Value) MapRange() *MapIter
```

MapRangeは、```reflect.Value```に新しく実装されるメソッドで Map のイテレータ ```MapIter``` を返します。```v```の種類が Map でない場合はパニックになります。

```reflect.MapIter``` には3つのメソッドが生えている。

```go
func (it *MapIter) Next() bool
```

マップのイテレータを進めていきます。イテレータが終わるとfalseを返します。

```go
func (it *MapIter) Key() Value
```

イテレータの現在のマップのエントリのキーを返します。

```go
func (it *MapIter) Value() Value
```

イテレータの現在のマップのエントリの値を返します。

使用例は下記。マップに対する反復処理ができます。

```go
package main // import "try-go112"

import (
    "fmt"
    "reflect"
)

func main() {
    m := map[string]interface{}{
        "name":"po3rin",
        "age": 27,
        "live": "tokyo",
    }
    iter := reflect.ValueOf(m).MapRange()
    for iter.Next() {
        k := iter.Key()
        v := iter.Value()

        fmt.Printf("key: %v, val: %v \n", k, v)
    }
}
```

「for文で回すのと何が違うの？」となるが、```*MapIter.Value()``` と ```*MapIter.Key()``` は ```reflect.Value```インターフェースとして返してくれるので、そのまま ```reflect.Value``` に対しての処理ができる。

## strings.Builder.Cap

この前 https://qiita.com/po3rin/items/2e406645e0b64e0339d3 で紹介した```strings.Builder``` ですが、そこに新しくメソッドが追加されます。

```go
func (b *Builder) Cap() int
```

単に今の ```Builder``` のバイトスライスの容量を返します。使うと下のようになります。

```go
package main // import "try-go112"

import (
        "fmt"
        "strings"
)

func main(){
        var b strings.Builder
        b.Grow(10)

        // 10
        fmt.Println(b.Cap())
}
```

## その他、気になった変更点

### Binary-only packages のサポートが Go 1.12 で最後

> Binary-only packages
> Go 1.12 is the last release that will support binary-only packages.

Binary-only packages のサポートが Go.1.12 で最後になるよう。Binary-only packages に関しては僕も過去に記事にしていた。

[Go言語 で ソースコードを含めない「Binary-Only Package」を作ってみる！](https://qiita.com/po3rin/items/dff8ae7f4f7f187094d8)


###  -benchtime flag

テストのくり返し回数の設定をサポートするようになります。たとえば、-benchtime=100x はベンチマークを100回実行します。

```go
go test -bench . -benchtime=100x
```

### crypto/tls が TLS 1.3 をサポート

```crypto/tls```パッケージに RFC8446 で明示されている TLS 1.3 のサポートが追加されます。Configに明示的にMaxVersionを設定していなければ、利用可能な場合、自動的に TLS 1.3 を使用するようになる。

### まとめ

楽しみ Go1.12 ！！ もし間違った情報があったら教えてください。

この他にもたくさんの変更が入る予定なので是非リリースノートを確認してみてください。
[Go 1.12 Release Notes](https://tip.golang.org/doc/go1.12)

