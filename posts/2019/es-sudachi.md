---
title: Elasticsearch + Sudachi + Docker でユーザー辞書を作ってみるハンズオン
cover: img/gopher.png
date: 2019/10/18
id: es-sudachi
description: 今回は Elasticsearch + Sudachi でユーザー辞書を使う Dockerfile を作ったので作り方を共有します。 Elasticsearchのバージョンは現行の最新(v7.4.0)ですがv6.8あたりでも動くことを確認済みです。
---

## Sudachi とは

<img src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2019/1571356800/a7378513-f8e8-2bd0-7cc0-b9c420347ae1.png" width="120px">

Sudachi は日本語形態素解析器です。株式会社ワークスアプリケーションズ下の機関であるワークス徳島人工知能NLP研究所が開発しています。複数の分割単位をサポートしているなどの特徴があります。
ドキュメントはこちら
https://github.com/WorksApplications/Sudachi/#sudachi-%E6%97%A5%E6%9C%AC%E8%AA%9Ereadme


## 今回のハンズオンの最終構成

最終的に下記のような構成を目指します。

```bash
.
├── docker-compose.yml
└── elasticsearch
    ├── Dockerfile
    └── sudachi
        ├── README.md
        ├── custom_dict.txt
        └── sudachi.json
```

サンプルのリポジトリはこちらになります。
<a href="https://github.com/po3rin/sudachi-elasticsearch-sample"><img src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2019/1571356800/820c6338-c29e-f8e3-218a-eb63ca6aa7d3.png" width="460px"></a>



## custom_dict.txt

まずはユーザー辞書の元となるユーザー辞書ソースファイル```custom_dict.txt```を作りましょう。ちなみにファイル名は任意の名前でもOKです。今回の例では僕のTwitterIDの「po3rin」を辞書に追加する例を見てみます。

```txt:custom_dict.txt
po3rin,4786,4786,5000,po3rin,名詞,固有名詞,一般,*,*,*,po3rin,po3rin,*,*,*,*,*
```

フォーマットに関してはドキュメントに全てまとまっているのでこちらを参照して下さい。
https://github.com/WorksApplications/Sudachi/blob/develop/docs/user_dict.md

もちろん、こんなwordは辞書にないので本来は下記のようにバラバラにtokenizeされてしまいます。

```json
{
  "tokens" : [
    {
      "token" : "po",
      // ...
    }
    {
      "token" : "3",
      // ...
    }
    {
      "token" : "rin",
      // ...
    }
  ]
}

```

ユーザー辞書に「po3rin」を追加するとどうなるかは後ほどみてきましょう。

## sudachi.json

続いて Sudachi プラグインの設定ファイルである ```sudachi.json``` を上書きできるようにファイルを準備しましょう。デフォルトの設定から変えているのは ```systemDict``` と ```userDict``` フィールドです。```systemDict``` はハンズオン用に一番小さい辞書に切り替えています。デフォルトは```system_core.dic``` です。

```json:custom_dict.txt
{
    "systemDict": "system_small.dic",
    "userDict": [
       "custom.dic"
    ],
    "inputTextPlugin": [
        {
            "class": "com.worksap.nlp.sudachi.DefaultInputTextPlugin"
        },
        {
            "class": "com.worksap.nlp.sudachi.ProlongedSoundMarkInputTextPlugin",
            "prolongedSoundMarks": [
                "ー",
                "-",
                "⁓",
                "〜",
                "〰"
            ],
            "replacementSymbol": "ー"
        }
    ],
    "oovProviderPlugin": [
        {
            "class": "com.worksap.nlp.sudachi.MeCabOovProviderPlugin"
        },
        {
            "class": "com.worksap.nlp.sudachi.SimpleOovProviderPlugin",
            "oovPOS": [
                "補助記号",
                "一般",
                "*",
                "*",
                "*",
                "*"
            ],
            "leftId": 5968,
            "rightId": 5968,
            "cost": 3857
        }
    ],
    "pathRewritePlugin": [
        {
            "class": "com.worksap.nlp.sudachi.JoinNumericPlugin",
            "joinKanjiNumeric": true
        },
        {
            "class": "com.worksap.nlp.sudachi.JoinKatakanaOovPlugin",
            "oovPOS": [
                "名詞",
                "普通名詞",
                "一般",
                "*",
                "*",
                "*"
            ],
            "minLength": 3
        }
    ]
}
```

```systemDict``` には使うシステム辞書、```userDict``` にはこれからビルドするバイナリ辞書ファイルを指定します。ちなみに ```userDict``` には複数指定可能です。今回の例ではシステム辞書の追加やユーザー辞書のビルドはDocker内で行います。

## Dockerfile

それではユーザー辞書ソースファイルをビルドして一気にElasticsearchを立ち上げるDockerfileを作りましょう。
 ```elasticsearch```ディレクトリの中にDockerfileを準備します。中身は下記です。

