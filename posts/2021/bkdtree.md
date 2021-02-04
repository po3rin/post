---
title: 検索エンジンの数値インデックスを支える Bkd-Tree
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/bkdtree.jpeg
date: 2021/02/04
id: bkdtree
description: Elasticsearchの数値データインデックスに使われるBkd-Treeというアルゴリズムを論文を読んでまとめました。
tags:
    - Computer Science
    - Algorithm
---

## Overview

こんにちは [pon](https://twitter.com/po3rin) です。Elasticsearch & Lucene 輪読会を弊社で毎週開催しているのですが、そこで[Bkd-Tree](https://users.cs.duke.edu/~pankaj/publications/papers/bkd-sstd.pdf)というアルゴリズムに行き着きました。そこでBkd-Treeの論文を読んでみたので、まとめたものを共有しようと思います。

論文はこちら
[Bkd-Tree: A Dynamic Scalable kd-Tree](https://users.cs.duke.edu/~pankaj/publications/papers/bkd-sstd.pdf)

[[toc]]

## LuceneでのBkd-Tree

Bkd-TreeはLucene6から導入されたようで下記のようにスペース効率、パフォーマンスが大幅に改善されたようです。

以下こちらの[Elasticsearch公式ブログ](https://www.elastic.co/jp/blog/elasticsearch-5-0-0-released#data-structures)の引用

>Lucene 6 の登場により、数値とgeo-pointフィールドにBlock K-D treesという新しいPointsデータ構造がもたらされ、数値データのインデキシングと検索の方法に革命が起きました。こちらのベンチマークでは、 Pointsはクエリ時間で36%、インデックス時間で71%速く、ディスク使用量が66%、メモリ使用量が85%もそれぞれ少ないことが分かっています

...まじか。すごいぞBkd-Tree!!!

ただいきなりBkd-Treeの説明から入っても難しいので、Bkd-Treeにつながる簡単なデータ構造から説明していきます。

## kd-Tree

kd-Tree(k-dimensional tree)は以下の図のように軸を循環しながら木を構築していきます。一般的には、kd-treeの根ノードから葉ノードまでの各ノードには1つのポイント(N次元数値データ)が格納されます。図は[An Advanced k Nearest Neighbor Classification Algorithm Based on KD-tree](https://www.researchgate.net/publication/332434248_An_Advanced_k_Nearest_Neighbor_Classification_Algorithm_Based_on_KD-tree)から引用。

![kdtree](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/kdtree.png)

静的なkd-Treeの場合は効率が良いですが、木の回転などの標準的なバランシング手法を利用できないので、要素が追加される場合はバランスが保てない場合があります。

## K-D-B-tree

K-D-B-tree(k-dimensional B-tree)は外部メモリアクセスを最適化するために、B+Treeのブロック指向ストレージとkd-treeの検索効率を融合したものです。

数値データはツリーの葉に格納され、各リーフと内部ノードは1つのディスクブロックに格納されます。 下記はWikipediaからの図の引用です。K-D-B-treeの論文では内部ノードをRegion pages、葉をPoint pagesと表現しています。

![kdb](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/kdb.png)

木構造が浅くなり、大きなチャンクのデータを読み取ることができる為、B+TreeのようにディスクI/Oを最適化できます。

K-D-B-treeの大きな欠点は更新処理です。ある内部ノードを新しく分割する場合、その子ノードも新しく分割する必要が出てくるため非効率です。さらに分割によって疎な葉が生成される可能性があるため、スペース使用率が劇的に低下する可能性があります。下記はBkd-treeの論文の図の引用です。

![split-kdb](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/split-kdb.png)

その為、Elasticsearchのようにガンガン更新されるミドルウェアの場合は更新処理に最適化し、かつI/O効率を意識したデータ構造が必要です。

## Bkd-Tree

ここで出てくるのがBkd-Treeです。静的K-D-B-treeの高いストレージ使用率とクエリ効率を維持しながら、I/Oの更新を効率的に行うことが可能です。

Bkd-Treeはバランスの取れたkd-treeの集合で構成されています。Bkd-Treeで利用するkd-treeは内部ノードが完全な二分木であり、葉ノードはK-D-B-Treeと同じです。各kd-treeはディスクブロック上に格納されます。下記はBkd-Treeを構成する一つのkd-treeを表します。

![bkd-tree](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/bkd-tree.png)

複数のkd-treeを使うことで効率的な動的アップデートを実現します。そのためにBkd-treeの2つの工夫点を抑える必要があります。

* bulk loading algorithm
* logarithmic method

### bulk loading algorithm

普通のkd-treeはポイントを最初にソートしてルートからトップダウンで構築しますが、ここで1つのレベルを1個ずつ作成する代わりにまとめて木を構築していきます。

```go
Algorithm Bulk Load (grid)
(1) x,y軸ごとに2つのソートされたリストを作成
(2) 高さ log_2 t の高さの木を構築する
    (a) x,yそれぞれ直行するtグリッド線を計算する
    (b) グリッドセルのカウントを要素に持つグリッド行列Aを作成します。
    (c) グリッド行列を使って高さ log_2 tの木を作成
    (d) t個の葉に対応するように入力をt個に分割する
(3) 最下位レベルを構築するか、step(2) を再帰的に実行する。
```

上のアルゴリズムを一発で理解するのは多分無理なので1個ずつ見ていきます。
$N$ がポイントの総数で、$B$ がディスクブロックに収まるポイントの数、$M$ がメモリバッファが格納できるポイントの数だとすると一回で上位レベル $\log_2 t$ までサブツリーを作ります。ここで $t$ はポイント

\[
  t = Θ(min{M/B, \sqrt{M}})
\]

であると述べられています。そして t×t のグリッド線を引きます(下図のa)。ここからセルのが保持するポイント数を要素にもつ t×t グリッド行列Aを作成します。

![grid-cells-1](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/grid-cells1.png)

そして高さ $log_2 t$ の上位サブツリートップダウンアプローチを使用して（ステップ2-cで）計算できます。 

例えば、ルートノードが垂直な分割線を使用してポイントを分割する時を考えます。グリッド行列Aを使用して、分割線を含むセルの帯 $X_k$ を見つけます。その帯の中のポイントを分割する分割線を見つけます。

続いてその線を中心にグリッド行列Aを左右に分割し、左右それぞれのグリッド行列で再帰的に分割線を見つけていきます。グリッド行列Aの分割は帯 $X_k$ だけをスキャンするだけで決定できます。下記の図では帯 $X_k$ に含まれるセル$C_{j,k}$を分割しています。

![grid-cells-2](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/grid-cells2.png)

この処理をメモリバッファ $M$ が埋まるまで、つまり $\log_2 t$ のレベルまで一気に作成します。この時点で最終的に分割される長方形の数が決定するのでサブツリーを利用してポイントをこれらの長方形に分配します（ステップ2-d）。

これによりルートから毎回ソートしながらトップダウンで構築する代わりにグリッド行列を使って一気に木を構築する仕組みがわかりました。この bulk loading algorithm は更新処理の中で次に説明する ***logarithmic method*** と合わせて利用します。

### logarithmic method

Bkd-Treeでは ***logarithmic method*** という方法を使って動的なアップデートを改善します。

Bkd-Treeでは最大 $\log_2 (N / M)$ 個のkd-treeで構成されます。i番目のkdツリー $T_i$ は、空であるか、$2^i×M$ ポイントを含みます。 したがって、$T_0$ は最大でMポイントを格納できます。さらに、$T^{M}_{0}$ は内部メモリに保持されます。下記は論文からの引用です。

![dynamic-update](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/dynamic-update.png)

例えばポイントの削除では各ツリーに並行にクエリを実行して、ポイントを含むツリー $T_i$ を見つけ、$T_i$ からポイントを削除します。 最大で $\log_2 (N / M)$ ツリーがあるため、削除によって実行されるI/O回数は $O(log_B (N / B)log_2 (N / M))$ となります。

```go
Algorithm Delete(p)
(1) T^{M}_{0} にクエリを投げる。そこにポイントpがあれば削除する 
(2) 空じゃない T_i に対してクエリを投げてポイントpがあれば削除する
```

挿入アルゴリズムはメモリ内の $T^{M}_{0}$ に対して直接実行されます。 $T^{M}_{0}$ に格納するポイント数がいっぱいになると、空の $T_k$ kdツリーのなかで $k$ が最小のものを見つけ、 次に、$T^{M}_{0}$ と $T_i ( 0 	\leq i 	< k )$ のkd-treeのポイントを全て抽出し、前節で説明した ***bulk loading*** を実行します。

 $T^{M}_{0}$ は $M$ ポイントを格納し、各 $T_i ( 0 \leq i 	< k )$ は $2^iM$ ポイントを格納している為、bulk loading後の $T_k$ に格納されているポイントの数は $2^kM$ になります。 最後に、$T^{M}_{0}$ と $T_i ( 0 \leq i 	< k )$を空にします 。 
 
つまり、ポイントはメモリ内構造に挿入され、小さなkd-treeを、1つの大きなkd-treeに定期的に再編成することにより、大きなkdツリーに向かって徐々に統合されていきます。

## パフォーマンス

K-D-B-treeとBkd-treeのポイント挿入クエリを比べたが論文にあります。(a)がポイント数に対する実行時間、(b)がポイントすうに対するI/O回数です。どちらも2桁ほど改善できています。

![bkd-tree-@erformance](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/performance.png)

## LuceneでのBkd-Tree実装

Elasticsearch内部、LuceneではBkd-Treeの実装があります。
[Package org.apache.lucene.util.bkd](https://lucene.apache.org/core/8_8_0/core/org/apache/lucene/util/bkd/package-summary.html)

## まとめ

今回はkd-treeのからK-D-B-treeの紹介を経て、Bkd-Treeを説明しました。解釈が違うところがもしあればご指摘いただければ幸いです。

### 参考

[Bkd-Tree: A Dynamic Scalable kd-Tree](https://users.cs.duke.edu/~pankaj/publications/papers/bkd-sstd.pdf)

[The Bkd Tree](https://medium.com/@nickgerleman/the-bkd-tree-da19cf9493fb)

