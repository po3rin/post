---
title: PyTerrierを使った日本語検索パイプラインの実装
cover: 
date: 2022/04/23
id: m3-pyterrier
description: Go is a programming language
tags:
    - Search Engine
draft: true
---

エムスリーエンジニアリンググループ AI・機械学習チームでソフトウェアエンジニアをしている中村([po3rin](https://twitter.com/po3rin)) です。検索とGoが好きです。

今回は社内で[PyTerrier](https://github.com/terrier-org/pyterrier)を採用して文書検索BatchをPythonで実装したので、PyTerrierの紹介とPyTerrierで日本語を扱う方法を紹介します。

<!-- more -->

[:contents]

## PyTerrierとは

PyTerrierは、Pythonでの情報検索実験のためのプラットフォームです。 Javaベースの[Terrier](http://terrier.org/)を内部的に使用して、インデックス作成と検索操作を行うことができます。開発と評価を一気通貫で行うことが可能です。基本的なQuery Rewritingややスコアリングアルゴリズムがすぐに使えます。

ECIR2021ではLearning to rankの実験などPyTerrierで行うチュートリアルが公開されています。
https://github.com/terrier-org/ecir2021tutorial

パイプラインを演算子で構築できるのが特徴で、例えば、BM25で100件取ってきて、PL2でリランキングするパイプラインは下記のように宣言的に実装できます。

```py
bm25 = pt.BatchRetrieve(index, wmodel="BM25")
pl2 = pt.BatchRetrieve(index, wmodel="PL2")
pipeline = (bm25 % 100) >> pl2
```

パイプラインの評価もすぐに行うことができます。例えば下記はTF-IDFとBM25の比較をmapメトリクスで行う例です。

```py
pt.Experiment([tf_idf, bm25], topic, qrels, eval_metrics=["map"])
```

## PyTerrierで日本語検索

PyTerrierで用意しているTokenizerに日本語の形態素解析はないので、自前で用意してあげる必要があります。

PyTerrierで英語以外の検索を行う例が公開されているので、これも参考にしてください。
https://colab.research.google.com/github/terrier-org/pyterrier/blob/master/examples/notebooks/non_en_retrieval.ipynb

今回は[Sudachi]で形態素解析して、PyTerrierで検索を行う方法を紹介します。SudachiをElasticsearchに導入した記事も弊社から公開しているので、Sudachiに興味のある方は是非そちらもご覧ください。

https://www.m3tech.blog/entry/sudachi-es

モジュールは下記を用意します。また、PyTerrierのcoreはJavaで実装されているので、Javaの環境も用意しておきましょう。

```py
import os

import pyterrier as pt
import pandas as pd
from sudachipy import dictionary, tokenizer
```

PyTerrierを初期化します。
```py
if not pt.started():
  pt.init()
```

今回検索する対象のドキュメントを用意しておきます。PyTerrierはPandasをそのままインデックスするインターフェースが用意されているので便利です。

```py
df = pd.DataFrame([
        ["d1", "検索方法の検討"]
    ], columns=["docno", "text"])

df
```

ドキュメントとクエリの両方を形態素解析するので、それぞれTokenizerを用意してあげます。ドキュメントは品詞でインデックスするタームを絞ります。

```py
class DocTokenizer():
    tokenizer_obj = dictionary.Dictionary().create()
    mode = tokenizer.Tokenizer.SplitMode.C

    def tokenize(self, txt: str) -> list[str]:
        return [
            m.dictionary_form() for m in self.tokenizer_obj.tokenize(txt, self.mode)
            if len(set(['名詞', '動詞', '形容詞', '副詞', '形状詞']) & set(m.part_of_speech())) != 0
        ]

class TokenizeDoc():
    tokenizer = DocTokenizer()

    def tokenize(self, df: pd.DataFrame):
        df['tokens'] = df['text'].apply(lambda x: ' '.join(self.tokenizer.tokenize(x)))
        return df
```

これで事前にドキュメントをタームに分割する用意ができました。ドキュメントのDataFrameをTokenizeします。

```py
doc_tokenizer = TokenizeDoc()
phrase_query_converter = PhraseQueryConverter()

df = doc_tokenizer.tokenize(df=df)
df

# 	docno	text	      tokens
#   d1	    検索方法の検討	検索 方法 検討
```

これでドキュメントの準備ができたので、実際にIndex処理を行います。日本語の場合はスペースで区切られる`UTFTokeniser`を利用します。事前にドキュメントをタームのスペース区切りにしてあるので、そのまま渡してあげればインデックス完了です。

```py
indexer = pt.DFIndexer('./askd-terrier', overwrite=True, blocks=True)
indexer.setProperty('tokeniser', 'UTFTokeniser')
indexer.setProperty('termpipelines', '')
index_ref = indexer.index(df['tokens'], docno=df['docno'])
index = pt.IndexFactory.of(index_ref)
```


後はクエリの処理です。PyTerrierではクエリ言語をサポートしており、And検索やPhrase検索が可能です。例えばAnd検索は`+term1 +term2`のように記述でき、Phrase検索は`"term1 term2"`のように記述できます。その他の記述方法はドキュメントをご覧ください。

http://terrier.org/docs/v5.1/querylanguage.html

今回はPhrase検索を使ってみます。形態素解析したクエリをクエリ言語に展開する実装です。

```py
class QueryTokenizer():
    tokenizer_obj = dictionary.Dictionary().create()
    mode = tokenizer.Tokenizer.SplitMode.C

    def tokenize(self, txt: str) -> list[str]:
        return [m.surface() for m in self.tokenizer_obj.tokenize(txt, self.mode)]

class PhraseQueryConverter():
    query_tokenizer = QueryTokenizer()

    def convert(self, text: str) -> str:
        tokens = [t for t in self.query_tokenizer.tokenize(text)]
        if len(tokens) <= 1:
            return text
        joined = ' '.join(tokens)
        return f'"{joined}"'
```

クエリを処理する準備ができたので、実際に検索パイプラインを実装します。今回はクエリをフレーズクエリに変換して、BM25でスコアリングして上位100件を取得するパイプラインを用意しました。

```py
pipe = (pt.apply.query(lambda row: phrase_query_converter.convert(row.query)) >> (pt.BatchRetrieve(index, wmodel='BM25') % 100).compile())
```

`compile()`は検索パイプラインのDAGを書き換えて最適化してくれます。例えば`compile`無しだとクエリにヒットするドキュメントを全件とってきて、BM25でスコアリングして上位100件を取得します。一方で`compile()`を行うとLuceneでも採用されている[Block Max WAND](http://engineering.nyu.edu/~suel/papers/bmw.pdf)などの動的プルーニング手法に書き換えられ、検索がより高速になります。compileによる最適化についてはこちらの論文が詳しいです。

https://arxiv.org/abs/2007.14271

これで検索パイプラインを実装する準備ができました。

```py
res = pipe.search('検索方法')
res

# 	qid	docid	docno	rank	score	query_0	query
# 0	1	0	d1	0	-1.584963	検索方法	"検索 方法"
```

ヒットしたドキュメントのIDとともにrankやscoreが返ってきます。また、`query_0`には元のクエリ、`query`には実際に検索が走ったクエリが結果に記載されます。もちろんフレーズクエリに書き換えているので、`検索検討`などのクエリにはヒットしません。

```py
res = pipe.search('検索検討')
res

# empty...
```

## Phrase Queryの注意点

現在Issueにあげているのですが、フレーズ検索のタームがインデックスされていないものだと、そのタームを無視して検索をする挙動を発見しました。

https://github.com/terrier-org/pyterrier/issues/298


具体的には、今回の例で言うと、下記のようなクエリでもフレーズクエリでヒットしてしまいます。
```py
res = pipe.search('検索専門')
res

#	qid	docid	docno	rank	score	query_0	query
# 0	1	0	d1	0	-1.584963	検索専門	"検索 専門"
```

直近のできる対応としては、インデックスされているタームをチェックして、もし存在しないなら、そのままのクエリを投げることでヒットを防ぐなどの対応が考えられます。

```py
def convert(self, text: str, lexicon) -> str:
    tokens = [t for t in self.query_tokenizer.tokenize(text)]

    if len(tokens) <= 1:
            return text

    # indexed tokens inculde query term (bug?: phrase query ignore non indexed term)
    for t in tokens:
        if lexicon.getLexiconEntry(t) is None:
            return text

    joined = ' '.join(tokens)
    return f'"{joined}"'


lex = index.getLexicon()

pipe = (pt.apply.query(lambda row: phrase_query_converter.convert(row.query, lex)) >> pt.BatchRetrieve(index, wmodel='BM25').compile())
```

弊社ではフレーズクエリが必要だったので、一旦この方法で対応しています。根本の原因は現在調査中です。

## まとめ

PyTerrierの紹介と、PyTerrierで日本語検索を行う方法を簡単に紹介しました。Pythonでサクッと検索したい時には便利です。一方で、PyTerrierは今回のようなLexical Searchにとどまらず、情報検索モデルの適用や、実験の評価などでも活躍するので、興味のある方は是非触ってみてください。個人的にはECIR2021のチュートリアルが非常に良い入門になりました。

https://github.com/terrier-org/ecir2021tutorial

### We're hiring !!!

エムスリーでは検索&推薦基盤の開発&改善を通して医療を前進させるエンジニアを募集しています！社内では日々検索や推薦についての議論が活発に行われています。

「ちょっと話を聞いてみたいかも」という人はこちらから！
[https://jobs.m3.com/product/:embed:cite]
