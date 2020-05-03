---
title: リトライ処理の効率的アプローチ「Exponential Backoff」の概要とGoによる実装
cover: img/gopher.png
date: 2019/06/11
id: exp-backoff
description: 通信先サーバーに過度の負荷をかけないようにするためのリトライ手法「Exponential Backoff」の実装方法をGoを使って説明します。
tags:
    - Go
---

## Exponential Backoff とは

少しCloud IoT Coreよりですが、GCPのドキュメントに良い解説がありますので参考に。
https://cloud.google.com/iot/docs/how-tos/exponential-backoff

リトライの際に一定感覚で処理を再試行すると、通信を受け取る側のサーバーに大きな負荷がかかる可能性があります。そこで「Exponential Backoff」という考え方が出てきます。「Exponential Backoff」はクライアントが通信に失敗した際に要求間の遅延を増やしながら定期的に再試行するアプローチです。一般的なエラー処理戦略として知られています。

例えば

* 1回目のリクエスト失敗、1 + random_number_milliseconds 秒待って再試行。
* 2回目のリクエスト失敗、2 + random_number_milliseconds 秒待って再試行。
* 3回目のリクエスト失敗、4 + random_number_milliseconds 秒待って再試行。
* ...(任意回数、待機時間を指数的に増加させながら再試行)

と待機時間を指数関数的に増加させていきます。そして任意の最大再試行回数まで再試行します。これでサーバの負荷を軽減したり、無駄なリクエストも省けます。結果的にリトライ成功可能性が上がります。

日本語だとここの解説が良いでしょう
https://note.mu/artrigger_jp/n/n0795148b062d

## Goによる実装

例としてGoによる実装を見ていきます。簡単に実装できることが分かると思います。

```go
// Exponential Backoff Algorism for retry
func Retry() error {
    var retries int
    maxRetries := 5

    for {
        err = Something()
        if err != nil {
            if retries > maxRetries {
                return err
            }

            waitTime := 2 ^ retries + rand.Intn(1000)/1000
            time.Sleep(time.Duration(waitTime) * time.Second)

            retries++
            continue
        }
        break
    }
    return nil
}
```

当然、maxRetriesは実装によって変わります。これで Retry の待機時間を指数関数的に増やしていけます。Retry機構を実装する際には是非この実装を試してみてください。

