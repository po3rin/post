# VueCLIからVue.js入門①【VueCLIで出てくるファイルを概要図で理解】

![vue.001.jpeg](https://qiita-image-store.s3.amazonaws.com/0/186028/b0f820cb-216f-68c1-6f7c-d1045524318f.jpeg)

Vue.jsをVueCLIからSPA開発に入門してみます。VueCLIはVue.js開発できるようにいろいろ準備してくれるすげーやつ。

しかしいきなりVueCLIから入ると、訳が分からないファイルがいっぱい出てきて、理解するのに苦労します、てか苦労しました笑。

なので、僕がVueCLIを始めた時に知っておきたかったことを中心に、Vue.jsに入門していきます。Vue.jsの公式ドキュメントはCDN読み込みから入っているので、VueCLIから入門してVue.jsの威力を味わいっていきましょう。記事を三回に分けてVue.jsの基本からルーティングの実装まで行います。

今回はVueCLIを使って環境を整えます。

## 必要なインストール
node.jsをインストールしておいてください
インストールされているか下記で確認しましょう。

```bash
$ node -v
v9.5.0

$ npm -v
v5.6.0
```

## 1 Vueアプリケーションの雛形作成
Vue.jsの開発環境を整えるためにvue-cliを使います。まずはインストールしましょう

```bash
$ npm install -g vue-cli
```
これでVueアプリケーションの雛形が作成できます。早速下記を実行。test-vueの部分はプロジェクト名&ディレクトリ名になります。

```bash
$ vue init webpack test-vue
```

いろいろ聞かれますが、全部EnterでもOKです。実行が終わったら下記を実行し、雛形を確認してみましょう。

```bash
$ cd test-vue
$ npm run dev
```

localhost:8080にブラウザからアクセスしてみてください。Vueアプリケーションの土台ができています。

## 2 Vueアプリケーション雛形の中身確認

![vue.007.jpeg](https://qiita-image-store.s3.amazonaws.com/0/186028/85e76723-bb85-23fd-17a1-ace7713518e4.jpeg)

いろんなファイルが出来ててキョどります。でも大丈夫。基本的に我々が触るのは上の図で表したファイルだけです。難しい設定をしなければ、だいたいsrcディレクトリの中だけでいけてしまいます。早速各々のファイルが何をしているか見ていきましょう。

### index.html

下の様な記載があります。

```html:index.js
<body>
  <div id="app"></div>
  <!-- built files will be auto injected -->
</body>
```
ここにbuilt files will be auto injectedなる一文があります。webpackでビルドしたやつらが注入されるようです。もちろんmain.jsも読み込まれています。ではその```src/main.js```を見てみます。

### main.js

```js:main.js
import Vue from 'vue'
import App from './App'
import router from './router'

Vue.config.productionTip = false

/* eslint-disable no-new */
new Vue({
  el: '#app',
  router,
  components: { App },
  template: '<App/>'
})

```

ここでVueインスタンスが生成されています。ここを読むとindex.htmlの```id="app"```にAppという名前のコンポーネントをマウントしています。このAppというものは```import App from './App'```の部分で定義されています。それでは読み込まれているApp.vueを見てみましょう。

### App.vue

```vue:App.vue
<template>
  <div id="app">
    <img src="./assets/logo.png">
    <router-view/>
  </div>
</template>

<script>
export default {
  name: 'App'
}
</script>

<style>
#app {
  font-family: 'Avenir', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  margin-top: 60px;
}
</style>

```
これが Vue単一コンポーネントファイルと呼ばれるものです。拡張子が```.vue```になっています。```<template>```にhtmlを、```<script>```にjavascriptを、```<style>```にcssをそれぞれ一つのファイルに書くようになっています。そして何やら見慣れないのは```<router-view/>```の部分です。ここでは、vue-routerライブラリによる表示が行われます。vue-routerライブラリは、パスに応じて表示するコンポーネントを切り替えています。ではvue-routerがどこで設定されているかというと```src/router/index.js```です。

### router/index.js

```js:router/index.js
import Vue from 'vue'
import Router from 'vue-router'
import HelloWorld from '@/components/HelloWorld'

Vue.use(Router)

export default new Router({
  routes: [
    {
      path: '/',
      name: 'HelloWorld',
      component: HelloWorld
    }
  ]
})

```
ここでルーティングの設定をしています。pathで'/'にアクセスすると、先ほどの```<router-view/>```にHelloWorld.vueが表示されます。ではここでルーティングが設定されているHelloWorld.vueをみてみましょう。

### HelloWorld.vue

```vue:HelloWorld.vue
<template>
  <div class="hello">
    <h1>{{ msg }}</h1>
    <h2>Essential Links</h2>
  </div>
</template>

<script>
export default {
  name: 'hello',
  data () {
    return {
      msg: 'Welcome to Your Vue.js App'
    }
  }
}
</script>

<style scoped>
h1, h2 {
  font-weight: normal;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}
```

dataで持っているmsgの値を{{ msg }}で表示しています。また、Vue単一コンポーネントはコンポーネントスコープCSSという機能があり、```<style scoped>```の部分でスコープ宣言されてます。これによって、CSSが他のコンポーネントに影響を与えることなく分離し、CSSの管理がしやすくなります。

### 今回はここまで
以上でVueアプリケーションの雛形の簡単な確認は以上です。次回から実際にコードをいじっていきます。てか書いてて思ったんだけど、Qiitaがvueファイルのシンタックスハイライトをサポートしだしたっぽい。前からあった？


次回：[VueCLIからVue.js入門 ②【トグル機能作成からVue.jsの基本的な機能を掴む】 - Qiita](https://qiita.com/po3rin/items/15e1972ef5165b3725bf)

