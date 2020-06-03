---
title: Elasticsearch の Mapping 管理を Go + CUE に移行した
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/cue-es-i.jpeg
date: 2020/06/01
id: mapping-cue
description: Elasticsearch の Mapping JSON を全て Go の構造体で定義していたのを Go + CUE に移行したので知見を共有します。
tags:
    - Go
    - Elasticsearch
---

## Overview

こんにちは[pon](https://twitter.com/po3rin)です。私が働いている[白ヤギコーポレーション](https://shiroyagi.co.jp/)では```Elasticsearch```を利用しているのですが、顧客ごとにIndexの設定、言語、Analyzerなどをカスタマイズできるようになっています。そのため、顧客の設定をDBから取得してGoで構造体を通してJSONを生成し、Mappingを作成/更新する機構が存在します。これを ***Go + CUE*** に移行して課題が解決できたので共有します。少し珍しいCUEのusecaseだと思います。

## Before

CUEの紹介の前に、まずは弊社が抱えていた課題をお話しします。改修前に問題になっていたのが ***顧客別の設定をGoの構造体にねじ込んでJSONに変換する部分*** です。顧客の設定が非常に複雑であるため、構造体生成のコードも複雑になり、コードが追えなくなっていました。

実際に改修前のGoのコード一例を見てみましょう。これでも見やすいように省略してますが、伝えたいことは伝わると思います。

```go
func getBaseSetting(e env) settings {
	s := settings{
		Analysis: analysis{
			Analyzer:   getAnalyzers(e.analyzers),
			Tokenizer:  getTokenizers(e.tokenizers),
			Filter:     getFilters(e.filters),
			CharFilter: e.charFilters,
		},
		NumberOfReplicas: e.numberOfReplicas,
		NumberOfShards:   e.numberOfShards,
		Index: index{
			Similarity: similarity{
				Default: defaultSim{
					Type: defaultSimilarity,
				},
			},
		},
	}
	return s
}
```

先述のようにMappingとして投げるJSONを構造体経由で生成しようとすると、フィールドの値を関数で生成したり、グローバル変数を渡したりする必要が出てきます。その為、***一目ではでどんなJSONが作られるか分からないコードが完成します***。もちろんfieldの値を生成する関数も他の関数をコールしたり、渡す引数によって値を変えるということをやっているので、コードを追うだけで時間的コストが発生します。こんな関数が10個以上あったのでそれは大変でした。

更に構造体を介してJSONを生成しているのでMappingに必要なフィールドをもつ構造体を１つ１つ定義していく必要がありました。改修前は下記のような構造体が10個以上定義されていました。

```go
type settings struct {
	Analysis         analysis `json:"analysis"`
	MaxResultWindow  int      `json:"max_result_window"`
	MaxRescoreWindow int      `json:"max_rescore_window"`
	NumberOfReplicas int      `json:"number_of_replicas"`
	NumberOfShards   int      `json:"number_of_shards"`
	Index            index    `json:"index,omitempty"`
}
```

### CUE

これを改修するに辺り満たすべきポイントは2点ありました。

* Go からテンプレートに値を渡してJSONを生成できる
* Mappingの値をバリデーションしたい
* どんなMappingが生成されるのかがすぐに分かる(可読性)

そこで弊社では [CUE](https://cuelang.org/) に目をつけました。***CUEは、オープンソースのデータ検証言語および推論エンジンです。*** データ検証、データのテンプレート化、設定、クエリ、コード生成、さらにはスクリプト化など、多くの機能を備えています。CUE の 更なる良さとして Go とのスムーズな連携が挙げられます。公式からCUEを扱うGoパッケージが提供されているのは心強いですね。

CUEについてはフューチャーの[澁川さん](@shibu_jp)のブログがまとまっています。2記事の連載で後半はCUEをGoで扱う話もあります。
[CUEを試して見る](https://future-architect.github.io/articles/20191002/)

私はCUEを Kubernetes Meetup Tokyo の [チェシャ猫](https://twitter.com/y_taka_23) さんの発表で知りました。発表では Kubernetes の Manifest 管理を CUE で管理する話が上がっています。
[設定記述言語 CUE で YAML Hell に立ち向かえ](https://speakerdeck.com/ytaka23/kubernetes-meetup-tokyo-29th)

これを使うとMappiingをCUEファイルで管理できます。下記は省略していますが、弊社での一例```article.cue```です。

```json
// 渡せる変数の型指定、データ検証が記述可能
var_lang:       "ja" | "en" | "zh" | "ko" //利用言語
var_similarity: "classic" | "BM25" // 許容する scoring algorithm
var_analyzer:   string // 型定義
var_additional_fields: [...] // 配列

index: {
	settings: {
		index: {
			similarity: {
				default: {
					type: (var_similarity)
				}
			}
		}
	}
	mappings: {
		article: {
			"_meta": {
				lang: (var_lang)
			}
			properties: {
                // 利用する Analyzer を埋め込める
				title: {
					analyzer:    (var_analyzer)
					type:        "text"
                }
                // ...

                // 顧客が設定できる任意長のカスタムフィールドを for で生成
				for f in (var_additional_fields) {
					"\(f.name)": {
						if f.type == "text" {
							type:     "text"
							analyzer: (var_analyzer)
						}
						if f.type == "number" {
							type: "double"
						}
					}
				}
			}
		}
	}
}
```

データの型、バリデーションをファイル内に記述でき、どんなJSONが生成されるか一目で分かります。JSONのフィールドの値を生成する関数をいちいち追わなくても、このファイルを見るだけでどんな値が渡されるのか、どんなデータを許容しないのかが一目で分かります。(懸念はfor文の箇所が少し見にくいかも？くらい)

弊社では日本語/英語/中国語など、言語ごとにAnalyzerを選べるので上記のCUEに別のAnalyzer用のCUEをマージしてJSONを生成します。下記はマージする英語のAnalyzerを記述したCUEです。

```json
index: {
	settings: {
		analysis: {
			analyzer: {
				english_analyzer: {
					// ...
				}
				// ...
			}
			// ...
		}
	}
}
```

この２つをマージし、テンプレートに値を埋め込むと下記のような Elasticsearch 用の Mapping JSON が生成できます。

```json
{
    "mappings": {
        "article": {
            "_meta": {
                "lang": "en"
            },
            "properties": {
                "title": {
                    "analyzer": "english_analyzer",
                    "type": "text"
                },
                "company_name": {
                    "type": "text",
                    "analyzer": "english_analyzer"
                },
            }
        }
    },
    "settings": {
        "index": {
            "similarity": {
                "default": {
                    "type": "BM25"
                }
            }
        },
        "analysis": {
            "analyzer": {
                "english_analyzer": {
                    // ...
                },
               // ...
            },
            // ...
        },
        // ...
    }
}
```

実際のJSON生成はGoで行っています。CUEを扱うためのGoパッケージが公式で提供されているのでこれを利用します。下記は Goで Mapping CUE に変数を渡し、Analyzer CUE をマージして JSON を生成する例です。

```go
var r cue.Runtime
var lang = "ja"

// CUEインスタンスを作成
// // 記事用
articleIns, _ := r.Compile("article.cue", "article")
// // Analyzer用
analyzerIns, _ := r.Compile(fmt.Printf("%s_analyzer.cue", lang), "analyzer")

// テンプレートに値を渡す例
articleIns, err := articleIns.Fill(lang, "var_lang")
// ...

// Analyzer CUE をマージ
merged := cue.Merge(articleIns, analyzerIns)

// バリデーションもGo側で可能
err = merged.Value().Validate()

// CUE から JSON に変換
json, _ := ins.Lookup("index").MarshalJSON()

```

かなり楽にCUEに値を渡せることに加え、渡したデータの検証もスムーズに行えます。余談ですが、弊社では ***Builder Pattern***でこの辺の処理をラップして下記のように扱いやすい形で提供しています。Builder Patternは複雑なオブジェクトの構築とその表現を分離して構築プロセスを提供できるものです。

```go
import "mapping"

mappingJSON := mapping.Article().
    Lang(nlp.LangJapanese).
    Similarity("BM25").
    AdditionalField("company_name", "text").
    JSON()

```

これでCUEの処理をうまい具合に抽象化できました。GoのElasticsearchクライアントである [olivere/elastic](https://github.com/olivere/elastic) を使っているのであればこのjsonをそのまま```BodyJson```に渡せばOKです。

```go
_, err := client.CreateIndex(name).BodyJson(mappingJSON).Do(ctx)
```

ご覧の通り、CUEファイルからJSON生成、インデックス生成まで一回も構造体を介していません。そのため、***Mapping生成のためだけに存在していた構造体を全て削除できました。***

## CUEを使ってみた所感

他のテンプレート言語に比べて表現力の高さに驚きます。型チェックやらデータの整合性チェックをGoなどの言語側で行わなくても、CUEファイル上で宣言できるので複雑な構造のJSONやYAMLを生成するのには便利です。更にGoとの相性がよく、エコシステムもかなり充実してます。```cue fmt```によるフォーマットなどが公式から提供されているのは良いですね。デメリットとしてはやはり学習コストです。まだ資料も少なく、ほとんど公式ドキュメント頼りです。そして、CUEがかなり表現豊かなので、仕様を掴むのに少し時間がかかるかもしれません。また、Go以外のパーサーが提供されていないので、Go以外の言語でCUEを扱うのが現状困難です。

## まとめ

CUEのusecaseとして珍しい使い方を紹介しました。CUEはまだ発展途上なので、今後の成長にも期待したいところです。
