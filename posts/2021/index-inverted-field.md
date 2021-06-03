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

***Invert indexed fields*** は転地インデックスとして格納するフィールド、つまりElasticsearchでいう```text```フィールドなどですね。***Invert indexed fields***を処理するコードの部分を見てみます。

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

ここで```IndexOptions.None```ですが、これはIndexするときのオプションでいくつか種類があります。termの頻度や出現位置を格納するかどうかを決定するようです。

```java
public enum IndexOptions { 
  NONE,
  DOCS,
  DOCS_AND_FREQS,
  DOCS_AND_FREQS_AND_POSITIONS,
  DOCS_AND_FREQS_AND_POSITIONS_AND_OFFSETS,
}
```

先ほどのコードに戻りましょう。getOrAddFieldでは名前から```PerField```を取得、なければ指定したフィールド情報を追加した状態で返します。```PerField```はプライベートクラスであり、初期化メソッドではElasticsearch利用者ならご存知のように、analyzerの設定などもフィールドごとに行われていることがわかります。これらの設定の多くはコードの上流で```indexWriterConfig```経由で渡されます。

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

では```FieldInvertState```に情報を付与していく```invert```をみていきます。この時点でドキュメントを```tokenStream```に変換して処理していることが伺えます。そして各種情報(token位置やFrequencyなど)を```FieldInvertState```インスタンスに格納していきます。そして最終的に```termsHashPerField```に格納していることがわかります。

```java
private final class PerField implements Comparable<PerField> {
    // ...

    FieldInvertState invertState;
    TermsHashPerField termsHashPerField;

    // ...

    public void invert(int docID, IndexableField field, boolean first) throws IOException {
          // ...
          try (TokenStream stream = tokenStream = field.tokenStream(analyzer, tokenStream)) {
            // reset the TokenStream to the first token
            stream.reset();
            invertState.setAttributeSource(stream);
            termsHashPerField.start(field, first);

            while (stream.incrementToken()) {
              // FieldInvertState(変数名: invertState)にpositionやoffsetなどの情報を格納していく

              try {
                // termsHashPerFieldにトークンを情報を格納する
                termsHashPerField.add(invertState.termAttribute.getBytesRef(), docID);
              } catch (MaxBytesLengthExceededException e) {
                // ...
              }
    // ...
```

```FieldInvertState```と```TermsHashPerField```がそれぞれどのようなクラスなのかみていきましょう。まずは```FieldInvertState```です。

```java
/**
 * This class tracks the number and position / offset parameters of terms being added to the index.
 * The information collected in this class is also used to calculate the normalization factor for a
 * field.
 *
 * @lucene.experimental
 */
public final class FieldInvertState {
  final int indexCreatedVersionMajor;
  final String name;
  final IndexOptions indexOptions;
  int position;
  int length;
  int numOverlap;
  int offset;
  int maxTermFrequency;
  int uniqueTermCount;
  // we must track these across field instances (multi-valued case)
  int lastStartOffset = 0;
  int lastPosition = 0;
  AttributeSource attributeSource;

  OffsetAttribute offsetAttribute;
  PositionIncrementAttribute posIncrAttribute;
  PayloadAttribute payloadAttribute;
  TermToBytesRefAttribute termAttribute;
  TermFrequencyAttribute termFreqAttribute;

  // ...
```

処理したtermの数や処理したtermの現在地などを追跡するクラスのようです。このクラスの```termAttribute```が```TermsHashPerField.add```に渡されます。


ぶっちゃけここからが本番。```TermsHashPerField```はこんな感じ。クラスのドキュメントを読むとこれがlinked listであることが伺える。メンバ変数を見るとついにインメモリ転置インデックスにたどり着いた感がある。

```java
abstract class TermsHashPerField implements Comparable<TermsHashPerField> {
  private static final int HASH_INIT_SIZE = 4;
  private final IntBlockPool intPool;
  final ByteBlockPool bytePool;
  private int[] termStreamAddressBuffer;
  private int streamAddressOffset;
  private final int streamCount;
  private final String fieldName;
  final IndexOptions indexOptions;
  private final BytesRefHash bytesHash;
  ParallelPostingsArray postingsArray;

  void add(BytesRef termBytes, final int docID) throws IOException {
    assert assertDocId(docID);
    int termID = bytesHash.add(termBytes);
    if (termID >= 0) { // New posting
      initStreamSlices(termID, docID);
    } else {
      termID = positionStreamSlice(termID, docID);
    }
    if (doNextCall) {
      nextPerField.add(postingsArray.textStarts[termID], docID);
    }
  }
// ...
```

まずは```bytesHash```に```termBytes```を```add```している。```bytesHash```のクラス```BytesRefHash```をのぞいてみましょう。

```java
/**
 * {@link BytesRefHash} is a special purpose hash-map like data-structure optimized for {@link
 * BytesRef} instances. BytesRefHash maintains mappings of byte arrays to ids
 * (Map&lt;BytesRef,int&gt;) storing the hashed bytes efficiently in continuous storage. The mapping
 * to the id is encapsulated inside {@link BytesRefHash} and is guaranteed to be increased for each
 * added {@link BytesRef}.
 *
 * <p>Note: The maximum capacity {@link BytesRef} instance passed to {@link #add(BytesRef)} must not
 * be longer than {@link ByteBlockPool#BYTE_BLOCK_SIZE}-2. The internal storage is limited to 2GB
 * total byte storage.
 *
 * @lucene.internal
 */
public final class BytesRefHash implements Accountable
```

