---
title: kubernetes/git-sync でブログのPull型データ同期を構築した
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/blog-git-sync-cover.jpeg
date: 2020/08/07
id: blog-git-sync
description: ブログをPush型からPull型のデータ同期に移行したので、そのアーキテクチャを共有します。
tags:
    - Go
    - Kubernetes
---

## Overview

こんにちは [pon](https://twitter.com/po3rin) です。このテックブログはk8s上で運用しているのですが、今回、ブログの公開フローを自動化したのでそのお話をします。今回のお話はブログのデータ同期ですが ***「Gitリポジトリの何かしらのデータを使って、何かの処理を自動化したい」*** という抽象度を持つ課題であれば、今回お話するアーキテクチャが利用可能です。

## Before: 改善前の課題

まずは公開フローを自動化する前の構成をみてください。

![改修前アーキテクチャ](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/sync-before.png)

Elasticsearchに記事データを保存する為に、ローカルで書いたMarkdownをAPI経由で送信しています。APIの口は公開していないのでローカルで ```kubectl port-forward``` を叩いてから、独自実装のCLIである```postctl```で送信していました。これだと下記のような課題がありました。

* ローカルからkubectlなど公開手順が手間
* Gitリポジトリと実際の記事の状態が乖離する

今っぽく無いですね。***本来は記事データをGitHubにpushしたらそのままElasticsearchと同期するのが理想*** です。これができれば、Kubernetesを知らない人でもブログを公開でき、より使いやすいアーキテクチャになりそうです。

## After: git-sync を使ったPull型データ同期

今回は git-sync を使ったサイドカーパターンでこれを実現しました。こんな感じになってます。

![改修後アーキテクチャ](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/sync-after.png)

リポジトリにPushしたら、リポジトリを監視している```sync-agent```が差分を取得し、inserter(記事データの簡単な処理を行うやつ)に記事データを流します。では sync-agent の仕組みを紹介していきます。

### リポジトリの同期

リポジトリの状態の監視と、記事データの同期を担当する```sync-agent```は ```kubernetes/git-sync``` を使ったサイドカーパターンで構築しています。

![sync-agentアーキテクチャ](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/sync-agent.png)

```kubernetes/git-sync```は、Gitリポジトリをローカルディレクトリに引っ張ってくるシンプルなコマンドです。これはサイドカーとしての利用を前提にして開発されています。今回の利用例ではでリポジトリへの記事データのpushをマウントしたvolume```sync-git```でpullしてきます。図から分かりますが、この方式はPull型のデータ同期なので、***EKSにアクセスする機密情報が一切必要ありません***。

[![git-sync](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/git-sync.png)](https://github.com/kubernetes/git-sync)

```sync-git```はWebHookをサポートしており、新しいリビジョンを取り込んだことをAPI経由で外部に伝えることができます。



```git-sync```を使ったサイドカーパターンのmanifestは下記のようになります。

```yaml
apiVersion: apps/v1
kind: Deployment
# ...
spec:
  # ...
  template:
    # ...
    spec:
      containers:
      - name: sync-agent
        image: "****/sync-agent:****"
        command:
          - postctl
          - sync
          - --agent-mode
          # ...
        volumeMounts:
        - mountPath: /tmp/git
          name: git
          readOnly: true

      - name: git-sync
        image: k8s.gcr.io/git-sync:v3.1.6
        args:
          - --dest
          - repo
          - --webhook-url
          - http://localhost:9300/api/v1/sync
        env:
        - name: GIT_SYNC_REPO
          value: https://github.com/po3rin/post
        volumeMounts:
        - mountPath: /tmp/git
          name: git
      volumes:
      - emptyDir: {}
        name: git
```

```/tmp/git/repo```にリポジトリの最新の状態をマウントしています。```GIT_SYNC_REPO``` 環境変数で同期するリポジトリを指定できます。```--webhook-url```オプションを使うと、最新のpushを外部に伝えることができます。今回はサイドカーパターンなのでホスト名もlocalhostで解決できます。

### 差分データ同期

WebHookを受け取った独自実装のエージェントで```git diff```して、差分のある Markdown ファイルだけを集約し、記事データ投入を行います。この実装は今までローカルから記事投入をしていた```postctl```コマンドにエージェントモードを追加することで対応しました。内部ではWebHookだけでなく内部で独自の調整ループも行なっています。

Gitリポジトリをマウントしているので、差分ファイルは下記のように取得できます。

```go
// ...
cmd := exec.Command("git", "diff", "--name-only", oldRevision)
cmd.Dir = root
out, err := cmd.CombinedOutput()
files := strings.Split(string(out), "\n") // 差分ファイルたち
// ...
```

上のコード例だと差分ファイルを全てとってくるので、ブログ記事のMarkdownかを判断する関数が必要です。

```go
func isPostMdfile(path string) bool {
	e := filepath.Ext(path)
	if e != ".md" {
		return false
	}
	b := filepath.Base(path)
	if b == "README.md" {
		return false
	}
	return true
}
```

新規の記事ファイルのパースにはMarkdown Parserの```github.com/yuin/goldmark```を使っています。タグなんかもparseできて便利です。

[![goldmark](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/goldmark.png)](https://github.com/yuin/goldmark)

あとはデータをAPI経由で流すだけです。データを同期したら現時点での最新のrevisonを次の差分取得の為にメモリ上で保存しています。ガッチリ実装するなら永続化した方がいいかもしれません。今回はブログデータなので数も少なく、すぐにmasterとゼロから同期できるのでメモリで十分です。

ちなみに今回の実装ではエージェントは起動時に全てのデータ同期します。これは起動時に強制的に同期させ、最新のリビジョンを確保するためです。これを行う為にデータの送り先で冪等性を担保しています。

また、このagentのDockerfileですが、Go経由で```git```コマンドを叩くので、gitインストール済みのベースイメージである。```alpine/git```を使っています。

```Dockerfile
FROM golang:1.14.3 as builder

WORKDIR /src

COPY go.mod /src/go.mod
COPY go.sum /src/go.sum

RUN go mod download

# Perform the build
COPY ./cmd/postctl ./cmd/postctl
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /dist/postctl ./cmd/postctl


FROM alpine/git:v2.24.3
COPY --from=builder /dist/postctl /usr/local/bin/postctl
```

## Push型 vs Pull型

これは廃案ですが、CIからデータを送信するのも検討しました。ブログのリポジトリはGithub ActionsでCIを回していたので、そのまま記事公開もやれば良さそうです。

![sync-ciアーキテクチャ](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/ci-sync.png)

```git-sync```を使ったサイドカーと違い、***Push型のデータ同期*** になります。この方法は一見良さそうですがGitOpsな観点ではPull型がよく使われます。もちろんArgoCDなどのもPull型の同期を行なっています。ではなぜ今回Push型を採用したのかをお話します。

### 強い権限を持つCI問題

Push型では ***CIからkubectlするためのcredentialをGitHubに渡す必要があります***。Inserterのエンドポイントを認証でアクセスできるようにしておくのも手間かつ、それでもGitub Actionsにアイパスを渡す必要があります。やはりCIの権限は最小限にしておきたいところです。

### 自動データ同期

***Pull型なら自動的かつすぐに元の状態に復元できます***。なぜならPush型では意図的にこちらがフックしないとデータ同期が発動しないからです。僕のテックブログではEKSのコストを抑える為に、ギリギリのリソース & テスト環境なし & 永続化もスポットインスタンスで運用しているので、データを失う危険が常にある為、安全面からもPull型が理想です。今は差分ベースのみですが、時間があれば、Elasticsearchの状態を監視し、不健康なら全データ同期を行う実装にする予定です。(全データ同期はブログデータという少ない記事量だからできることかも)

## 余談) GitOps Engine

GitOps を提供するツールはどれも似たようなコア機能を持っていました。そこで、GitOps を実現するコア機能を実装した再利用可能なライブラリをArgoCD と Flux CD のチームが作りました。それが ***GitOps Engine*** です。

[![goldmark](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/gitops-engine.png)](https://github.com/argoproj/gitops-engine)

最初は GitOps Engine にリポジトリの状態監視に使えるコンポーネントがないか調べたのですが、Gitリポジトリへアクセスするコンポーネントは future work だったので、諦めました。しかし、GitOps Engine の実装例である```gitops-agent```で ```kubernetes/git-sync``` を使ったPull型のデプロイ例があったので、そのアイデアを今回ブログに利用しました。

## まとめ

今回はブログのデータ同期でしたが、「Gitリポジトリの何かしらのデータを使って、何かをしたい」という抽象度を持つ課題であれば、今回と同じようなアーキテクチャが可能です。これからも趣味でテックブログの改善をやっていきたいと思います。
