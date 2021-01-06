---
title: 楽しいBkd-Tree
cover: img/gopher.png
date: 2021/01/08
id: kdtree
description: Go is a programming language
tags:
    - Computer Science
    - Algorithm
draft: true
---

## Overview

Elasticsearch & Lucene 輪読会を弊社で毎週開催しているのですが、Codecを読んでいくとBkd-Treeというアルゴリズムに行き着きました。今回は勉強に調べたBkd-Treeの仕組みをまとめました。

## BSTの復習

BST(Binary Search Tree)は左ノードに小さい要素、右側に大きいノードを格納する二分木です。

![bst](../../img/bst.png)

BSTの欠点は、k次元データの範囲検索です。k次元データの場合はkd-Treeを利用します。

## BSP

kd-treeの説明の前にBSPについて説明します。BSP(Binary space partitioning)はN次元をチャンクに分割していくデータ構造です。下記はWikiからの引用ですが、100個の3次元データを平面で分割したものです。

![bst](../../img/bsp.png)

## kd-Tree

kd-Tree(k-dimensional tree)はBSPに属するデータ構造です。BSTとの主な違いは、キーの比較が木のレベルによって異なることです。以下のように軸を循環しながら木を構築していきます。一般的には、kd-treeの根ノードから葉ノードまでの各ノードには1つの点が格納されます。

![bst](../../img/kdtree.png)

BSP木では分割平面の角度は任意ですが。kd-treeは、座標軸に垂直な平面だけを使って分割を行います。

静的なkd-Treeの場合は上手く機能しますが、木の回転などの標準的なバランシング手法を利用できないので、要素が追加される場合はバランスが保てない場合があります。

## K-D-B-tree

K-D-B-tree(k-dimensional B-tree)はk次元の検索空間を細分化するためのツリーデータ構造です。 外部メモリアクセスを最適化するために、B+Treeのブロック指向ストレージとkd-treeの検索効率を融合したものです。

![bst](../../img/kdb.png)

木構造が浅くなり、大きなチャンクのデータを読み取ることができる為、B+TreeのようにディスクI/Oを最適化できます。またバランシングも可能なので木構造を浅く保つことも可能です。

## Bkd-Tree

## 参考

[Bkd-Tree: A Dynamic Scalable kd-Tree](https://users.cs.duke.edu/~pankaj/publications/papers/bkd-sstd.pdf)

[The Bkd Tree](https://medium.com/@nickgerleman/the-bkd-tree-da19cf9493fb)