---
title: Rust初心者が楽して作るHTTPクライアントCLI (surf & clap)
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/rust-http-client.jpeg
date: 2020/12/19
id: rust-http-client
description: 実務で使うツールをRustでサラッと実装したので、僕が踏んだ実装方法を紹介します。
tags:
    - Rust
---

## Overview

こんにちは [pon](https://twitter.com/po3rin) です。これは[Rust Advent Calendar 2020](https://qiita.com/advent-calendar/2020/rust) の記事です。

初心者がHTTPクライアントCLIをRustで書いて、実務で利用したので、実装方法を紹介します(ほとんどライブラリの紹介になる気がするが...)。Rustで何か作ってみたい人の足がかりになると思います。

## 作ったやつ

社内のAPIを叩くので実際のコードは公開できませんが、どんな感じのツールかを共有します。テキストからキーワード一覧を取得して、そのキーワードごとに検索エンジンが何件返すかを調べる簡単なツールです。

これを作るのに使ったライブラリを紹介します。これらを使うとRustでも簡単にHTTPクライアントCLIを作成できます。

## HTTPクライアント: surf

[surf](https://github.com/http-rs/surf)はHTTPクライアントライブラリです。async/await に対応した HTTP クライアントです。非同期処理ランタイムとしてはasync_std を利用しています。

非同期処理ランタイムごとに素晴らしいまとめがあるこちらの記事がおすすめです。
[2019 年の非同期 Rust の動向調査](https://qiita.com/legokichi/items/53536fcf247143a4721c#)

例えばGETリクエストをしたい場合は下記のように書きます。(これは日本語でsurfを紹介してくれている[@helloyuki_](https://twitter.com/helloyuki_)さんの[記事](https://blog-dry.com/entry/2020/02/29/152325)の引用です。)

```rust
extern crate surf;

#[async_std::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync + 'static>> {
    let mut res = surf::get("https://httpbin.org/get").await?;
    dbg!(res.body_string().await?);
    Ok(())
}
```

これだけでhttpbinに対してGETリクエストが放てます。今回はレスポンスのJSONを構造体にparse処理して結果を集計したかったので```recv_json()```を利用します。構造体には Serialize/Deserialize traitを実装するためのマクロを書いておきます。

```rust
#[macro_use]
extern crate serde_derive;
extern crate serde;
extern crate serde_json;

#[derive(Serialize, Deserialize)]
struct Res {
    total: u32
}
```

そうすると```recv_json()```でレスポンスが構造体で受けれるようになります。今回は検索ヒット数だけが欲しのでクエリに対してヒット数をprintしてみます。

```rust
let Res {total} = surf::get("https://example.com").recv_json().await?;

println!("{},{}", total, s);
```

検索クエリをセットしたい場合は```set_query```を使います。Query構造体を用意してSerialize/Deserialize traitを実装させてそれを利用するだけです。

```rust
#[derive(Serialize, Deserialize)]
struct Query {
    query: String
}

let query = Query { query: "コロナ" };
let Res {total} = surf::get("https://example.com")
    .set_query(&query)?
    .recv_json().await?;

println!("{},{}", total, s);
```

また、今回はBasic認証のかかったテスト用APIへのコールだったのでHTTP Headerをつける必要があります。Headerは```.set_header()```です。なおBaisc認証ではBase64エンコードした文字列を渡すので```base64::encode```を利用します。

```rust
extern crate base64;
use base64::encode;

// ...

let b = format!("{}:{}", u, p);
let e = format!("Basic {}", encode(b)); //Base64 エンコード

let query = Query { query: s.to_string() };
let Res {total} = surf::get("https://example.com")
    .set_header("Authorization", &e)
    .set_query(&query)?
    .recv_json().await?;

println!("{},{}", total, s);
```

これで目的のBasic認証込みの、APIクライアントができました。これをCLIとして利用できるようにします。

## CLIライブラリ: clap

[clap](https://github.com/clap-rs/clap)はRustのためのコマンドラインパーサーです。

いろんな書き方ができますが僕は下記のBuilder patternを使った設定記述が好きです。
[clap using Builder Pattern](https://github.com/clap-rs/clap#using-builder-pattern)


例えば今回の用途で言うとこんな感じになります。

```rust
extern crate clap;
use clap::{App, Arg};

// ...

let app = App::new("rust-http-cli")
    .version("0.1.0")  
    .author("po3rin")     
    .about("get search his")
    .arg(Arg::new("user")
        .about("basic auth user")
        .short('u')          
        .long("user")         
        .takes_value(true) // 値をとるフラグかどうか
    )
    .arg(Arg::new("pass")
        .about("basic auth pass")
        .short('p')            
        .long("pass")           
        .takes_value(true)   
    )
    .arg(Arg::new("file")
            .about("input file")
            .short('f')           
            .long("file")        
            .takes_value(true) 
            .required(true) //　必須
    );

// flagで渡された値の取得
let matches = app.get_matches();
let u = matches.value_of("user").unwrap();
let p = matches.value_of("pass").unwrap();
let f = matches.value_of("file").unwrap();
```

こんな感じに記述してビルドしておくと```-h```でヘルプが出ます。

```bash
./target/debug/rust-http-cli -h   
rust-http-cli 0.1.0
po3rin
get search his

USAGE:
    askdhit [OPTIONS] --file <file>

FLAGS:
    -h, --help       Prints help information
    -V, --version    Prints version information

OPTIONS:
    -f, --file <file>    input csv file
    -p, --pass <pass>    basic auth pass
    -u, --user <user>    basic auth user
```

少しの記述でサクッとCLIの形が完成しました。

今回の僕の用途だとクエリがリストアップされたテキストファイルをフラグで受けてパースしてsurfを使ったリクエストに利用しています。、また、Baisc認証用にフラグで受け取った情報をこの後にエンコードしていきます。

## まとめ

surf + clap でRust初心者でもサクッとHTTPクラアントCLIが作れました。何かRustで小さなツールを実装してみたいと思っている方の参考になればと思います。

## 参考

[Rust の HTTP クライアント surf を試してみる](https://blog-dry.com/entry/2020/02/29/152325)

[clap-rs/clap](https://github.com/clap-rs/clap)

