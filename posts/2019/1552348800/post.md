# Docker に正式統合された BuildKit の buildctl コマンドで Dockerfileを使わずにコンテナイメージをビルドするハンズオン

こんにちはpo3rinです。日本語解説があまりなかったので buildctlコマンドをセットアップを行い、 docker build を使わずに コンテナイメージをビルドする過程を紹介します。OS は Mac OSX を想定してます。

## BuildKitとは

BuildKitは、命令からイメージレイヤを作成するための操作を行うツールキットです。Buildkit === 次世代 docker build という表現で説明されることも多いですが、Buildkit 自体は Docker とは別物です。そもそも Docker は moby という OSS から作られており、その中の moby/buildkit がイメージレイヤを作成する責務を持ちます。ゆえに、Buildkit は次世代の機能を持ったビルドツールキットと言った感じで、docker build はそれを単に利用していると言った感じです。

## BuildKit の主要コンポーネント

BuildKitプロジェクト自体は、2つの主要コンポーネントで構成されています。ビルドデーモンの buildkitd と buildctl という CLI ツールです。buildctl は buildkitd とのコマンドライン通信を簡単に行う為に使われます。

## buildctl の環境構築

ビルドデーモンの buildkitd と buildctl という CLI ツールを使えるようにしましょう。

前提条件は下記になります。jqコマンドは、JSON データを加工するためのコマンドです。なくても大丈夫ですがmoby/buildkitのREADME.mdでも使っているほど便利なので今回はこいつを使いましょう。

```console
$ go version
go version go1.12 darwin/amd64

$ docker version
Client: Docker Engine - Community
 Version:           18.09.2
 API version:       1.39
 Go version:        go1.10.8
 Git commit:        6247962
 Built:             Sun Feb 10 04:12:39 2019
 OS/Arch:           darwin/amd64
 Experimental:      false

Server: Docker Engine - Community
 Engine:
  Version:          18.09.2
  API version:      1.39 (minimum version 1.12)
  Go version:       go1.10.6
  Git commit:       6247962
  Built:            Sun Feb 10 04:13:06 2019
  OS/Arch:          linux/amd64
  Experimental:     true

$ make --version
GNU Make 3.81
Copyright (C) 2006  Free Software Foundation, Inc.
This is free software; see the source for copying conditions.
There is NO warranty; not even for MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE.

This program built for i386-apple-darwin11.3.0

$ jq --version
jq-1.6
```

### buildctl の準備

まずは buildctl コマンドをインストールしましょう。README.mdの方法だと、「このOSでは実行できない」と言われてしまったので、https://github.com/moby/buildkit/releases から直接OSにあうバイナリをダウンロードしてくるのをおすすめします。そして、バイナリをPATHの通っているディレクトリ(```/usr/local/bin```など)に配置すれば完了です。

### buildkitd の準備

