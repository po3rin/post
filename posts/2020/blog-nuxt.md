---
title: Nuxt.js + Markdown でテックブログを作るハンズオン
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/teck-b-a.jpeg
date: 2020/05/10
id: blog-nuxt
description: Nuxt.js で シンタックスハイライト や 画像の lazy loading などを入れたブログの作り方を紹介します。
tags:
    - Nuxt.js
    - Vue.js
---

## Introduction

こんにちは。po3rinです。今回、テックブログを新しくNuxt.jsで作ったのでその方法を紹介します。
この記事ではNuxt.jsでブログを作る為の下記のTipsを紹介します。

* MarkdownをParseしてHTMLに変換する方法。
* Prism.jsでシンタックスハイライトをつける。
* パフォーマンスの為に画像のLazy loadingを入れる。

では早速実装していきましょう！

## 検証用Nuxt.jsアプリケーションの準備

雛形を作ります。設定は適当に。。

```bash
$ create-nuxt-app test-app
```

## MarkdownをParseしてHTMLに変換する

Markdownを J avaScript で Parse する際には拡張性の高い markdown-it が便利でしょう。

[![https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/mdit.png](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/mdit.png)](https://github.com/markdown-it/markdown-it)

```bash
$ yarn add markdown-it
```

そして ```plugins/markdownit.js``` を作成します。ここでは ```img``` タグに ***lazy loading*** の attribute を追加する設定を追加します。このように独自の拡張を加えれるのも ```markdown-it``` の特徴です。

```js
import MarkdownIt from 'markdown-it'

export default ({ app }, inject) => {
  const md = new MarkdownIt({
    // Your Basic Settings
  })

  const defaultRender =
    md.renderer.rules.image ||
    function(tokens, idx, options, env, self) {
      return self.renderToken(tokens, idx, options)
    }

  md.renderer.rules.image = function(tokens, idx, options, env, self) {
    tokens[idx].attrPush(['loading', 'lazy'])
    return defaultRender(tokens, idx, options, env, self)
  }

  inject('md', md)
}
```

## シンタックスハイライトを入れる

テックブログで重要なシンタックスハイライトの機能も入れましょう。ここでは有名な```Prism.js```を導入します。

[https://prismjs.com/](https://prismjs.com/)

```bash
$ yarn add prismjs
```

こちらも ```plugins/prism.js``` を作成します。ここではシンタックスハイライトのルールや、使うテーマを宣言しておきます。テーマは自分のブログにあったオシャレな物を選びましょう。デフォルトのテーマでも良いですが、追加のテーマもあるので、こちらも選択肢に入れて選んで見てください。

[https://github.com/PrismJS/prism-themes](https://github.com/PrismJS/prism-themes)

今回は ```prism-themes``` も使っていきましょう。

```bash
$ yarn add prism-themes
```

```js
import Prism from 'prismjs'

// テーマの追加
import 'prism-themes/themes/prism-nord.css'

// 言語別にシンタックスハイライトを追加
import 'prismjs/components/prism-go'
// ...

export default Prism
```

ちなみにサポートされているシンタックスハイライトはこちらで確認できます。

[https://prismjs.com/#supported-languages](https://prismjs.com/#supported-languages)

さて、これでプラグインの準備ができました。```nuxt.config.js``` に追加します。

```js
module.exports = {
    // ...
    plugins: ['~/plugins/prism', '~/plugins/markdownit'],
    // ...
}
```

## 動作確認
動作確認の為に ```pages/index.js``` を修正します。今回はdataプロパティでMarkdownを渡していますが、本番では```asyncData```で Markdown のデータをサーバーから引っ張ってくるのが良いでしょう。

```html
<template>
  <div class="container">
    <div>
      <div v-html="$md.render(post)"></div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      post: '# Goの変数宣言\n解説します。\n```go\nvar name = "go"\n```\n![]()\n'
    }
  },
  mounted() {
    Prism.highlightAll()
  },
}
</script>
```

これで ```yarn dev``` を実行してブラウザで確認すると下記のようにMarkdownがParseされているのが確認できます。もちろんGoのコードもシンタックスハイライトが付いています。画像は確認用で空で設定してありますが、```loading="lazy"``` が付与されているのも確認できます。

![https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/go-syntax.png](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/go-syntax.png)

あとはCSSでタグごとにデザインを整えて行けばテックブログの完成です！やったね！
