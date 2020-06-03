---
title: 破棄容易性の観点から、データパイプラインのk8s移行にまつわるGoコード改修を振り返る
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/mq-k8s-container.jpeg
date: 2020/06/02
id: mq-reconnect
description: 破棄容易性の観点から、弊社でのk8s移行にまつわるコード改修を振り返ります。
tags:
    - Go
    - Kubernetes
---

## Overview

こんにちは [pon](https://twitter.com/home) です。***Kubernetes*** が盛り上がるこの頃ですが、私が働いている[白ヤギコーポレーション](https://shiroyagi.co.jp/)では記事データを処理するパイプラインがあります。その記事データパイプラインを ***EKS*** に移行するに辺り、発生したコード改修について紹介します。Kubernetes移行を検討するにあたって、KubernetesのManifestを淡々と書いていくことも大切ですが、場合によってはアプリケーションコードの改修も発生します。今回は破棄容易性の観点から、弊社のk8s移行で発生したコード修正を紹介します。

## 廃棄容易性とは

[The Twelve-Factor App](https://12factor.net/ja/)で ***廃棄容易性(Disposability)*** について言及されているので知っている方も多いと思います。***The Twelve-Factor App*** はモダンなアプリケーションが満たすべき12のベストプラクティスをまとめた方法論です。そこから破棄容易性についての記述を取り上げます。

> Twelve-Factor Appの プロセス は 廃棄容易 である、すなわち即座に起動・終了することができる。 この性質が、素早く柔軟なスケールと、コード や 設定 に対する変更の素早いデプロイを容易にし、本番デプロイの堅牢性を高める。

弊社での今回の移行のポイントは ***「終了することができる」*** というポイントでした。弊社の記事処理パイプラインは今まで、アプリケーションを安全に終了するという点をあまり考えず開発されていました。今までは大きな問題になりませんでしたが、k8sで動かそうとすると大きな問題になります。

k8sではPodの停止ができることが前提であり、これは[ドキュメント](https://kubernetes.io/ja/docs/concepts/workloads/pods/pod/#pod%E3%81%AE%E7%B5%82%E4%BA%86)でも述べられています。

> Podは、クラスター内のNodeで実行中のプロセスを表すため、不要になったときにそれらのプロセスを正常に終了できるようにすることが重要です（対照的なケースは、KILLシグナルで強制終了され、クリーンアップする機会がない場合）。 ユーザーは削除を要求可能であるべきで、プロセスがいつ終了するかを知ることができなければなりませんが、削除が最終的に完了することも保証できるべきです。

ゆえにk8s移行において破棄容易性は満たすべき必須の性質になります。Podを終了する際にk8s側で猶予期間を設定できますが筆者は、***安全な停止はアプリケーション側で責務として負う必要がある*** と考えています。終了時に処理していたデータの退避や扱っていたユーザーからのリクエストを捌いてからの終了はアプリケーション側でないとハンドリングできないからです。

ここからは実際にk8s移行で必要だった改修を簡単にご紹介します。弊社でのk8s移行の際には主に3点のコード改修が発生しました。

* Gracefull Shutdown
* 再入可能性の担保
* MQの再接続処理

順番に紹介していきます。

## 弊社の記事データパイプライン構成

まずは弊社の記事データパイプラインの構成を簡単に紹介します。 Go + RabbitMQ で実装されており、クローラーがデータベースに溜めた記事ポーリングして、タグ付け、重複判定、不適切記事判定などを行い、最終的にElasticsearchに投げます。ElasticsearchやデータパイプラインはEKS上にデプロイされています。

![パイプラインの構成](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/mq-k8s-pipeline.png)

## Gracefull Shutdown

Podの終了の際には、まず開いているリスナーをすべて閉じ、処理中のリクエストが終了するまで待ってから終了(***gracefull shutdown***)する必要があります。まずは弊社の元々のコードを見てみましょう。エラーハンドリング含め、コードは少し省略してありますが、ほとんど元々のコードです。

```go
func main() {
    // ...
    // データベースから記事を読み込んでMQに投げる。内部でgorutineを走らせている。
    _ = RunReader()
    // HTTPで記事のパイプライン処理を走らせるためのエンドポイント
    _ = http.ListenAndServe(port, nil)
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
    <-sigs
}

```

この実装だと```SIGTERM``` を受けたらそのまま直ぐにプロセスが終了してしまうのでリクエストを処理していてもレスポンスを返せず終了してしまいます。そのため Twelve-Factor App で言及のある破棄容易性に沿って ***gracefull shutdown*** を実装する必要があります。

改修にあたってまずはgorutineのエラーハンドリングをして全てのgorutineを安全に終了できるようにする必要があります。ここでは```golang.org/x/sync/errgroup```パッケージで並行処理でエラーが発生したらcontext cancelを呼び、他のgorutineも停止していきます。当然```SIGTERM```を受けた際にもcontext cancelをしてgorutineに安全な停止を命令します。

```go
ctx, cancel := context.WithCancel(ctx)
defer cancel()

eg, ctx := errgroup.WithContext(ctx)
eg.Go(func() error {
    // context cancel で終了できるようにする。
    return mq.RunReader(ctx)
})
eg.Go(func() error {
     // context cancel で終了できるようにする。
     // context cancel を受けたら関数内で gracefull shutdown を行なっている。
    return httpServe(ctx)
})

sigs := make(chan os.Signal, 1)
signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
select {
case s := <-sigs:
    // シグナルを受けたら context cancel で gorutine を安全に止めに行く。
    logger.Infof("receive %v", s.String())
    cancel()
case <-ctx.Done():
}

// 全ての gorutine の終了を待つ。
if err := eg.Wait(); err != nil {
    if err != context.Canceled {
        // error handling
    }
}
```

このような実装は特に ***gorutineを複数起動している時*** に役立ちます。```golang.org/x/sync/errgroup```を使って全てのgorutineが安全に終了するのを待ってプロセスを終了することが可能なので、データの退避やリクエストを処理しきるなどの実装が可能になります。

上記のコードの```httpServe```関数ではcontext cancelを受けたら```gracefull shutdown```をするようにします。Goの標準パッケージを使ったgracefull shutdownでは[func (*Server) Shutdown](https://golang.org/pkg/net/http/#Server.Shutdown)というメソッドが提供されています。

```go
func httpServe(ctx context.Context) error {
	// ...
	srv := &http.Server{
		// ...
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return srv.ListenAndServe()
	})

	<-ctx.Done()

    // shutdhown用のcontext
	sCtx, sCancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer sCancel()
	if err := srv.Shutdown(sCtx); err != nil {
		logger.WithError(err).Error("Failed to shutdown server")
	}

	return ctx.Err()
}
```

これでk8sから```SIGTERM```を受ける、もしくは対処できないエラーが発生した際でもcontext cancelを利用した安全な停止が行えます。

## 再入可能性の担保

[破棄容易性](https://12factor.net/ja/disposability)のページではドンピシャでRabbitMQを利用している際の対処方が説明されています。

> ワーカープロセスの場合、グレースフルシャットダウンは、処理中のジョブをワーカーキューに戻すことで実現される。例えば、RabbitMQではワーカーはNACKを送ることができる。

処理中のジョブをワーカーキューに戻すことで実現されるこの性質をTwelve-Factor Appでは ***再入可能性*** として挙げています。

GoのRabbitMQクライアントでは[func (*Channel) Nack](https://godoc.org/github.com/streadway/amqp#Channel.Nack)が```NACK```を送る関数です。これで記事データが処理できなかったことをRabbitMQサーバに通知できます。

```go
func (ch *Channel) Nack(tag uint64, multiple bool, requeue bool) error
```

ちゃんと処理ができたことをRabbitMQサーバに伝える際には[func (*Channel) Ack](https://godoc.org/github.com/streadway/amqp#Channel.Ack)を使いましょう。```ACK```を返さないうちにChannel, Connection, あるいはTCPコネクションがclosedになると、RabbitMQはメッセージが正しく処理されていないものとして、re-queueしてくれます。

```go
func (ch *Channel) Ack(tag uint64, multiple bool) error
```

これで再入可能性をクリアできます。


## MQの再接続処理

***再接続処理*** を実装していないと、RabbitMQのPodを入れ替えるタイミングなどでTCPコネクションが切れてGoのパイプラインがRabbitMQと接続できなくなり、 パイプライン全体が死にます。そのため、ある一定期間までは再接続処理を行う必要があります。弊社ではコネクションが切れたら再接続を行う関数を```gorutine```で実行しています。

```go
func (mq *rabbitMQ) Reconnector(ctx context.Context) error {
	for {
		select {
		case conErr := <-mq.conErrC:
			if conErr != nil {
				logger.WithError(conErr).Error("mq: reader received connection error")
			}
			err := mq.connectWithRetry(connectRetry)
			if err != nil {
				return errors.Wrap(err, "mq: retry connecting to RabbitMQ")
			}
			return mq.Reconnector(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
```

再接続処理は非常に重要なのでテストも実装しています。省略バージョンですが下記のようなテストが実装されています。1秒に1回、計6回送信を行い、3秒立ったところでコネクションを切っても処理が継続するかをチェックしています。再接続処理のテストがあるだけで安心感が違います。

```go
func TestReconnector(t *testing.T) {
    // ...
	m, _ = mq.NewRabbitMQ("localhost", port, "guest", "guest", 10)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return m.Reconnector(ctx)
	})

	go func() {
		time.Sleep(3 * time.Second)
		m.Close()
	}()

    testNum := 6
	for i := 0; i < testNum; i++ {
		err = m.Send([]byte("test!!"))
		if err != nil {
			t.Errorf("unexpected error: %+v", err)
		}
		time.Sleep(1 * time.Second)
	}

	// wait gorutine
	cancel()
	if err := eg.Wait(); err != nil {
		if err != context.Canceled {
			t.Errorf("unexpected err: %v", err)
		}
	}
}
```

RabbitMQの再接続処理実装の全貌は下記のGistが勉強になります。
[Example of RabbitMQ reconnect feature.](https://gist.github.com/tomekbielaszewski/51d580ca69dcaa994995a2b16cbdc706)

## まとめ

弊社でのデータパイプラインのEKS移行で発生したコード改修を破棄容易性の観点から振り返りました。APIサーバーの場合は ***gracefull shutdown*** はもちろん、複数の並行処理を走らせる場合は、context cancel などで全ての gorutine を安全に止めることが重要です。MQを使ったデータパイプラインではキューの ***再入可能性*** を考慮し、アプリケーションによっては ***再接続処理*** も必要になります。k8sを初めて導入する際に気づかないような改修ポイントもあるので、The Twelve-Factor App は一読しておきましょう。

## おまけ

今回紹介したTwelve-Factor Appにはアップデート版の ***Beyond the Twelve-Factor App*** が存在する。[ここ](https://tanzu.vmware.com/content/blog/beyond-the-twelve-factor-app)からPDFをダウンロードできる。