クラスのドキュメントを見ると、```BytesRef```を効率的に格納する特別なハッシュマップになっているようだ。```BytesRefHash```の```add```をみていく

```java
public int add(BytesRef bytes) {
    assert bytesStart != null : "Bytesstart is null - not initialized";
    final int length = bytes.length;
    // final position
    final int hashPos = findHash(bytes);
    int e = ids[hashPos];

    if (e == -1) {
      // new entry
      // ...
      e = count++;

      // ...
      return e;
    }
    return -(e + 1);
  }
```

この関数の返り値として0からインクリメントされるterm_idを返している。もしすでに存在していた場合はその```-(e + 1)```を返している。これですでにあった場合のidであることを認識している。


ここで ```field field filed```というstreamを空のFiledにindexする場合にどうなるかみてみる。

```java
  void add(BytesRef termBytes, final int docID) throws IOException {
    // ...
    int termID = bytesHash.add(termBytes);
    System.out.println("add term=" + termBytes.utf8ToString() + " doc=" + docID + " termID=" + termID);
    // ...
  }
// ...
```

これでテスト用のドキュメントを書き換えてテスト実行すると下記のようになる。

```
-----field1----
---------
add term=field doc=0 termID=0
---------
add term=one doc=0 termID=1
---------
add term=text doc=0 termID=2
---------

-----field2----
---------
add term=field doc=0 termID=0
---------
add term=field doc=0 termID=-1
---------
add term=field doc=0 termID=-1
---------
```

つまりここまででtermのbytesとdoc_idとterm_id(重複termかもわかる)が用意できている。

ついにこれらを格納していくぞ！新しいtermだった場合に呼ばれる```TermsHashPerField```.addのinitStreamSlicesをみていこう。

```java
private void initStreamSlices(int termID, int docID) throws IOException {
    // Init stream slices
    // TODO: figure out why this is 2*streamCount here. streamCount should be enough?
    if ((2 * streamCount) + intPool.intUpto > IntBlockPool.INT_BLOCK_SIZE) {
      // can we fit all the streams in the current buffer?
      intPool.nextBuffer();
    }

    if (ByteBlockPool.BYTE_BLOCK_SIZE - bytePool.byteUpto
        < (2 * streamCount) * ByteBlockPool.FIRST_LEVEL_SIZE) {
      // can we fit at least one byte per stream in the current buffer, if not allocate a new one
      bytePool.nextBuffer();
    }

    termStreamAddressBuffer = intPool.buffer;
    streamAddressOffset = intPool.intUpto;
    intPool.intUpto += streamCount; // advance the pool to reserve the N streams for this term

    postingsArray.addressOffset[termID] = streamAddressOffset + intPool.intOffset;

    for (int i = 0; i < streamCount; i++) {
      // initialize each stream with a slice we start with ByteBlockPool.FIRST_LEVEL_SIZE)
      // and grow as we need more space. see ByteBlockPool.LEVEL_SIZE_ARRAY
      final int upto = bytePool.newSlice(ByteBlockPool.FIRST_LEVEL_SIZE);
      termStreamAddressBuffer[streamAddressOffset + i] = upto + bytePool.byteOffset;
    }
    postingsArray.byteStarts[termID] = termStreamAddressBuffer[streamAddressOffset];


    newTerm(termID, docID);
  }
```

最初の２つのif文を見るとなにやらbufferのサイズをチェックして新しいバッファを使うかを決めている。ここまででわかる通り、```IntBlockPool```や```ByteBlockPool```は複数のバッファを管理している。

```java
public final class ByteBlockPool implements Accountable {
  // ...

  public byte[][] buffers = new byte[10][];

  /** index into the buffers array pointing to the current buffer used as the head */
  private int bufferUpto = -1; // Which buffer we are upto
  /** Where we are in head buffer */
  public int byteUpto = BYTE_BLOCK_SIZE;

  /** Current head buffer */
  public byte[] buffer;
  /** Current head offset */
  public int byteOffset = -BYTE_BLOCK_SIZE;
```

buffersが全てのbuffer(byteBlock)を保持しており、bufferUptoが現在参照しているbyteBlockのインデックスを保持している。そして現在の向き先のbyteblockは```buffer```という変数で保持されていることがわかる。

クラスのドキュメントを読むと最初のスライスは 5バイト、次のスライスは14バイトのように容量を増やしてながらbufferを作成しいくようです。```IntBlockPool```と```ByteBlockPool```でそれぞれ定義されているサイズは異なります。

```java
// in ByteBlockPool
public static final int[] LEVEL_SIZE_ARRAY = {5, 14, 20, 30, 40, 40, 80, 80, 120, 200};

// in IntBlockPool
private static final int[] LEVEL_SIZE_ARRAY = {2, 4, 8, 16, 32, 64, 128, 256, 512, 1024};
```

最初の5バイトに書き込んで、スライスの終わりに到達すると、新しいスライスのアドレスを前のスライスの最後の4バイトに書き込みむ。 

面白いのは各スライスは最初は 0 で埋められ、最後はゼロ以外のバイトでマークされる。このようにして、スライスに書き込んでいるメソッドが、その長さを追跡せずに、ゼロ以外のバイトにヒットしたら、代わりに新しいスライスを割り当てます。
