---
title: 巨大文字列辞書のパターンマッチングをAho-Corasick法で突破した
cover: 
date: 2022/04/23
id: m3-aho-corasick
description: 
tags:
    - Computer Sciencs
    - Algorithm
draft: true
---

## Overview

エムスリーエンジニアリンググループ AI・機械学習チームでソフトウェアエンジニアをしている中村([po3rin](https://twitter.com/po3rin)) です。検索とGoが好きです。

<!-- more -->

[:contents]

## BigQueryが諦めました

弊社でとある文字列辞書の部分一致を含む検索ログを取得するというタスクがありました、最初はBigQueryで実行したのですが、BigQueryが下記の`resourcesExceeded`エラーを出してストップしました。これはクエリで使用するリソースの数が多すぎるときに返されます。

https://cloud.google.com/bigquery/docs/error-messages?hl=ja

部分一致したい文字列の辞書が数十万件、対象ログも数十万件あったので、ナイーブに部分一致を考えると約100億回のループが必要です。これは非常に重く、BigQueryでも計算に長時間かかってしまいました。

## Aho-Corasick法

そこで弊社ではAho-Corasick法でパターンマッチングを行い、これをBatch処理として実行することで、この計算量の問題を回避しました。

Aho-Corasick法はAlfred Ahoらの論文[1]で提案された辞書式マッチングアルゴリズムで、全パターンのマッチングを一斉に探索するため、アルゴリズムの計算量は辞書の大きさに対しても対象テキストの長さに対しても線形であるという特徴を持ちます。

<!-- Aho-Corasick法の解説は下記の記事がおすすめです。Daachorseという複数パターン検索を提供するRust製ライブラリの解説記事ですが、その元となるトライ木やAho-Corasick法の解説が充実しています。

https://tech.legalforce.co.jp/entry/2022/02/24/140316 -->

弊社のように数十万件の巨大な辞書に対して文字列部分一致を行いたい場合は、大幅に計算コストを削減できます。

## ahocorapyとgokartを使った実装

今回は下記のAho-Corasick法のPython実装ライブラリを利用しました。

https://github.com/abusix/ahocorapy

ただ単純にahocorapyの使い方を紹介してもつまらないので、今回はgokartと合わせてAho-Corasick法を利用する方法を紹介します。
弊社ではPythonで何かを実装するときは主に[gokart[()を採用しているので、実際の運用でもgokartと合わせて実装しました。

まずはACマシンを構築するためのgokartタスクです。こうすることにより辞書が変わらなければ、ACマシンをキャッシュから復元することが可能になり効率が良くなります。

```py
class ACMachine(gokart.TaskOnKart):
    keywords = gokart.TaskInstanceParameter()
    target_column_name: str = luigi.Parameter(default='name')

    def run(self) -> None:
        df = self.load_data_frame('keywords')
        keywords = df[self.target_column_name].to_list()
        kw_tree = self._run(keywords=keywords)
        self.dump(kw_tree)

    @staticmethod
    def _run(keywords: list[str]) -> KeywordTree:
        kw_tree = KeywordTree(case_insensitive=True)

        for t in keywords:
            if t is None:
                continue
            kw_tree.add(t)

        kw_tree.finalize()
        return kw_tree
```

これでAC木を構築する用意ができました。ちなみにAC木の構築はスレッドセーフではないので、キーワードを1つ1つ渡して構築してあげる必要があります。実際に動作確認しておきます。

```py
class Keywords(gokart.TaskOnKart):
    def run(self) -> None:
        df = pd.DataFrame({
            {'name': 'aaa'},
            {'name': 'bbb'},
        })
        self.dump(df)

ac = gokart.build(ACMachine(keywords=Keywords()))
ac.search_all('aaabbbccc')
```

続いて実際に部分文字列一致を検索するgokartタスクの実装です。

```python
class Searcher():

    def __init__(self, kw_tree: KeywordTree) -> None:
        self.kw_tree = kw_tree

    def search(self, text: str) -> list[str]:
        results = self.kw_tree.search_all(text)
        result = [r[0] for r in results]
        return result


class SubStringMatchExtractor(gokart.TaskOnKart):
    data = gokart.TaskInstanceParameter()
    kw_tree = gokart.TaskInstanceParameter()

    def run(self) -> None:
        kw_tree = self.load('kw_tree')
        df = self.load_data_frame('data')
        searcher = Searcher(kw_tree=kw_tree)
        df = self._run(df=df, searcher=searcher)
        self.dump(df)

    @staticmethod
    def _run(df: pd.DataFrame, searcher: Searcher) -> pd.DataFrame:
        df['hit'] = df['text'].apply(searcher.search)
        return df
```

これでgokartパイプラインを走らせることができます。

```py
class LoadLogs(gokart.TaskOnKart):
    def run(self) -> None:
        df = pd.DataFrame({
            {'text': 'aaabbbccc'},
            {'text': 'cccdddeee'},
        })
        self.dump(df)

kw_tree = ACMachine(keywords=Keywords())
gokart.build(SubStringMatchExtractor(data=LoadLogs(), kw_tree=kw_tree))
```

## パターンマッチングの並列化

ACマシンの構築はスレッドセーフではありませんでしたが、検索はスレッドセーフなので並列化できます。データ量が多い場合は並列化が効いてくる可能性があります。例えばPandasの処理を並列化する`pandarallel`を使って並列化すると下記のようになります。

```py
from pandarallel import pandarallel

pandarallel.initialize()


class SubStringMatchExtractor(gokart.TaskOnKart):
    # ...

    @staticmethod
    def _run(df: pd.DataFrame, searcher: Searcher) -> pd.DataFrame:
        df['hit'] = df['text'].parallel_apply(searcher.search)  # <- applyをparallel_applyに置換
        return df
```

`pandarallel`を使った並列化に関する記事は下記がおすすめです。Pandasの処理を簡単に並列化できる他のライブラリとの比較などもあり、かなり詳しく書かれてます。
https://blog.ikedaosushi.com/entry/2020/07/26/173109

```py
--------------------------------------------------- benchmark: 1 tests --------------------------------------------------
Name (time in s)                  Min       Max      Mean  StdDev    Median     IQR  Outliers     OPS  Rounds  Iterations
-------------------------------------------------------------------------------------------------------------------------
test_something_benchmark     125.6607  137.9942  130.0162  5.1857  128.0501  7.7924       1;0  0.0077       5           1
-------------------------------------------------------------------------------------------------------------------------
```

## 他のAho-Corasickライブラリとの比較

## まとめ

### We're hiring !!!

エムスリーでは検索&推薦基盤の開発&改善を通して医療を前進させるエンジニアを募集しています！社内では日々検索や推薦についての議論が活発に行われています。

「ちょっと話を聞いてみたいかも」という人はこちらから！
[https://jobs.m3.com/product/:embed:cite]
