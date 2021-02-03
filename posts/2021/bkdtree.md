---
title: 楽しいBkd-Tree
cover: img/gopher.png
date: 2021/02/03
id: bkdtree
description: Elasticsearchの数値データ保管に使われるBkd-Treeというアルゴリズムの仕組みをまとめました。
tags:
    - Computer Science
    - Algorithm
draft: true
---

## Overview

Elasticsearch & Lucene 輪読会を弊社で毎週開催しているのですが、Codecを読んでいくと[Bkd-Tree](https://users.cs.duke.edu/~pankaj/publications/papers/bkd-sstd.pdf)というアルゴリズムに行き着きました。

論文はこちら
[Bkd-Tree: A Dynamic Scalable kd-Tree](https://users.cs.duke.edu/~pankaj/publications/papers/bkd-sstd.pdf)

Bkd-TreeはLucene6から導入されたようで下記のようにスペース効率、パフォーマンスが大幅に改善されたようです。

以下こちらの[Elasticsearch公式ブログ](https://www.elastic.co/jp/blog/elasticsearch-5-0-0-released#data-structures)の引用

>Lucene 6 の登場により、数値とgeo-pointフィールドにBlock K-D treesという新しいPointsデータ構造がもたらされ、数値データのインデキシングと検索の方法に革命が起きました。こちらのベンチマークでは、 Pointsはクエリ時間で36%、インデックス時間で71%速く、ディスク使用量が66%、メモリ使用量が85%もそれぞれ少ないことが分かっています

...まじか。すごいぞBkd-Tree!!!

そこで今回はBkd-Treeの仕組みをまとめました。ただいきなりBkd-Treeの説明から入っても難しいので、Bkd-Treeにつながる簡単なデータ構造から説明していきます。

## kd-Tree

kd-Tree(k-dimensional tree)はBSPに属するデータ構造です。以下のように軸を循環しながら木を構築していきます。一般的には、kd-treeの根ノードから葉ノードまでの各ノードには1つのポイント(N次元数値データ)が格納されます。図は[An Advanced k Nearest Neighbor Classification Algorithm Based on KD-tree](https://www.researchgate.net/publication/332434248_An_Advanced_k_Nearest_Neighbor_Classification_Algorithm_Based_on_KD-tree)から引用。

![kdtree](../../img/kdtree.png)

静的なkd-Treeの場合は効率が良いですが、木の回転などの標準的なバランシング手法を利用できないので、要素が追加される場合はバランスが保てない場合があります。

## K-D-B-tree

K-D-B-tree(k-dimensional B-tree)は外部メモリアクセスを最適化するために、B+Treeのブロック指向ストレージとkd-treeの検索効率を融合したものです。

数値データはツリーの葉に格納され、各リーフと内部ノードは1つのディスクブロックに格納されます。 下記はWikipediaからの図の引用です。K-D-B-treeの論文では内部ノードをRegion pages、葉をPoint pagesと表現しています。

![kdb](../../img/kdb.png)

木構造が浅くなり、大きなチャンクのデータを読み取ることができる為、B+TreeのようにディスクI/Oを最適化できます。

K-D-B-treeの大きな欠点は更新処理です。ある内部ノードを新しく分割する場合、その子ノードも新しく分割する必要が出てくるため非効率です。さらに分割によって疎な葉が生成される可能性があるため、スペース使用率が劇的に低下する可能性があります。下記はBkd-treeの論文の図の引用です。

![split-kdb](../../img/split-kdb.png)

その為、Elasticsearchのようにガンガン更新されるミドルウェアの場合は更新処理に最適化したデータ構造が必要です。

## Bkd-Tree

ここで出てくるのがBkd-Treeです。静的K-D-B-treeの高いストレージ使用率とクエリ効率を維持しながら、I/Oの更新を効率的に行うことが可能です。

Bkd-Treeはバランスの取れたkd-treeの集合で構成されています。Bkd-Treeで利用するkd-treeは内部ノードが完全な二分木であり、葉ノードはK-D-B-Treeと同じです。各kd-treeはディスクブロック上に格納されます。下記はBkd-Treeを構成する一つのkd-treeを表します。

![bkd-tree](../../img/bkd-tree.png)

* bulk loading algorithm
* logarithmic method

### bulk loading algorithm

普通のkd-treeはポイントを最初にソートしてルートからトップダウンで構築しますが、ここで1つのレベルを1個ずつ作成する代わりにまとめて木を構築していきます。

```md
Algorithm Bulk Load (grid)
(1) x,y軸ごとに2つのソートされたリストを作成
(2) 高さ log{2} t の高さの木を構築する
    (a) x,yそれぞれ直行するtグリッド線を計算する
    (b) グリッドセルのカウントを要素に持つグリッド行列Aを作成します。
    (c) グリッド行列を使って高さ log{2} tの木を作成
    (d) t個の葉に対応するように入力をt個に分割する
(3) 最下位レベルを構築するか、step(2) を再帰的に実行する。
```

上のアルゴリズムを一発で理解するのは多分無理なので1個ずつ見ていきます。
\(N\) がポイントの総数で、\(B\) がディスクブロックに収まるポイントの数、\(M\) がメモリバッファが格納できるポイントの数だとすると一回で上位レベル \( \log{2} t \) までサブツリーを作ります。ここで \(t\) は

\[
  t = Θ(min{M/B, \sqrt{M}})
\]

であると述べられています。そして t×t のグリッド線を引きます(下図のa)。ここからセルのが保持するポイント数を要素にもつ t×t グリッド行列Aを作成します。

![grid-cells-1](../../img/grid-cells1.png)

そして高さ \( \log{2} t \) の上位サブツリートップダウンアプローチを使用して（ステップ2-cで）計算できます。 

例えば、ルートノードが垂直な分割線を使用してポイントを分割する時を考えます。グリッド行列Aを使用して、分割線を含むセルの帯\( X_k \)を見つけます。その帯の中のポイントを分割する分割線を見つけます。

続いてその線を中心にグリッド行列Aを左右に分割し、左右それぞれのグリッド行列で再帰的に分割線を見つけていきます。グリッド行列Aの分割は帯\( X_k \)だけをスキャンするだけで決定できます。下記の図では帯\( X_k \)に含まれるセル\( C_{j,k} \)を分割しています。

![grid-cells-2](../../img/grid-cells2.png)

この処理をメモリバッファ\(M\) が埋まるまで、つまり\( \log{2} t \)のレベルまで一気に作成します。この時点で長方形の数が決定するのでサブツリーを利用してポイントをこれらの長方形に分配します（ステップ2-d）。 

### logarithmic method

![dynamic-update](../../img/dynamic-update.png)

## 参考

[Bkd-Tree: A Dynamic Scalable kd-Tree](https://users.cs.duke.edu/~pankaj/publications/papers/bkd-sstd.pdf)

[The Bkd Tree](https://medium.com/@nickgerleman/the-bkd-tree-da19cf9493fb)