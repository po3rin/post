---
title: Lucene More like this コードリーディング
cover: img/nnn.png
date: 2022/01/08
id: es-mlt
description: LuceneのMore like thisの内部実装を読む
tags:
    - Lucene
    - Elasticsearch
draft: true
---

## Introduction

LuceneにはMore like this(MLT)という機能があり、似た文書を探すなどに利用できる。
https://lucene.apache.org/core/7_2_0/queries/org/apache/lucene/queries/mlt/MoreLikeThis.html#like-int-

ElasticsearchからもLuceneのMLT機能にアクセスでき、More like this APIとして利用できる。
https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl-mlt-query.html

今回はLuceneのMLTのコードリーディングを通して、MLTの実装を理解する。

## Lucene MLT Reading

documentに簡単な使い方が書いてあるのでここをコードリーディングの起点とする。

https://lucene.apache.org/core/7_2_0/queries/org/apache/lucene/queries/mlt/MoreLikeThis.html#like-int-

```java
IndexReader ir = ...
IndexSearcher is = ...

MoreLikeThis mlt = new MoreLikeThis(ir);
Reader target = ... // orig source of doc you want to find similarities to
Query query = mlt.like(target);

Hits hits = is.search(query);
// now the usual iteration thru 'hits' - the only thing to watch for is to make sure
//you ignore the doc if it matches your 'target' document, as it should be similar to itself
```

MoreLikeThisは主に検索クエリを生成するためだけの責務を持つようだ。実際の検索はMLTから分離されたいつもの検索インターフェースを利用する。`IndexReader`を`MoreLikeThis`クラスに渡しているが、これはタームのTFやIDFをインデックスから取得するために利用していると考えられる。

MoreLikeThisクラスの具体的な使い方は下記(テストコードから抜粋)。

```java
MoreLikeThis mlt = new MoreLikeThis(reader);
    Analyzer analyzer = new MockAnalyzer(random(), MockTokenizer.WHITESPACE, false);
    mlt.setAnalyzer(analyzer);
    mlt.setMaxQueryTerms(3);
    mlt.setMinDocFreq(1);
    mlt.setMinTermFreq(1);
    mlt.setMinWordLen(1);
    mlt.setFieldNames(new String[] {"one_percent"});

    BooleanQuery query =
        (BooleanQuery) mlt.like("one_percent", new StringReader("tenth tenth all"));
    Collection<BooleanClause> clauses = query.clauses();
```

上の例ではAnalyzerの設定や、ElasticsearchのMLTから指定できるパラメータを付与できることがわかる。では実際にlikeメソッドの中を除いてみる。

```java
public Query like(String fieldName, Reader... readers) throws IOException {
    Map<String, Map<String, Int>> perFieldTermFrequencies = new HashMap<>();
    for (Reader r : readers) {
      addTermFrequencies(r, perFieldTermFrequencies, fieldName);
    }
    return createQuery(createQueue(perFieldTermFrequencies));
  }
```

内部的にはReaderから`perFieldTermFrequences`マップに`addTermFrequencies`メソッドでタームを追加し、最終的にcreateQueueに渡されてクエリを生成している。`addTermFrequencies`の中身を見てみると実際にAnalyzerでテキストをタームに分割していることがわかる。

```java
private void addTermFrequencies(
      Reader r, Map<String, Map<String, Int>> perFieldTermFrequencies, String fieldName)
      throws IOException {
    if (analyzer == null) {
      throw new UnsupportedOperationException(
          "To use MoreLikeThis without " + "term vectors, you must provide an Analyzer");
    }
    Map<String, Int> termFreqMap =
        perFieldTermFrequencies.computeIfAbsent(fieldName, k -> new HashMap<>());
    try (TokenStream ts = analyzer.tokenStream(fieldName, r)) {
      int tokenCount = 0;
      // for every token
      CharTermAttribute termAtt = ts.addAttribute(CharTermAttribute.class);
      TermFrequencyAttribute tfAtt = ts.addAttribute(TermFrequencyAttribute.class);
      ts.reset();
      while (ts.incrementToken()) {
        String word = termAtt.toString();
        tokenCount++;
        if (tokenCount > maxNumTokensParsed) {
          break;
        }
        if (isNoiseWord(word)) {
          continue;
        }

        // increment frequency
        Int cnt = termFreqMap.get(word);
        if (cnt == null) {
          termFreqMap.put(word, new Int(tfAtt.getTermFrequency()));
        } else {
          cnt.x += tfAtt.getTermFrequency();
        }
      }
      ts.end();
    }
  }
```

