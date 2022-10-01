---
title: Rust製パターンマッチングマシンDaachorseをPythonから利用して30%高速化する話
cover: img/gopher.png
date: 2022/09/25
id: m3-daachorse-py
description: Go is a programming language
tags:
    - Python
    - Rust
draft: true
---

エムスリーエンジニアリンググループ AI・機械学習チームでソフトウェアエンジニアをしている中村([po3rin](https://twitter.com/po3rin)) です。検索とGoが好きです。

今回は文字列界隈を賑わせている高速なパターンマッチングマシン Daachorse（ダークホース）を使って文字列パターンマッチを30%高速化したお話をします。

<!-- more -->

[:contents]

## Daachorseとは

LegalForce Researchで開発運用している文字列パターンマッチを行うRust製ライブラリです。

https://github.com/daac-tools/daachorse

技術的なトピックに関してはLegalForceさんの記事が完全体なので、そちらを参照してください。ダブル配列の改善などウハウハな話題で盛りだくさんです。

https://tech.legalforce.co.jp/entry/2022/02/24/140316

## なぜPythonから呼び出したいのか

AI・機械学習チームではデータ処理やモデル学習のパイプラインにgokartというモジュールを利用しており、基本的に何かを実装するときはPythonで開発されることが多いです。
実際にPython + CookicutterでLinterやCIの用意をスキップしすぐにロジックの開発を行えるような環境が整っています。

そのため、既存のデータ処理、モデル学習のコードはPyhtonで書かれており、とあるロジックがパフォーマンスのボトルネックになってしまうことが多々あります。
私自身も文字列パターンマッチのロジックを組んでいたのですが、いまいち高速化できずに苦しんでいたところDaachorseのリリースがあり、是非とも使ってみたいと胸に希望を抱きました。


しかし、gokartというパイプラインにデータ処理が乗っている以上。全てをRustで書き換えるのは時間のかかる工事です。そこでロジック部分だけRustで書き直して高速化できないかと考えました。

## PyO3でDaachorseをPythonで利用する

Rustの処理をPythonで呼び出すための方法はいくつかあり、その中でも開発の活発でドキュメントも充実しているPyO3を利用しました。

https://github.com/PyO3/pyo3

まずはRust側の準備です。既存のプロジェクトで始めるためのガイドは下記のURLから参照できます。

https://pyo3.rs/v0.17.1/getting_started.html#adding-to-an-existing-project


まずは既存PythonプロジェクトのリポジトリでRustのプロジェクトを初期化します。

```sh
cargo new --lib
```

Cargo.tomlに下記の記述を追加します。

```toml
[package]
name = "daachorsepyo3"
version = "0.1.0"
edition = "2021"

# See more keys and their definitions at https://doc.rust-lang.org/cargo/reference/manifest.html

[lib]
# The name of the native library. This is the name which will be used in Python to import the
# library (i.e. `import string_sum`). If you change this, you must also change the name of the
# `#[pymodule]` in `src/lib.rs`.
name = "daachorsepyo3"

# "cdylib" is necessary to produce a shared library for Python to import from.
crate-type = ["cdylib"]

[dependencies]
pyo3 = { version = "0.17.1", features = ["extension-module"] }
daachorse = "1.0.0"
```

そして、src/lib.rsでDaachorseの処理を記述します。ヒットさせたい文字列パターンと検索対象のテキストをリストで受け取って、それぞれヒットしたパターンのindexを返すようにしています。

```rust
use daachorse::CharwiseDoubleArrayAhoCorasick;
use pyo3::prelude::*;

#[pyfunction]
fn substring_match(text_list: Vec<String>, patterns: Vec<String>) -> PyResult<Vec<Vec<i32>>> {
    let result = find_iter_with_charwise(text_list, patterns);
    Ok(result)
}

#[pymodule]
fn daachorsepyo3(_py: Python<'_>, m: &PyModule) -> PyResult<()> {
    m.add_function(wrap_pyfunction!(substring_match, m)?)?;
    Ok(())
}

fn find_iter_with_charwise(text_list: Vec<String>, patterns: Vec<String>) -> Vec<Vec<i32>> {
    let pma = CharwiseDoubleArrayAhoCorasick::new(patterns).unwrap();

    let mut result: Vec<Vec<i32>> = Vec::new();
    for text in text_list {
        let it = pma.find_iter(text);
        let vec: Vec<i32> = it.map(|i| i.value()).collect();
        result.push(vec);
    }
    result
}
```

PyO3の使い方はシンプルで`#[pymodule]`のデコレータを付与することでこの関数がPythonのモジュールにになります。そして、`#[pyfunction]`デコレータを付与した関数をPyModule.add_functionメソッドに渡すことでモジュール内の関数として`substring_match`が定義されます。簡単ですね。

最後にPyO3のチュートリアルにもあるようにmaturinというツールを使ってビルドします。 maturinはRustベースのPythonパッケージをビルドおよび公開を簡単に実行できる便利ツールです。

```sh
// Install the crate as module in the current virtualenv
poetry run maturin develop
```

これで.venv配下にRustで実装したCrateがPythonモジュールとして読み込まれます。これで下記のように実行できます。

```
from daachorsepyo3 import substring_match

text_list = ["全世界の子供たちが", "世界は広い"]
patterns = ["世界", "子供"]

result = substring_match(
    text_list,
    patterns
)

for i, r in enumerate(result):
    print('----------')
    print(text_list[i])
    print([patterns[pattern_index] for pattern_index in r])
```

結果です。

```
poetry run python main.py
----------
全世界の子供たちが
['世界', '子供']
----------
世界は広い
['世界']
```

簡単にPythonからDaachorseを呼び出すことができました。

## てかPythonバインディングすでにあった

ベンチマーク取り終えてひと段落してから気づいたのですが、普通にDaachorseのPythonバインディング公開されていたのでこちらを使えば良かった笑。まぁRustのCrateをPythonで呼び出すという練習になったので良しとします。

https://github.com/vbkaisetsu/python-daachorse

`python-daachorse`でも内部でPyO3を利用しているようなので、ここまでの知識はコードリーディングで生かせると思います。

## 実験

弊社の利用用途である日本語の文字列パターンマッチを使ったベンチマークを行い実際にどのような条件で高速化ができるのか確認します。ベースラインは弊社の現行ロジックで利用されている`ahocorasick`というPythonモジュールと、pure Python実装の`ahocorapy`、そしてdaachorseと同じRust CrateのPythonバインディングである``ahocorasick_rs``です。これらはLegalForceさんの記事にも解説のあるACマシンのPython実装になっています。

弊社の利用用途では、文字列パターンは変更があまり発生しないため、構築済みACマシンをgokartキャッシュとして利用します。そのため、ベンチマークではACマシン構築は含めず、純粋なパターンマッチのみのベンチマークを取りました。その他のベンチマークのデータセットは下記になります。

```
日本語のデータセット
パターン数: 22948(弊社のとある医療辞書)
テキスト数: 5094(弊社とある記事データ)
```

また、パターンとテキスト数はそれぞれ下記のような文字列長の分布になっています。実践的な分布かと思います。


結果は下記になります。

```
-------------------------------------------------------------------------------------------------- benchmark: 4 tests -------------------------------------------------------------------------------------------------
Name (time in ms)                              Min                   Max                  Mean              StdDev                Median                 IQR            Outliers      OPS            Rounds  Iterations
-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
test_match_daachorse_benchmark             80.8431 (1.0)        147.8423 (1.0)         93.7325 (1.0)       19.8576 (1.0)         86.4954 (1.0)       11.9653 (1.12)          1;1  10.6687 (1.0)          10           1
test_match_ahocorasick_rs_benchmark       123.6172 (1.53)       412.8799 (2.79)       179.8610 (1.92)     114.3097 (5.76)       136.1129 (1.57)      10.6961 (1.0)           1;1   5.5598 (0.52)          6           1
test_match_ahocorasick_benchmark          736.3777 (9.11)       901.6807 (6.10)       776.5008 (8.28)      70.4611 (3.55)       745.9376 (8.62)      54.7538 (5.12)          1;1   1.2878 (0.12)          5           1
test_match_ahocorapy_benchmark          1,339.0980 (16.56)    3,124.5482 (21.13)    1,908.8495 (20.36)    744.7254 (37.50)    1,565.3466 (18.10)    979.0138 (91.53)         1;0   0.5239 (0.05)          5           1
-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------```

## まとめ
