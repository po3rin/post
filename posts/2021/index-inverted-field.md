---
title: 読んで理解する全文検索 (IndexingChain ~ Codec到達編)
cover: img/gopher.png
date: 2021/02/03
id: index-inverted-field
description: Go is a programming language
tags:
    - Lucene
    - Java
draft: true
---

## Overview

こんにちは [pon](https://twitter.com/po3rin) です。
全文検索ライブラリであるLuceneの内部を理解する第一弾として **「DWPT, IndexingChain 導入編」** を前回書きました。今回はその続きの **「IndexingChain -> Codec到達編」** です！

## 前回の復習

indexChain内で```processField```でドキュメントのフィールドごとに処理が別れていることを確認しました。

```java
private int processField(int docID, IndexableField field, long fieldGen, int fieldCount) throws IOException {
    // ...

    // Invert indexed fields:
    if (fieldType.indexOptions() != IndexOptions.NONE) {
      // ...
    }

    // Add stored fields:
    if (fieldType.stored()) {
      // ...
    }

    if (dvType != DocValuesType.NONE) {
      // ...
    }

    if (fieldType.pointDimensionCount() != 0) {
      // ...
    }
    
    return fieldCount;
  }
```

今回は ***Invert indexed fields*** の処理を追っていきます。

## Invert indexed fields

***Invert indexed fields*** は転地インデックスとして格納するフィールド、つまりElasticsearchでいう```text```フィールドなどですね。もう一度、***Invert indexed fields***を処理するコードの部分を見てみます。

```java
private int processField(int docID, IndexableField field, long fieldGen, int fieldCount) throws IOException {
    String fieldName = field.name();
    IndexableFieldType fieldType = field.fieldType();

    PerField fp = null;

    // ...
    // Invert indexed fields:
    if (fieldType.indexOptions() != IndexOptions.NONE) {
      fp = getOrAddField(fieldName, fieldType, true);
      boolean first = fp.fieldGen != fieldGen;
      fp.invert(docID, field, first);

      if (first) {
        fields[fieldCount++] = fp;
        fp.fieldGen = fieldGen;
      }
    } else {
      // ...
    }

    // ...
}
```

getOrAddFieldでは名前から```PerField```を取得、なければ指定したフィールド情報を追加した状態で返します。```PerField```はプライベートクラスであり、初期化メソッドではElasticsearch利用者ならピンと来るように、analyzerの設定などもフィールドごとに行われていることがわかります。これらの設定の多くは```indexWriterConfig```で渡されます。

```java
PerField(int indexCreatedVersionMajor, FieldInfo fieldInfo, boolean invert, Similarity similarity, InfoStream infoStream, Analyzer analyzer) {
      this.indexCreatedVersionMajor = indexCreatedVersionMajor;
      this.fieldInfo = fieldInfo;
      this.similarity = similarity;
      this.infoStream = infoStream;
      this.analyzer = analyzer;
      if (invert) {
        setInvertState();
      }
    }
```

```getOrAddField```内部では ***ハッシュマップ*** を利用しており、フィールド名のハッシュを使ってフィールド情報をとってくるようです。

```java
private PerField getOrAddField(String name, IndexableFieldType fieldType, boolean invert) {

    // Make sure we have a PerField allocated
    final int hashPos = name.hashCode() & hashMask;
    PerField fp = fieldHash[hashPos];
    while (fp != null && !fp.fieldInfo.name.equals(name)) {
      fp = fp.next;
    }

    // 省略...

    return fp;
}
```

もしハッシュマップに指定フィールドがない場合は、先ほど紹介した```PerField```の初期化が呼ばれます。コードは載せないので、興味のある方は```getOrAddField```関数を追ってみてください。

ちなみにこの```getOrAddField```の処理は今回追っているフィールドのタイプ以外でもindexChain内で同様の処理が行われます。

最初に記載したコード(```processField```)に戻ると```PerField```から生えている```invert```メソッドが呼ばれています。こちらかなり長いので後ほどみていきます。

また、```PerField```初期化時に呼ばれる```setInvertState```関数では転置インデックスを利用するときのみ呼ばれます。この関数でここに転置インデックスの情報をメモリに加えていってflushが呼ばれた時に永続化するようです。```FieldInvertState```のコメントを読むとtermの```position```や```offset```などが格納されることがわかります。

```java
/**
 * This class tracks the number and position / offset parameters of terms
 * being added to the index. The information collected in this class is
 * also used to calculate the normalization factor for a field.
 * 
 * @lucene.experimental
 */
public final class FieldInvertState 
```

では```FieldInvertState```に情報を付与していく```invert```をみていきます。この時点でドキュメントを```tokenStream```に変換して処理していることが伺えます。

```java
public void invert(int docID, IndexableField field, boolean first) throws IOException {
      // ...
      try (TokenStream stream = tokenStream = field.tokenStream(analyzer, tokenStream)) {
        // reset the TokenStream to the first token
        stream.reset();
        invertState.setAttributeSource(stream);
        termsHashPerField.start(field, first);

        while (stream.incrementToken()) {
            // FieldInvertStateにpositionやoffsetなどの情報を格納していく

          try {
            // termsHashPerFieldにトークンを情報を格納する
            termsHashPerField.add(invertState.termAttribute.getBytesRef(), docID);
          } catch (MaxBytesLengthExceededException e) {
            // ...
          }
```