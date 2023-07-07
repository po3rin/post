---
title: Elastic Learned Sparse Encoder(ELSE)によるセマンティック検索へのゼロコスト移行
cover: img/gopher.png
date: 2023/03/05
id: m3-cruft
description: Go is a programming language
tags:
    - golang
    - markdown
draft: true
---

エムスリーエンジニアリンググループ AI・機械学習チームでソフトウェアエンジニアをしている中村([po3rin](https://twitter.com/po3rin)) です。検索とGoが好きです。

今回はElasticsearch8.8で利用可能になったElastic Learned Sparse Encoder(ELSE) をElastic Cloudで利用してみたので、RRFのロジックの説明と、触ってみた初感をお伝します。

## ELSE とは

## Elastic CloudでのELSE利用

RRFは上田さんのブログによるとElastic Cloudのプラチナプランであれば利用できるようです。エムリーではElastic Cloudのプラチナプランを契約しているので、すぐに利用できます。

https://shunyaueta.com/posts/2023-06-02-2323/#fn:3

今回はBM25による検索とベクトル検索の結果をRRFで結合してみます。

## ELSEとBM25はRFFで統合できない？