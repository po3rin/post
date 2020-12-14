---
title: go-plugin × gRPC で自作Goツールにプラグイン機構を実装する方法
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/go-plug.jpeg
date: 2020/12/15
id: go-plug
description: go-pluginパッケージを使ってgRPCプラグイン機構を提供する方法を調べたので紹介します。
tags:
    - Go
---

## Overview

こんにちは [pon](https://twitter.com/po3rin) です。これは[Go 2 Advent Calendar 2020 ](https://qiita.com/advent-calendar/2020/go2) の15日目の記事です。

自作ツールに素敵なプラグイン機構を仕込みたいことありますよね。今回はTerraformやPackerなどでプラグイン機構をして利用されているパッケージである```hashicorp/go-plugin```の使い方を紹介し、実際にどのように実装するかをコードをあげて紹介します。

## go-pluginとは

go-pluginは、RPCを介したGoのプラグインシステムです。

[![go-plugin](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/go-plugin.png)](https://github.com/hashicorp/go-plugin)

TerraformなどHashicorpの様々なOSSの内部でも使われているので実績は抜群です。RPCを介すると言いますがローカルでの接続しかサポートしていませんが、gRPCベースのプラグインを使用すると、プラグインを任意の言語で作成できます。内部的にはプラグインをfork-execし、やりとりをRPCで行います。下記はgRPCを使った例です。

![go-plugin architecture](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/go-plug-archi.png)

go-pluginパッケージはfork-exec、negotiation info、health checkのやりとりはgo-pluginが内部でやってくれるので、やるべきことはProtocol Buffersを定義し、コード生成し、インターフェースを実装してgo-pluginに渡してあげるだけです。

gRPCが利用できるとGo以外の言語もサポートできますが、go-pluginパッケージが使えないので、negotiation infoの出力や、health checkを自前で実装する必要があります。

## go-pluginを使ってみる

今回はプラグイン機構をもつ単純なCLIを作成します。サポートするプラグインはGreeterプラグインという、渡された名前を使って挨拶の文字列を返すプラグインを実装します。

まずはプラグインのインターフェースをProtocol Bufferで定義します.
```proto/greeter/greeter.proto```を作成しましょう。

```go
syntax = "proto3";

option go_package = "github.com/po3rin/helloplug/proto/greeter";

package greeter;

// The greeting service definition.
service Greeter {
  // Sends a greeting
  rpc Say(Request) returns (Reply) {}
}

// The request message containing the user's name.
message Request { string name = 1; }

// The response message containing the greetings
message Reply { string message = 1; }
```

そしてGoコードを生成します。必要なツールは下記の公式にしたがってインストールします。
[gRPC Go Quick Start #Prerequisites](https://www.grpc.io/docs/languages/go/quickstart/#prerequisites)

```bash
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/greeter/greeter.proto
```

Goのコードが作成されているのを確認します。

```bash
.
└── proto
    └── greeter
        ├── greeter.pb.go
        ├── greeter.proto
        └── greeter_grpc.pb.go
```

続いて生成されたインターフェースを満たす実装を定義します。準備するべきは```greeter.GreeterServer```の実態です。```greeter.GreeterServer```は下記のように```greeter.pb.go```に生成されているはずです。

```go
type GreeterServer interface {
	// Sends a greeting
	Say(context.Context, *Request) (*Reply, error)
	mustEmbedUnimplementedGreeterServer()
}
```

それでは```greeter.GreeterServer```の実装のために```plug/plug.go```を作成しましょう。

```go
package plug

import (
	"context"

	"github.com/po3rin/helloplug/proto/greeter"
)

type Greeter interface {
	Say(name string) (string, error)
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl Greeter
	greeter.UnimplementedGreeterServer
}

func (m *GRPCServer) Say(ctx context.Context, r *greeter.Request) (*greeter.Reply, error) {
	msg, err := m.Impl.Say(r.Name)
	if err != nil {
		return nil, err
	}
	return &greeter.Reply{Message: msg}, nil
}
```

```GRPCServer.Impl```はプラグインとして渡される処理を定義するフィールドです。Greeterの実態はプラグインによって渡されます。

続いてプラグインを呼び出す側の実装を```plug/plug.go```に作っていきます。

```go
type GRPCClient struct{ client greeter.GreeterClient }

func (m *GRPCClient) Say(name string) (string, error) {
	r := &greeter.Request{Name: name}
	res, err := m.client.Say(context.Background(), r)
	if err != nil {
		return "", err
	}
	return res.Message, nil
}
```

ここではプラグインに定義される処理を呼び出すための```reeter.Request```初期化や、実際の呼び出しを行なっています。

次にこれらの処理をラップしてgo-pluginで使えるようにしていきます。その為にはgo-pluginで定義されている```plugin.Plugin```インターフェースの実装を作っていく必要があります。```plugin.Plugin```は下記の定義を持ちます。

```go
type Plugin interface {
	Server(*MuxBroker) (interface{}, error)
	Client(*MuxBroker, *rpc.Client) (interface{}, error)
}
```

また、gRPCでプラグインを作る場合は下記のインターフェースも実装していきます。

```go
type GRPCPlugin interface {
	GRPCServer(*GRPCBroker, *grpc.Server) error
	GRPCClient(context.Context, *GRPCBroker, *grpc.ClientConn) (interface{}, error)
}
```

それではこれらのインターフェースをみたす```GreeterPlugin```構造体を作っていきます。

```go
type GreeterPlugin struct {
	plugin.Plugin
	Impl Greeter
}

func (p *GreeterPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	greeter.RegisterGreeterServer(s, &GRPCServer{Impl: p.Impl}) // TODO: impl
	return nil
}

func (p *GreeterPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: greeter.NewGreeterClient(c)}, nil
}
```

```GreeterPlugin.GRPCServer```メソッドではgRPCサーバーの登録を行い、```GreeterPlugin.GRPCClient```ではプラグインの処理をコールする実装を返してあげます。

これであとは、プラグイン初期化に必要な二つの変数を作るのみです。

```go
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "GREETER_PLUGIN",
	MagicCookieValue: "greeter",
}

var PluginMap = map[string]plugin.Plugin{
	"greeter": &GreeterPlugin{},
}

```

```Handshake```は クライアントとサーバーでのハンドシェイクの設定、 ```PluginMap```はサポートするプラグインの実装をマップとして保持するmapです。これでCLIにプラグイン機構を実装する準備ができました。実際にCLIを作っていきます。まずは```helloplug.go```を作ります(エラーは省略)。

```go
package helloplug

import (
    // ...

	"github.com/hashicorp/go-plugin"
	"github.com/po3rin/helloplug/plug"
)

func Run() {
    pluginName := os.Getenv("GREETER_PLUGIN")
	if pluginName == "" {
		fmt.Println("no plugin")
		return
	}

	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(
		&plugin.ClientConfig{
			HandshakeConfig: plug.Handshake,
			Plugins:         plug.PluginMap,
			Cmd:             exec.Command("sh", "-c", os.Getenv("GREETER_PLUGIN")),
			AllowedProtocols: []plugin.Protocol{
				plugin.ProtocolGRPC,
			},
			Logger: hclog.New(&hclog.LoggerOptions{
				Output: hclog.DefaultOutput,
				Level:  hclog.Error, // デフォルトで hclog.Trace
				Name:   "plugin",
			}),
		},
	)
	defer client.Kill()

	rpcClient, _ := client.Client()
	raw, _ := rpcClient.Dispense("greeter")
	say, _ := raw.(plug.Greeter)

	msg, _ := say.Say("gopher")
	fmt.Println(msg)
}
```

```plugin.NewClient```でプラグインを呼び出すためのクライアントを初期化します。````plugin.ClientConfig```に各設定を渡していきます。```Cmd```プラグインのバイナリを実行してgRPC接続できるようにします。

ここまで実装すれば、サーバー側の起動、クライアントサーバー間のハンドシェイク(ヘルスチェック、ハンドシェイク情報のやりとりなど)はgo-plug内で行なってくれます。

環境変数で利用するプラグインが指定されていない場合はプラグイン利用せずに終了します。

あとは```cmd/helloplug/main.go```を作ってあげます。

```go
package main

import "github.com/po3rin/helloplug"

func main() {
	helloplug.Run()
}
```

これでプラグイン機構つきCLIが完成しました。まだプラグインを作成していませんので呼び出すとプラグインを利用せずそのまま処理が終了します。

```bash
$ go build ./cmd/helloplug/main.go
$ ./main
no plugin
```

## Goでプラグインを作成する

ではGoでプラグインを作ってみましょう。```plugins/hello/main.go```を作成します。こちらでは先ほど定義した```plug.Greeter```インターフェースの実装を作成し、go-plugに渡してあげるだけです。

```go
package main

import (
	// ...

	"github.com/hashicorp/go-plugin"
	"github.com/po3rin/helloplug/plug"
)

type Hello struct{}

func (h *Hello) Say(name string) (string, error) {
	return fmt.Sprintf("hello %s", name), nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plug.Handshake,
		Plugins: map[string]plugin.Plugin{
			"greeter": &plug.GreeterPlugin{Impl: &Hello{}},
		},

		GRPCServer: plugin.DefaultGRPCServer,
	})
}
```

これでプラグインは完成です。簡単ですね。早速プラグインを利用してみます。

```bash
$ go build -o hello-plugin ./plugins/hello/main.go

$ export GREETER_PLUGIN="./hello-plugin" # 環境変数で利用するプラグインを指定
$ ./main
hello gopher
```

プラグインをgRPC経由で利用できました。gRPCを使ったプラグイン機構なのでもちろん他の言語でもプラグインを実装可能です。

## まとめ

今回はgo-plugin＋gRPCでプラグイン機構を提供する方法を紹介しました。自分もプラグイン機構を提供したい自作ツールが数点あるので、実装してみたいと思います。

## 参考

Exampleが非常に参考になります。
[hashicorp/go-plugin](https://github.com/hashicorp/go-plugin)