BuildKit は Dockerコンテナ内で buildkitd デーモンを実行し、それにリモートでアクセスすることによっても使用できます。[moby/buildkit](https://hub.docker.com/r/moby/buildkit)でDockerイメージが配布されています。下記のコマンドを実行すれば準備は完了です。

```console
$ docker run --name buildkit -d --privileged -p 1234:1234 moby/buildkit --addr tcp://0.0.0.0:1234

$ export BUILDKIT_HOST=tcp://0.0.0.0:1234

$ buildctl build --help
```

これで準備ができました。

## 何はともあれ BuildKit を触ってみる

早速実行できる例が [moby/buildkit/example](https://github.com/moby/buildkit/tree/master/examples) にあります。examples/buildkit0/buildkit.go は サンプルの LLB(後で説明する) を標準出力に渡してくれます。これらをクローンして実行してみましょう。

```console
$ go run examples/buildkit0/buildkit.go | buildctl debug dump-llb | jq '.'
```

下記は ```dump-llb``` からのJSON出力の抜粋です

```json
// ...
{
  "Op": {
    "inputs": [
      {
        "digest": "sha256:50cd8db1b5d8630e4fef4b359c215f2938d5052391a39e5126fc90174e694ceb",
        "index": 0
      }
    ],
    "Op": {
      "Exec": {
        "meta": {
          "args": [
            "git",
            "checkout",
            "-q",
            "6635b4f0c6af3810594d2770f662f34ddc15b40d"
          ],
          "env": [
            "PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
            "GOPATH=/go"
          ],
          "cwd": "/go/src/github.com/opencontainers/runc"
        },
        "mounts": [
          {
            "input": 0,
            "dest": "/",
            "output": 0
          }
        ]
      }
    },
    "platform": {
      "Architecture": "amd64",
      "OS": "linux"
    },
    "constraints": {}
  }
}
// ...
```

これは何でしょうか。このJSONはDAG構造を備える中間言語であるLLBを表現しています。

## LLB とは

<img src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2019/1552348800/qiita-deb798ed9c1edac5cc4b-1.png" width="480px">

BuildKitは、LLB というプロセスの依存関係グラフを定義するために使用されるバイナリ中間言語を利用して。イメージをビルドしています。

なぜこの中間言語を挟むかというと、LLBはDAG構造(上の画像のような非循環構造)を取ることにより、ステージごとの依存を解決し、並列実行可能な形で記述可能だからです。これにより、BuildKitを使ったdocker build は並列実行を可能にしています。

Dockerfileのビルドを例に見てみましょう。DockerfileをASTに変換した後、ASTからstageごとに解析した構造を生成し、そこからLLBに変換します。

<img src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2019/1552348800/qiita-deb798ed9c1edac5cc4b-2.png" width="300px"></img>

LLBを使うことのメリットは並列実行以外にも、効率的にキャッシュが実現できたり、ベンダーに依存しない（つまり、Dockerfile以外の言語を簡単に実装できる）ということが挙げられます。

それでは実際にビルド処理をみてみましょう。

```bash
go run examples/buildkit0/buildkit.go | buildctl build
```

buildctl build を実行すると並列でビルドされているのがわかります。ビルド結果と中間キャッシュはBuildKitの内部にのみ残ります。

```console
$ buildctl du
ID									RECLAIMABLE	SIZE		LAST ACCESSED
sha256:07e99186a7474c586eab459d6375cc17298bad5f725f2205688e0e4c907d99f6	true       	471.44MB

// ...
```

これらのレイヤーをDockerイメージとして具体的なものにするには、ビルド呼び出しで--exporterフラグを使用します。ビルドキャッシュのおかげで、ビルドが非常に速いのが確認できるはずです。

```bash
go run examples/buildkit0/buildkit.go | buildctl build --exporter=docker --exporter-opt name=buildkit0 | docker load
```

Dockerイメージとしてコンテナイメージが作成されたことがわかります。

```console
$ docker image ls
REPOSITORY   TAG     IMAGE ID      CREATED        SIZE
buildkit0    latest  8e063c66117d  2 hours ago    129MB
```

これでコンテナイメージ生成の一通りの流れがbuildctlを使って行うことができました。exampleのコードで何をしているかは暇なときに記事にします。

## 参考

buildkit のREADME.mdが一番具体的
[moby/buildkit](https://github.com/moby/buildkit)

buildctl の入門資料
[Getting Started With BuildKit](https://george.macro.re/posts/getting-started-with-buildkit/)


## その他の関連記事

僕が書いたBuildkit周辺の解説記事もどうぞ

[Goを読んでDockerの抽象構文木の構造をサクッと理解する](https://qiita.com/po3rin/items/a3934f47b5e390acfdfd)

[Buildkit の Goのコードを読んで Dockerfile 抽象構文木から LLB を生成するフローを覗いてみよう！！](https://qiita.com/po3rin/items/f414660bd2a6173c587a)

