---
title: JINS ASSISTを使ったソフトウェアエンジニアの新しい作業効率化を探求しようか
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/jins-assist.jpg
date: 2025/04/01
id: jinsassist
description: JINS ASSISTを使ってみた感想と、どのように作業効率化を実現できるかを紹介
tags:
  - gadget
---

## Overview

こんにちは。[pon](github.com/po3rin)です。ショートカットによる作業効率化の次のステージ。手を使わないパソコン操作を探求するために最近[JINS ASSIST](https://www.jins.com/jp/jins-assist)を購入しました。
今回の記事ではJINS ASSISTを使ってみた感想と、どのように作業効率化を実現できるかを紹介します。

## JINS ASSISTとは
[JINS ASSIST](https://www.jins.com/jp/jins-assist)とはメガネに装着する形で使用するUSBデバイスです。

頭を動かすことでパソコン操作を可能とします。こんな感じでメガネに装着して使います。普段使いのメガネに装着するだけなので非常に簡単です。

![装着イメージ](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/pon-jins.jpg)

次の画像は[公式のリファレンス](https://jins-assist.jins.com/manual.html)からの引用です。下を向いたり、横を向いたりしてパソコンを操作できます。

![通常モードにおける基本操作](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/jins.png)

開発背景としては手が不自由な方がパソコンなどのデバイスを操れるようにするというものです。しかし、手を使わずにパソコン操作をするという点では我々のようなエンジニアの作業効率を上げる未来も見えます。次からは機能を検討していきます。

### カーソル機能

頭を動かすことでカーソル移動ができます。残念ながらカーソル移動は作業効率化には使えませんでした。ボタンをクリックするためにカーソルを頭で操作すると狙ったところにカーソルを合わせるのが難しく、手で操作した方が断然早いという結論になりました。筆者はJINS ASSISTではカーソル機能を無効にして使っています。

### スクロール機能

デフォルトではスクロール機能はオフになていますが、設定でオンにすると頭を傾けることでスクロールが可能です。次の画像は[公式のリファレンス](https://jins-assist.jins.com/manual.html)からの引用です。

![通常モードにおける基本操作](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/tilt_scroll.webp)

残念ながらスクロールも作業効率化には寄与しませんでした。頭の傾き量でスクロール量を調整するのが少し難しく、期待したところで止めるというのが難しいです。もちろん設定で傾きに対してどれだけ動かすかは設定可能ですが、自分が納得する設定を見つけれませんでした。こちらも筆者は機能を無効にして使っています。

特に筆者はChromeでは[Vimium](https://github.com/philc/vimium)を入れているのでカーソル移動、スクロールはキーボードで完結しており、頭で移動を微調整する難しさを採用することはありませんでした。ここまでに紹介した機能を有効にするときは両手でハンバーガーを食べているときですね。

ここまででの紹介だと「あまり使えなさそう」と思うかもしれませんがちょっと待ってください。ここからJINS ASSISTを使った作業効率化ができた機能を紹介します。

### ショートカット機能

これです。これがJINS ASSISTが作業効率化を寄与するものです。この機能により頭の動作で任意の入力を行えます。

JINS ASSISTでは左を向く、右を向く動作にショートカットを割り当てることができます。ショートカット設定は次のようなUIで設定が可能です。

![筆者のショートカット設定画面](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/shake.png)

この機能をランチャーアプリなどと組み合わせて、パソコン操作をより強力にできます。

私はこのショートカット機能を[Raycast](https://www.raycast.com/)のHotKey発火に使っています。例えば頭を右に向けるとiTermをHotKeyで起動できるように設定しています。

その他にも頭を2回右に向けるとRaycast経由でカレンダー閲覧。左に一回でRaycast経由でChromeのタブ検索ができるようにしています。

このような手以外で作業効率化する選択肢としてフットスイッチがあります。例えば次の記事ではフットスイッチを使った作業効率化について紹介しています。

[足で動かすUnitTest](https://zenn.dev/kawahara/articles/1d6c724d92068a)

しかし、フットステップの弱点として、フットステップに足をかけるための椅子や机の高さの調整が必要です。また、外に持ち出しにくいなどの問題もあります。しかしJINS ASSISTは非常に軽量であり、メガネにつけるだけなので、いつもの作業場にすぐに導入でき、外にも持ち出しやすいです。その点ではJINS ASSISTは新しい作業効率化ツールの選択肢になるのではないでしょうか。

### 他の動作も上書きする

クリック、つまり頷く動作はデフォルトのままでは使わないので[Karabiner-Elements](https://karabiner-elements.pqrs.org/)で上書きしています。通常時ではRaycast経由でSlackのHotKeyとして利用しています。次はKarabiner-Elementsで筆者が設定したComplex Modificationの例です。

```json
{
    "description": "JINS ASSIST left_click",
    "manipulators": [
        {
            "conditions": [
                {
                    "identifiers": [
                        {
                            "product_id": 11111,
                            "vendor_id": 11111
                        }
                    ],
                    "type": "device_if"
                }
            ],
            "from": { "pointing_button": "button1" },
            "to": [
                {
                    "key_code": "z",
                    "modifiers": ["left_shift", "left_option"]
                }
            ],
            "type": "basic"
        }
    ]
}
```

`product_id`と`vendor_id`の調べ方は先ほど紹介した[記事](https://zenn.dev/kawahara/articles/1d6c724d92068a)を参考にしてください。

筆者はアプリケーションにフォーカスが当たっている場合はさらに設定を上書きするようにしています。例えばClaude Desktopではチャットのやり取りで改行エンターが「Shift + Enter」で面倒なので、頷きに「Shift + Enter」を当てています。

```json
{
    "description": "JINS ASSIST left_click on Claude Desktop ",
    "manipulators": [
        {
            "conditions": [
                {
                    "identifiers": [
                        {
                            "product_id": 16,
                            "vendor_id": 14114
                        }
                    ],
                    "type": "device_if"
                },
                {
                    "bundle_identifiers": [
                        "com.anthropic.claudefordesktop"
                    ],
                    "type": "frontmost_application_if"
                }
            ],
            "from": { "pointing_button": "button1" },
            "to": [
                {
                    "key_code": "return_or_enter",
                    "modifiers": ["left_shift"]
                }
            ],
            "type": "basic"
        }
    ]
}
```

そのほかにもiTermでの頷きをVim用に上書きしたりしています。Karabiner-Elementを使えば、当然頷きだけでなく、横を向いたりする動作もアプリケーションごとに上書きが可能ですので作業効率化の幅が格段に広がります。

## JINS ASSISTは作業効率化の未来かもしれない。

ここまでの紹介で、JINS ASSISTで作業効率化する方法をお伝えしました。

実際にJINS ASSISTを導入してまだ困っている点、要望も挙げておきます。

 * JINS ASSISTが途中で傾いてきてしまう。もう少し強く固定できるようにしてほしい。
 * スクロール(頭を傾ける)にもショートカットがつけれると良い。現状使っていないアクションなので。
 * Bluetooth版も是非。。。

この辺が更に改良されればより使いやすい作業効率化ツールとしても広まるかもしれません。ショートカット機能に関しては正に新しい操作体験なので、この機会にみなさんもぜひ購入を検討してみてはいかがでしょうか。

