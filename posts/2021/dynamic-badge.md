---
title: サウナ好きエンジニアの為にサ活数を表示するバッジを作った
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/sauna-dynamic-badge.jpg
date: 2021/01/23
id: dynamic-badge
description: 今回サウナ好きエンジニアの為のサ活バッジを作る君を開発したので、オリジナル動的バッジの作り方と合わせて共有します。
tags:
    - Python
---

## Overview

こんにちは [pon](https://twitter.com/po3rin) です。ホームサウナは[「かるまる池袋」](https://sauna-ikitai.com/saunas/6656)です。今回サウナ好きエンジニアの為の[サウナイキタイ](https://sauna-ikitai.com/)のサ活数バッジを作る君である [saunadge](https://github.com/po3rin/saunadge) を開発したので、オリジナル動的バッジの作り方と合わせて共有します。

## saunadge

![saunadge](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/saunadge.png)

[saunadge](https://github.com/po3rin/saunadge) はサウナイキタイのサ活数をバッジにして生成してくれるCLI兼、sheilds.io が叩く為のAPIサーバーです。

コードはこちら！！

![saunadge-github](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/saunadge-github.png)

saunadge CLIの使い方は簡単で、サウナイキタイのユーザーIDを渡せばバッジのフォーマットを出力してくれます。これをREADME.mdなどにそのまま貼れば完了です。

```sh
$ pip install saunadge
$ saunadge -i <saunaikitai-id>
[![sakatsu badge](https://img.shields.io/endpoint.svg?url=https://saunadge-gjqqouyuca-an.a.run.app/api/v1/badge/46531&style=flat-square)](https://sauna-ikitai.com/saunners/46531)
```

## 動的バッジ生成の仕組み

実は shields.io には動的にバッジの内容を外部JSONで読み込む仕組みがあります。

[shields.io Endpoint ドキュメント](https://shields.io/endpoint)

これを使うと外部APIを叩いてバッジの設定をロードできます。

```sh
https://img.shields.io/endpoint?url=<external api endpoint>
```

つまり shields.io が求めるJSONを返す外部APIを作成すれば動的なバッジを作成できます。返すJSONのフォーマットは決まっているのでそれに合わせてレスポンスを作れば完成です。

```json
{
    "schemaVersion": 1,
    "label": "Sakatsu",
    "message": 11,
    "color": "0051e0",
}
```

まとめるとこんな感じになります。

![saunadge-archi](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/saunadge-archi.png)

今回紹介したJSONフィールド以外にも色々なフィールドをサポートしているのでドキュメントを是非眺めてみてください。

[shields.io Endpoint ドキュメント](https://shields.io/endpoint)

## 動的バッジAPI実例

[saunadge](https://github.com/po3rin/saunadge) は Python + Flask で動かしています。インフラはGCPのCloud Runを利用しています。

下記のサンプルコードは見やすいように実際のコードから例外処理などを省いています。

```py
BASE_URL = "https://sauna-ikitai.com/saunners/"

@app.route("/api/v1/badge/<int:user_id>")
def tonttu_badge(user_id):
    res = requests.get(BASE_URL + f"{user_id}")
    soup = BeautifulSoup(res.text, "lxml")

    sakatsu = (
        soup.find("body")
        # 省略... 
        .get_text()
        .strip()
    )

    return {
        "schemaVersion": 1,
        "label": "Sakatsu",
        "message": sakatsu,
        "color": "0051e0",
        "cacheSeconds": 1800,
        "logoSvg": '<svg xmlns="http://www.w3.org/2000/svg" 省略... </svg>',
    }
```

[サウナイキタイ](https://sauna-ikitai.com/)のサ活数はBeautiful Soupによるスクレイピングで取得しています。スクレイピングに関する注意事項は下記が詳しいです。
[Webスクレイピングする際のルールとPythonによる規約の読み込み](https://vaaaaaanquish.hatenablog.com/entry/2017/12/01/064227)

確認したところ、[サウナイキタイ](https://sauna-ikitai.com/)では```robot.txt```を設定していないようです。また、今回はaタグを追っていかないのでrelチェック行いません。```robot meta```も確認したところ、大丈夫そうです。スクレイピングする場合は絶対にここは確認しておきましょう。また、実際のコードではスクレイピングの作法であるUser-agentも設定しています。

```
headers = {
    "User-Agent": f"saunadge/{__version__} (https://github.com/po3rin/saunadge)"
}
res = requests.get(BASE_URL + f"{user_id}", headers=headers)
```

また、負荷をかけないようにキャッシュを効かせることも重要です。今回は固定で1800sというかなり長いキャッシュを shields.io に支持しています。

logoSvgフィールドはバッジに付与するSVGアイコンを設定できます。そのまま値として渡せばOKです。動的にアイコンが変更するようにしても面白いかもしれません。SVGアイコンを利用する場合は sheilds.io のエンドポイントが少し変わるので注意です。

```sh
https://img.shields.io/endpoint.svg?url=external api endpoint>
```

これでサ活数をユーザーIDで引っ張ってこればAPIの完成ですね。saunadgeのCLIモードはendpointを含んだshields.ioのURLを文字列を出力するだけなので簡単です。

## まとめ

shields.io のエンドポイント機能で簡単に動的なバッジが作れました。これを公開したところ、サウナ好きエンジニアがたくさんいることがわかりました。サウナ好きエンジニア集めてコロナ収束後にサウナ開発合宿したいですね。

saunadgeはざっと作っただけでまだ機能が足りないのでコントリビュート大歓迎です。
![saunadge-github](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/saunadge-github.png)


## 参考資料 & 役立ちリンク

全国民登録必須
[サウナイキタイ 日本最大のサウナ検索サイト](https://sauna-ikitai.com/)

バッジ作成の総締め
[shields io ドキュメント](https://shields.io)
