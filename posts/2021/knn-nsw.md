---
title: Lucene v9で登場するベクトルの近似最近傍検索をゆるくコードリーディング
cover: img/gopher.png
date: 2021/03/21
id: knn-nsw
description: Lucene v9 の近似最近某検索が待ちきれないので、ぶらりコードリーディング
tags:
    - Search Engine
    - Lucene
---

## Overview

こんにちは[po3rin](https://twitter.com/po3rin)です。この記事ではLuceneの次期メジャーバージョンで登場するベクトルの近似最近傍検索(ANN-search)のコードをゆるりと追ってみた記録です。

[[toc]]

## Luceneのベクトル近似最近傍検索

次期メジャーバージョンのLucene v9からNSWベースのベクトル近似最近傍検索 [1] がリリースされます。NSWやHNSWベースのANNの簡単な解説、Luceneへの導入背景に関しては、Luceneのコミッターである[@moco_beta](https://twitter.com/moco_beta)さんの[ブログ記事](https://mocobeta.medium.com/%E3%83%99%E3%82%AF%E3%83%88%E3%83%AB%E6%A4%9C%E7%B4%A2-%E8%BF%91%E4%BC%BC%E6%9C%80%E8%BF%91%E5%82%8D%E6%8E%A2%E7%B4%A2-%E3%81%A7%E3%81%84%E3%81%84%E6%84%9F%E3%81%98%E3%81%AE-morelikethis-%E3%82%92%E5%AE%9F%E7%8F%BE%E3%81%99%E3%82%8B-7eba63ffb593) [2] が最高なのでこちらをどうぞ！

## NSWベース近似最近傍検索の実装を読む

まずは実装が提案されている [Issue](https://issues.apache.org/jira/browse/LUCENE-9004) [3] をみてみます。ここからリンクされている下記のPRを読んでいけばどんな実装になっているか理解できそうです。

* [LUCENE-9004: KNN vector search using NSW graphs #2022](https://github.com/apache/lucene-solr/pull/2022)

まずは使い方をザッとさらうために実際の検索テストを見てみます。

```java
public void testSearch() throws Exception {
    try (Directory dir = newDirectory();
         IndexWriter iw = new IndexWriter(dir, new IndexWriterConfig().setCodec(Codec.forName("Lucene90")))) {
      
      // ...
      int n = 5, stepSize = 17;
      float[][] values = new float[n * n][];
      int index = 0;
      for (int i = 0; i < values.length; i++) {
        values[i] = new float[]{index % n, index / n};
        index = (index + stepSize) % (n * n);
        add(iw, i, values[i]);
        if (i == 13) {
          iw.commit();
        }
      }
      boolean forceMerge = random().nextBoolean();
      if (forceMerge) {
        iw.forceMerge(1);
      }
      assertConsistentGraph(iw, values);
      try (DirectoryReader dr = DirectoryReader.open(iw)) {
        // results are ordered by score (descending) and docid (ascending);
        // This is the insertion order:
        // column major, origin at upper left
        //  0 15  5 20 10
        //  3 18  8 23 13
        //  6 21 11  1 16
        //  9 24 14  4 19
        // 12  2 17  7 22

        // For this small graph the "search" is exhaustive, so this mostly tests the APIs, the orientation of the
        // various priority queues, the scoring function, but not so much the approximate KNN search algo
        assertGraphSearch(new int[]{0, 15, 3, 18, 5}, new float[]{0f, 0.1f}, dr);
        // ...
      }
    }
  }
```

ベクトルの保存、検索がそれぞれ```add```,```assertGraphSearch```という別関数に切り出されていることがわかります。ここからは2021/3/23時点のmasterの実装を見ていきます。

## HNSWを使ったグラフ保存

```add```を見るといつも通り```Document```クラスを初期化して```IndexWriter.addDocument```に渡しています。```createHnswType```というHNSWを使う```FieldType```を生成するメソッドが見つかります。

```java
  private static final String KNN_GRAPH_FIELD = "vector";

  // ...

  private void add(IndexWriter iw, int id, float[] vector) throws IOException {
    add(iw, id, vector, searchStrategy);
  }

  private void add(IndexWriter iw, int id, float[] vector, SearchStrategy searchStrategy)
      throws IOException {
    Document doc = new Document();
    if (vector != null) {
      FieldType fieldType =
          VectorField.createHnswType(
              vector.length, searchStrategy, maxConn, HnswGraphBuilder.DEFAULT_BEAM_WIDTH);
      doc.add(new VectorField(KNN_GRAPH_FIELD, vector, fieldType));
    }
    String idString = Integer.toString(id);
    doc.add(new StringField("id", idString, Field.Store.YES));
    doc.add(new SortedDocValuesField("id", new BytesRef(idString)));
    // XSSystem.out.println("add " + idString + " " + Arrays.toString(vector));
    iw.updateDocument(new Term("id", idString), doc);
  }
```

ちなみに```SearchStrategy```は```enum```になっており、HNSWに使う"距離"をユークリッド距離、ドット積の中から選べる。

```java
 public enum SearchStrategy {

    /**
     * No search strategy is provided. Note: {@link VectorValues#search(float[], int, int)} is not
     * supported for fields specifying this strategy.
     */
    NONE,

    /** HNSW graph built using Euclidean distance */
    EUCLIDEAN_HNSW(true),

    /** HNSW graph buit using dot product */
    DOT_PRODUCT_HNSW;
 }
```

```createHnswType```の中身は```FieldType```の生成につとめている。```createHnswType```の引数のドキュメントがあるのでここ読むとなんとなく引数の意味が見えてくる。

```java
  /**
   * Public method to create HNSW field type with the given max-connections and beam-width
   * parameters that would be used by HnswGraphBuilder while constructing HNSW graph.
   *
   * @param dimension dimension of vectors
   * @param searchStrategy a function defining vector proximity.
   * @param maxConn max-connections at each HNSW graph node
   * @param beamWidth size of list to be used while constructing HNSW graph
   * @throws IllegalArgumentException if any parameter is null, or has dimension &gt; 1024.
   */
  public static FieldType createHnswType(
      int dimension, VectorValues.SearchStrategy searchStrategy, int maxConn, int beamWidth) {
  
    // 引数の validation ...

    FieldType type = new FieldType();
    type.setVectorDimensionsAndSearchStrategy(dimension, searchStrategy);
    type.putAttribute(HnswGraphBuilder.HNSW_MAX_CONN_ATTRIBUTE_KEY, String.valueOf(maxConn));
    type.putAttribute(HnswGraphBuilder.HNSW_BEAM_WIDTH_ATTRIBUTE_KEY, String.valueOf(beamWidth));
    type.freeze();
    return type;
  }
```

```maxConn```はHNSWのノードの最大エッジ数、```beamWidth```はここまで読んだだけではよくわからないが、後述する```HnswGraph```クラスのドキュメントにそれぞれの引数が論文の何に対応するかが書いてある。

では実際にindexする処理を追っていきたい。前回記事でかいた[読んで理解する全文検索 (IndexWriter, DWPT, IndexingChain 導入編)](https://po3rin.com/blog/indexwriter-internal)の知識から```IndexChain```まで一気に掘れることが予想できる。実際に```processField```関数でベクトルの処理を分岐させている。

```java
  private int processField(int docID, IndexableField field, long fieldGen, int fieldCount)
      throws IOException {
    String fieldName = field.name();
    IndexableFieldType fieldType = field.fieldType();

    PerField fp = null;

    // その他フィールドタイプ...

    if (fieldType.vectorDimension() != 0) {
      if (fp == null) {
        fp = getOrAddField(fieldName, fieldType, false);
      }
      indexVector(docID, fp, field);
    }

    return fieldCount;
  }
```

では```indexVector```を覗く。中では```vectorValuesWriter```による書き込みに処理を渡している。

```java
private void indexVector(int docID, PerField fp, IndexableField field) {
    int dimension = field.fieldType().vectorDimension();
    VectorValues.SearchStrategy searchStrategy = field.fieldType().vectorSearchStrategy();

    // ...
    if (fp.fieldInfo.getVectorDimension() == 0) {
      fieldInfos.globalFieldNumbers.setVectorDimensionsAndSearchStrategy(
          fp.fieldInfo.number, fp.fieldInfo.name, dimension, searchStrategy);
    }
    fp.fieldInfo.setVectorDimensionAndSearchStrategy(dimension, searchStrategy);

    if (fp.vectorValuesWriter == null) {
      fp.vectorValuesWriter = new VectorValuesWriter(fp.fieldInfo, bytesUsed);
    }
    fp.vectorValuesWriter.addValue(docID, ((VectorField) field).vectorValue());
  }
```

さらに掘っていくととりあえずメモリに乗っけたところで処理が止まる。

```java
class VectorValuesWriter {
  // ...

  public void addValue(int docID, float[] vectorValue) {
    // validation ...

    assert docID > lastDocID;
    docsWithField.add(docID);
    vectors.add(ArrayUtil.copyOfSubArray(vectorValue, 0, vectorValue.length));
    updateBytesUsed();
    lastDocID = docID;
  }

  private void updateBytesUsed() {
    final long newBytesUsed =
        docsWithField.ramBytesUsed()
            + vectors.size()
                * (RamUsageEstimator.NUM_BYTES_OBJECT_REF
                    + RamUsageEstimator.NUM_BYTES_ARRAY_HEADER)
            + vectors.size() * vectors.get(0).length * Float.BYTES;
    if (iwBytesUsed != null) {
      iwBytesUsed.addAndGet(newBytesUsed - bytesUsed);
    }
    bytesUsed = newBytesUsed;
  }

  // ...
}
```

ここでflushフェーズを探すことになるが、すぐ下にあるので見つけやすい。ここでは永続化のためにLucene Codecに処理が渡っていることがわかる。

```java
import org.apache.lucene.codecs.VectorWriter;

// ...

class VectorValuesWriter {
  // ...

  /**
   * Flush this field's values to storage, sorting the values in accordance with sortMap
   *
   * @param sortMap specifies the order of documents being flushed, or null if they are to be
   *     flushed in docid order
   * @param vectorWriter the Codec's vector writer that handles the actual encoding and I/O
   * @throws IOException if there is an error writing the field and its values
   */
  public void flush(Sorter.DocMap sortMap, VectorWriter vectorWriter) throws IOException {
    VectorValues vectorValues =
        new BufferedVectorValues(
            docsWithField,
            vectors,
            fieldInfo.getVectorDimension(),
            fieldInfo.getVectorSearchStrategy());
    if (sortMap != null) {
      vectorWriter.writeField(fieldInfo, new SortingVectorValues(vectorValues, sortMap));
    } else {
      vectorWriter.writeField(fieldInfo, vectorValues);
    }
  }
```

ついにCodecにたどり着いた。テストケースのコードを見るとCodec名に```Lucene90```を指名していたので```lucene/core/src/java/org/apache/lucene/codecs/lucene90/Lucene90VectorWriter.java```まで読みにいく。めちゃくちゃHNSWを使ってることを示唆するimportがある。やっとアルゴリズムの始まりまできた。読んでいくといつものようにvectorを保存すると同時にHNSWを利用する場合は更にグラフ構築して保存していることがわかる。

```java
// ...
import org.apache.lucene.util.hnsw.HnswGraph;
import org.apache.lucene.util.hnsw.HnswGraphBuilder;
import org.apache.lucene.util.hnsw.NeighborArray;

// ...

public final class Lucene90VectorWriter extends VectorWriter {
  @Override
  public void writeField(FieldInfo fieldInfo, VectorValues vectors) throws IOException {
    long pos = vectorData.getFilePointer();
    // write floats aligned at 4 bytes. This will not survive CFS, but it shows a small benefit when
    // CFS is not used, eg for larger indexes
    long padding = (4 - (pos & 0x3)) & 0x3;
    long vectorDataOffset = pos + padding;
    for (int i = 0; i < padding; i++) {
      vectorData.writeByte((byte) 0);
    }
    // TODO - use a better data structure; a bitset? DocsWithFieldSet is p.p. in o.a.l.index
    int[] docIds = new int[vectors.size()];
    int count = 0;
    for (int docV = vectors.nextDoc(); docV != NO_MORE_DOCS; docV = vectors.nextDoc(), count++) {
      // write vector
      writeVectorValue(vectors);
      docIds[count] = docV;
    }
    // count may be < vectors.size() e,g, if some documents were deleted
    long[] offsets = new long[count];
    long vectorDataLength = vectorData.getFilePointer() - vectorDataOffset;
    long vectorIndexOffset = vectorIndex.getFilePointer();
    if (vectors.searchStrategy().isHnsw()) {
      if (vectors instanceof RandomAccessVectorValuesProducer) {
        writeGraph(
            vectorIndex,
            (RandomAccessVectorValuesProducer) vectors,
            vectorIndexOffset,
            offsets,
            count,
            fieldInfo.getAttribute(HnswGraphBuilder.HNSW_MAX_CONN_ATTRIBUTE_KEY),
            fieldInfo.getAttribute(HnswGraphBuilder.HNSW_BEAM_WIDTH_ATTRIBUTE_KEY));
      } else {
        // ...
      }
    }
    long vectorIndexLength = vectorIndex.getFilePointer() - vectorIndexOffset;
    if (vectorDataLength > 0) {
      writeMeta(
        // ...metadata...
      );
      if (vectors.searchStrategy().isHnsw()) {
        writeGraphOffsets(meta, offsets);
      }
    }
  }
}
```

今回はHNSWの実装を追っているので```writeGraph```関数を見てみる。ここをみると、```HnswGraphBuilder```でグラフを構築し、その情報を```IndexOutput.writeXXX```で書き出していることが分かる。

```java
public final class Lucene90VectorWriter extends VectorWriter {
  private void writeGraph(
      IndexOutput graphData,
      RandomAccessVectorValuesProducer vectorValues,
      long graphDataOffset,
      long[] offsets,
      int count,
      String maxConnStr,
      String beamWidthStr)
      throws IOException {
    int maxConn, beamWidth;
    // maxConn と beamWidth の デフォルト値セット ...

    HnswGraphBuilder hnswGraphBuilder =
        new HnswGraphBuilder(vectorValues, maxConn, beamWidth, HnswGraphBuilder.randSeed);
    hnswGraphBuilder.setInfoStream(segmentWriteState.infoStream);
    HnswGraph graph = hnswGraphBuilder.build(vectorValues.randomAccess());

    for (int ord = 0; ord < count; ord++) {
      // write graph
      offsets[ord] = graphData.getFilePointer() - graphDataOffset;

      NeighborArray neighbors = graph.getNeighbors(ord);
      int size = neighbors.size();

      // Destructively modify; it's ok we are discarding it after this
      int[] nodes = neighbors.node();
      Arrays.sort(nodes, 0, size);
      graphData.writeInt(size);

      int lastNode = -1; // to make the assertion work?
      for (int i = 0; i < size; i++) {
        int node = nodes[i];
        assert node > lastNode : "nodes out of order: " + lastNode + "," + node;
        assert node < offsets.length : "node too large: " + node + ">=" + offsets.length;
        graphData.writeVInt(node - lastNode);
        lastNode = node;
      }
    }
  }
}
```

ここで力尽きた。。次はwriteXXX系の中身を見ていく。

## References

[1] [Y. Malkov, A. Ponomarenko, A. Logvinov, and V. Krylov, "Approximate nearest neighbor algorithm based on navigable small
world graphs," Information Systems, vol. 45, pp. 61-68, 2014.](https://publications.hse.ru/mirror/pubs/share/folder/x5p6h7thif/direct/128296059)

[2] [ベクトル検索（近似最近傍探索）でいい感じの MoreLikeThis を実現する](https://mocobeta.medium.com/%E3%83%99%E3%82%AF%E3%83%88%E3%83%AB%E6%A4%9C%E7%B4%A2-%E8%BF%91%E4%BC%BC%E6%9C%80%E8%BF%91%E5%82%8D%E6%8E%A2%E7%B4%A2-%E3%81%A7%E3%81%84%E3%81%84%E6%84%9F%E3%81%98%E3%81%AE-morelikethis-%E3%82%92%E5%AE%9F%E7%8F%BE%E3%81%99%E3%82%8B-7eba63ffb593)