ここで作ったターム情報を`createQueue`に渡す。ここではMLTのオプションで指定した`minTermFreq`は`maxDocFreq`などでフィルタして最終的な的な検索用のタームを作成する。

```java
  private PriorityQueue<ScoreTerm> createQueue(
      Map<String, Map<String, Int>> perFieldTermFrequencies) throws IOException {
    // have collected all words in doc and their freqs
    final int limit = Math.min(maxQueryTerms, this.getTermsCount(perFieldTermFrequencies));
    FreqQ queue = new FreqQ(limit); // will order words by score
    for (Map.Entry<String, Map<String, Int>> entry : perFieldTermFrequencies.entrySet()) {
      Map<String, Int> perWordTermFrequencies = entry.getValue();
      String fieldName = entry.getKey();

      long numDocs = ir.getDocCount(fieldName);
      if (numDocs == -1) {
        numDocs = ir.numDocs();
      }

      for (Map.Entry<String, Int> tfEntry : perWordTermFrequencies.entrySet()) { // for every word
        String word = tfEntry.getKey();
        int tf = tfEntry.getValue().x; // term freq in the source doc
        if (minTermFreq > 0 && tf < minTermFreq) {
          continue; // filter out words that don't occur enough times in the source
        }

        int docFreq = ir.docFreq(new Term(fieldName, word));

        if (minDocFreq > 0 && docFreq < minDocFreq) {
          continue; // filter out words that don't occur in enough docs
        }

        if (docFreq > maxDocFreq) {
          continue; // filter out words that occur in too many docs
        }

        if (docFreq == 0) {
          continue; // index update problem?
        }

        float idf = similarity.idf(docFreq, numDocs);
        float score = tf * idf;

        if (queue.size() < limit) {
          // there is still space in the queue
          queue.add(new ScoreTerm(word, fieldName, score));
        } else {
          ScoreTerm term = queue.top();
          // update the smallest in the queue in place and update the queue.
          if (term.score < score) {
            term.update(word, fieldName, score);
            queue.updateTop();
          }
        }
      }
    }
    return queue;
  }
```

ここではタームごとのTF-IDFのスコアを計算している。スコアの高いタームを順番にqueueから取り出してクエリタームとして取り出している。

また、このスコアは後でクエリを作るときのブーストにも利用する。

最終的に残ったタームとスコアを使って`createQuery`を行う。

```java
  /** Create the More like query from a PriorityQueue */
  private Query createQuery(PriorityQueue<ScoreTerm> q) {
    BooleanQuery.Builder query = new BooleanQuery.Builder();
    ScoreTerm scoreTerm;
    float bestScore = -1;

    while ((scoreTerm = q.pop()) != null) {
      Query tq = new TermQuery(new Term(scoreTerm.topField, scoreTerm.word));

      if (boost) {
        if (bestScore == -1) {
          bestScore = (scoreTerm.score);
        }
        float myScore = (scoreTerm.score);
        tq = new BoostQuery(tq, boostFactor * myScore / bestScore);
      }

      try {
        query.add(tq, BooleanClause.Occur.SHOULD);
      } catch (
          @SuppressWarnings("unused")
          IndexSearcher.TooManyClauses ignore) {
        break;
      }
    }
    return query.build();
  }
```

それぞれのタームをスコアでboostして、shouldクエリで繋いでいる。つまりMLTのパフォーマンスは検索に使うターム数が影響することがわかる。これはElasticsearchの`max_query_terms`パラメータのドキュメントにもあるようにタームの数でパフォーマンスがわかる旨の説明と一致する。

https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl-mlt-query.html#mlt-query-term-selection

> The maximum number of query terms that will be selected. Increasing this value gives greater accuracy at the expense of query execution speed. Defaults to 25.

boostパラメータがFalseなら先ほど計算したスコアによるブーストは行わないようだ。掛け算、割り算くらいのコストなので、クエリ生成のパフォーマンスにはそれほど影響はなさそう。

また今回はドキュメントを直接Readerに変換して渡す方法をみたが、すでにインデックスされているドキュメントを入力ドキュメントとして指定する方法もある。

```java
  public Query like(int docNum) throws IOException {
    if (fieldNames == null) {
      // gather list of valid fields from lucene
      Collection<String> fields = FieldInfos.getIndexedFields(ir);
      fieldNames = fields.toArray(new String[fields.size()]);
    }

    return createQuery(retrieveTerms(docNum));
  }
```

## まとめ

今回はMore like thisの内部実装を除いた。やはりコードを読むと機能の理解が進みやすい。
