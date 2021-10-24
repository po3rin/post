---
title: Elasticsearchの新しいスコアリングBM25Fを内部実装から理解する
cover: img/nnn.png
date: 2021/10/27
id: es-bm25f
description: ElasticserchでBM25Fが使えるようなったので、概要とLucene実装の紹介
tags:
    - Search Engine
    - Elasticsearch
draft: true
---

## Overview

2021年6月にLuceneのBM25Fアルゴリズムを実装した```BM25FQuery```が```CombinedFieldQuery```に命名変更された。それを受け、Elasticsearchでもバージョン7.13で```combined_fields ```queryが実装された。

## BM25Fの概要

重み付きマルチフィールドに特化した検索用スコアリングアルゴリズムです。

AskDoctorsのQ&Aではtitleやbodyなど様々なフィールドに検索対象テキストが分かれています。今のAskDoctorsではtitleを重視している。そのためAskDoctorsのtitle重視の検索にBM25Fは有用であると考える。

## BM25Fのアルゴリズム

BM25Fは下記の論文で紹介されている。

[The Probabilistic Relevance Framework: BM25 and Beyond](https://www.staff.city.ac.uk/~sbrp622/papers/foundations_bm25_review.pdf)

[Microsoft Cambridge at TREC–13: Web and HARD tracks](http://citeseerx.ist.psu.edu/viewdoc/download;jsessionid=00CD065DCFDA823F98952765A83FAF44?doi=10.1.1.61.965&rep=rep1&type=pdf)

[Microsoft Cambridge at TREC–14: Enterprise track](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/craswell_trec05.pdf)

BM25Fは下記の式で与えられる。```w_i^simpleBM25F``` を

![simple-bm25f](./../../img/simple-bm25f.png)

ここで各変数は下記で与えられる。ここではElasticsearchの各fieldをstreamとしている。複数のフィールドを単一のstreamとみなしているためにこの呼び名になっている。

![notations](./../../img/notations.png)

```tf_is```はstream```s```に出現するターム```i```の出現頻度(frequency)である。

ここでBM25と比較すると```|D|```が```\~{dl}```に単純に置き換わっていることがわかる。つまり、フィールドの重要度で```|D|```部分を重み付けしていることがわかる。```W_i^RSJ```は　ロバートソン/スパルク・ジョーンズ重み関数であり、ドキュメントが相互に関連していないという条件を満たせばIDFと近似できる(要調査)。

![B](./../../img/b.png)
![bm25](./../../img/bm25.png)

また、フィールド毎のドキュメント長を正規化するバージョンがあり、こちらが一般的なBM25Fとされている。

![bm25](./../../img/bm25f.png)

```b_s```で正規化のレベルを調整する。

## Lucene CombinedFieldQuery コードリーディング

再掲だが、2021年6月にリリースされたLucene8.9.0からLuceneのBM25Fアルゴリズムを実装した```BM25FQuery```が```CombinedFieldQuery```に命名変更されているので注意。よって今回はLucene8.9.0を読んでいく。

ドキュメントは下記
[Class CombinedFieldQuery](https://javadoc.io/static/org.apache.lucene/lucene-sandbox/8.9.0/org/apache/lucene/search/CombinedFieldQuery.html)

ここで注意したいのが、ドキュメントを参照すると先ほど紹介した```w_i^simpleBM25F```を利用している点。教科書のBM25Fではないっぽい(simple formulaをそう解釈したが)？

> The scoring is based on BM25F's simple formula described in: http://www.staff.city.ac.uk/~sb317/papers/foundations_bm25_review.pdf. This query implements the same approach but allows other similarities besides BM25Similarity.

CombinedFieldQueryはQueryクラスを継承している。

```java
public final class CombinedFieldQuery extends Query implements Accountable 
```

使い方はいつも通り```Builder```メソッドを呼び出してクエリを取得する。

```java
CombinedFieldQuery query =
        new CombinedFieldQuery.Builder()
            .addField("field1")
            .addField("field2", 1.3f)
            .addTerm(new BytesRef("value"))
            .build();
```

今回は```CombinedFieldQuery```の実装を読んでいく。まずは```addField```だが、こちらは対象フィールドと重みを指定する。

```java
/**
* Adds a field to this builder.
*
* @param field The field name.
*/
public Builder addField(String field) {
    return addField(field, 1f);
}

/**
* Adds a field to this builder.
*
* @param field The field name.
* @param weight The weight associated to this field.
*/
public Builder addField(String field, float weight) {
    if (weight < 1) {
    throw new IllegalArgumentException("weight must be greater or equal to 1");
    }
    fieldAndWeights.put(field, new FieldAndWeight(field, weight));
    return this;
}
```

続いて```addTerm```で検索するタームを追加していく。

```java
/** Adds a term to this builder. */
public Builder addTerm(BytesRef term) {
    if (termsSet.size() > IndexSearcher.getMaxClauseCount()) {
        throw new IndexSearcher.TooManyClauses();
    }
    termsSet.add(term);
    return this;
}
```

最後に```build```メソッドを読んでQueryを取得する。

```java
/** Builds the {@link CombinedFieldQuery}. */
public CombinedFieldQuery build() {
    int size = fieldAndWeights.size() * termsSet.size();
    if (size > IndexSearcher.getMaxClauseCount()) {
        throw new IndexSearcher.TooManyClauses();
    }
    BytesRef[] terms = termsSet.toArray(new BytesRef[0]);
    return new CombinedFieldQuery(new TreeMap<>(fieldAndWeights), terms);
}
```

最後に```CombinedFieldQuery```というプライベートクラスを返している。ここで渡させた全ての設定をまとめ上げる。

```java
private CombinedFieldQuery(TreeMap<String, FieldAndWeight> fieldAndWeights, BytesRef[] terms) {
    this.fieldAndWeights = fieldAndWeights;
    this.terms = terms;
    int numFieldTerms = fieldAndWeights.size() * terms.length;
    if (numFieldTerms > IndexSearcher.getMaxClauseCount()) {
      throw new IndexSearcher.TooManyClauses();
    }
    this.fieldTerms = new Term[numFieldTerms];
    Arrays.sort(terms);
    int pos = 0;
    for (String field : fieldAndWeights.keySet()) {
      for (BytesRef term : terms) {
        fieldTerms[pos++] = new Term(field, term);
      }
    }

    this.ramBytesUsed =
        BASE_RAM_BYTES
            + RamUsageEstimator.sizeOfObject(fieldAndWeights)
            + RamUsageEstimator.sizeOfObject(fieldTerms)
            + RamUsageEstimator.sizeOfObject(terms);
  }
```

## ElasticsearchでのBM25F

下記のように```combined_fields```を指定することでElasticsearchから```CombinedFieldQuery```を利用できる。

```json
GET topics/_search
{
  "query": {
    "combined_fields": {
      "query": "コロナウィルス",
      "fields": [
        "title^5",
        "body",
      ]
    }
  }
}
```

下記はAskDoctorsの検索エンジンのBM25をBM25Fに入れ替える、つまり```combined_fields```を利用した結果である。

```
combined_fields(BM25F)
————————
コロナウイルスのワクチン
コロナウィルスのワクチンについて
コロナウィルスのワクチンと薬疹について
コロナワクチン接種について
コロナのワクチン接種について
コロナワクチンについての質問します
コロナワクチンとインフルエンザワクチンの同時接種
肺炎球菌ワクチンとコロナワクチン接種
ｺﾛﾅﾜｸﾁﾝについて
コロナワクチンについて

multi_match(従来)
——————-
コロナウイルスのワクチン
コロナウイルスのワクチン
コロナウィルスワクチン
コロナワクチン
コロナワクチンって
コロナウイルスとインフルエンザワクチン
コロナウィルスワクチン接種
ピルとコロナ・コロナワクチン
コロナウィルスワクチンについて
コロナウイルスワクチンについて
```