```Dockerfile:Dockerfile
FROM ibmjava:8-jre-alpine as dict_builder

## 辞書の種類の指定(small/core/full)
ARG sudachi_dict_type="small"

## ユーザー辞書ソースを持ってくる
COPY sudachi/custom_dict.txt /home

WORKDIR /home

# Sudachiプラグインのjarファイルを持ってくる (バイナリ辞書の作成のため)
RUN wget https://github.com/WorksApplications/elasticsearch-sudachi/releases/download/v7.4.0-1.3.1/analysis-sudachi-elasticsearch7.4-1.3.1.zip && \
    unzip analysis-sudachi-elasticsearch7.4-1.3.1.zip && \
    # 用意されているシステム辞書を持ってくる
    wget https://object-storage.tyo2.conoha.io/v1/nc_2520839e1f9641b08211a5c85243124a/sudachi/sudachi-dictionary-20190718-${sudachi_dict_type}.zip && \
    unzip sudachi-dictionary-20190718-${sudachi_dict_type}.zip && \
    # バイナリ辞書の作成
    java -Dfile.encoding=UTF-8 -cp /home/sudachi-0.3.0.jar com.worksap.nlp.sudachi.dictionary.UserDictionaryBuilder -o /home/custom.dic -s /home/sudachi-dictionary-20190718/system_small.dic /home/custom_dict.txt


FROM elasticsearch:7.4.0

ARG sudachi_dict_type="small"

# Sudachiプラグインの設定ファイル
COPY sudachi/sudachi.json /usr/share/elasticsearch/config/sudachi/
# 前ステージでダウンロードしたSudachiのシステム辞書
COPY --from=dict_builder /home/sudachi-dictionary-20190718/system_${sudachi_dict_type}.dic /usr/share/elasticsearch/config/sudachi/
# 前ステージで作ったユーザー辞書
COPY --from=dict_builder /home/custom.dic /usr/share/elasticsearch/config/sudachi/
# 前ステージでダウンロードしたプラグイン
COPY --from=dict_builder /home/analysis-sudachi-elasticsearch7.4-1.3.1.zip /usr/share/elasticsearch/

# Sudachiプラグインインストール
RUN elasticsearch-plugin install file:///usr/share/elasticsearch/analysis-sudachi-elasticsearch7.4-1.3.1.zip && \
    rm /usr/share/elasticsearch/analysis-sudachi-elasticsearch7.4-1.3.1.zip

```

SudachiプラグインのバージョンはElasticsearchのバージョンと基本的に一致させてください。マルチステージビルドにしている理由は余計なJavaコマンドなど不要な物をElasticserchのイメージに含めたくないからです。 ```dict_builder``` ステージでは主にユーザー辞書ソースからバイナリ辞書ファイルのビルドを行なっています。[ドキュメント](https://github.com/WorksApplications/Sudachi/blob/develop/docs/user_dict.md)にも記載がありますが、下記のようなコマンドでユーザー辞書ソースからバイナリ辞書を作成できます。


```bash
$ java -Dfile.encoding=UTF-8 -cp sudachi-XX.jar com.worksap.nlp.sudachi.dictionary.UserDictionaryBuilder -o output.dic -s system_core.dic [-d comment] input
```

output.dic 出力するバイナリ辞書ファイル名
system_core.dic Sudachi のシステム辞書
comment バイナリ辞書のヘッダーに埋め込むコメント
input.csv ユーザ辞書ソースファイル名

もちろん手元でビルドした物をDockerfileにコピーしてきても良いですが、筆者は管理するファイルを減らしたい&重い辞書バイナリをGitで管理したくない&Java環境の構築がめんどいという理由でDocker内で完結させています。

## 動作確認

動作確認用の docker-compose.yml を作ります。Kibanaを入れているのはKibanaに搭載されている```DevTools```という機能が便利だからです。

```yml:docker-compose.yml
version: '3.6'
services:
  elasticsearch:
    build: ./elasticsearch
    container_name: elasticsearch
    environment:
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
      - discovery.type=single-node
      - node.name=es01
    ports:
      - '9200:9200'
      - '9300:9300'

  kibana:
    image: docker.elastic.co/kibana/kibana:7.4.0
    links:
      - elasticsearch
    environment:
      - ELASTICSEARCH_URL=http://elasticsearch:9200
    ports:
      - 5601:5601

```

早速立ち上げます。ビルドと立ち上げは下記コマンド一発です。

```bash
$ docker-compose up --build
```

早速 Kibana の URL (http://localhost:5601) にブラウザからアクセスしてみましょう。Menu に ```DevTools```があるので開いてConsoleに以下を記述します。

```json
PUT /sample-index
{
  "settings": {
    "number_of_shards": 1,
    "analysis": {
      "analyzer": {
        "sudachi_analyzer": {
          "type": "custom",
          "tokenizer": "sudachi_tokenizer"
        }
      },
      "tokenizer": {
        "sudachi_tokenizer": {
          "type": "sudachi_tokenizer",
          "mode": "search",
          "discard_punctuation": true,
          "resources_path": "/usr/share/elasticsearch/config/sudachi/",
          "settings_path": "/usr/share/elasticsearch/config/sudachi/sudachi.json"
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "text": {
        "type": "text",
        "analyzer": "sudachi_analyzer"
      }
    }
  }
}

POST sample-index/_analyze
{
  "analyzer": "sudachi_analyzer",
  "text": "po3rin"
}
```

Kibana上ではこんな感じになります。

<img width="700" alt="スクリーンショット 2019-10-18 18.46.20.png" src="https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2019/1571356800/f09804fc-e926-62a4-fa3f-1b74c5bae5e4.png">


```settings_path``` で指定している ```sudachi.json``` を見て使う辞書をtokenizerが理解してくれます。
DevToolsだけでElasticsearchに対してAPIが叩けます。これらを実行すると下記のレスポンスが確認できます。

```json
{
  "tokens" : [
    {
      "token" : "po3rin",
      "start_offset" : 0,
      "end_offset" : 6,
      "type" : "word",
      "position" : 0
    }
  ]
}
```

ユーザー辞書に登録した「po3rin」が一単語として認識できています。

以上です。

