# Nuxt.js でスクロールでふわっと要素が出現するやつを「カスタムディレクティブ」で実装する

Nuxt.js でWEBサイト作っていると、スクロールで何か処理をしたいことがある。
そんなときはカスタムディレクティブが便利。特にスクロールでふわっと要素が出てくる系は、様々な箇所で同じアニメーションロジックを再利用する為、カスタムディレクティブは相性が良いです。

## Nuxt.jsでの実装上でのポイント

ポイントはドキュメントにも書いてある通り、「カスタムディレクティブは Vue インスタンス作成前に登録されなければならない」という点。Nuxt.js では Vue インスタンス作成前に実行したい処理は plugins ディレクトリがその責務を担います。```plugins/README.md``` に書いてある通りです。

> This directory contains your Javascript plugins that you want to run before instantiating the root vue.js application.

なので、まずは```plugins```ディレクトリの中で```scroll.js```を作成し、カスタムディレクティブを作りましょう。

```js:plugins/scroll.js
import Vue from 'vue'

Vue.directive('scroll', {
    inserted: function (el, binding) {
        let f = function (evt) {
            if (binding.value(evt, el)) {
                window.removeEventListener('scroll', f)
            }
        }
        window.addEventListener('scroll', f)
    }
})
```

コードはVueの公式ドキュメントから拝借しました。
[カスタムスクロールディレクティブの作成](https://jp.vuejs.org/v2/cookbook/creating-custom-scroll-directives.html)

```inserted``` はひも付いている要素が親 Node に挿入された時に呼ばれるもの。
他のカスタムディレクティブのフック関数は下記ドキュメント参照。
[カスタムディレクティブ](https://jp.vuejs.org/v2/guide/custom-directive.html)

そして、```nuxt.config.js```内の plugins に作成した ファイルへのパスを追加します。

```js:nuxt.config.js
module.exports = {
  // ~
  plugins: [
    '~plugins/scroll.js'
  ]
}
```

これで```v-scroll```というカスタムディレクティブがtemplate内で使えるようになります。

## 作ったカスタムディレクティブを使ってみる

もしあなたが　Nuxt.js　のプロジェクトを作りたてなら```pages/index.vue```で試して見ましょう。

```vue:pages/index.vue
<template>
  <section class="container">
    <div>
      <div class="box" v-scroll="handleScroll">scroll hock</div>
    </div>
  </section>
</template>

<script>
export default {
  methods: {
    handleScroll: function(evt, el) {
      console.log(window.scrollY);
      if (window.scrollY > 50) {
        el.setAttribute(
          "style",
          "opacity: 1; transform: translate3d(0, -10px, 0)"
        );
      }
      return window.scrollY > 100;
    }
  }
};
</script>

<style>
.container {
  min-height: 150vh;
  display: flex;
  justify-content: center;
  align-items: center;
  text-align: center;
}

.box {
  opacity: 0;
  transition: 1.5s all cubic-bezier(0.39, 0.575, 0.565, 1);
}
</style>
```

v-scrollで発火するメソッドを指定しています。
これでスクロールイベントを```v-scroll```と宣言するだけで行えるようになりました。

