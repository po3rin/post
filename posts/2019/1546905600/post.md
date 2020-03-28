# Pythonで電子書籍フォーマットのEPUBからデータを抽出してみる

仕事で電子書籍フォーマットであるEPUBからデータを抽出する必要があったので、調べたところ ebooklib　といライブラリが便利そう。使ってみたので軽く紹介。

## EPUB とは

電子書籍フォーマットの一種です。 多くのデバイスの間でも汎用性高く、ここ数年段々普及しているフォーマットです。EPUB については　「EPUB 3とは何か？」 という本が オライリーから無料でダウンロードできます。https://www.oreilly.co.jp/books/9784873115528/

## Python で EPUB の中から必要なデータを所得する。

EPUBを扱うならこちらが便利
https://github.com/aerkalov/ebooklib

DocumentはこのPDFが詳しい
https://media.readthedocs.org/pdf/ebooklib/latest/ebooklib.pdf

今回はこちらのモジュールを使ってEPUBからデータを抜き出してみる。使い方はめちゃ簡単。

```python:main.py
import sys
import os
import ebooklib
from ebooklib import epub

book = epub.read_epub('.<.epubファイルのパス>')

title = book.get_metadata('DC', 'title')
creator = book.get_metadata('DC', 'creator')
publisher = book.get_metadata('DC', 'publisher')
language = book.get_metadata('DC', 'language')
  
print(title) # タイトル
print(creator) # 執筆者
print(publisher) # 発行人
print(language) # 言語

items = book.get_items()
   for item in items:
       if item.get_type() == ebooklib.ITEM_DOCUMENT:
           print('==================================')
           print('ファイル名 : ', item.get_name()) # ファイル名
           print('==================================')
           print(item.get_content().decode()) # 本文をファイルごとに書き出し
           print('==================================')

```

これで .epub 拡張子のファイルパスからデータの抽出ができた！このライブラリは他にも EPUB から画像データを抽出したり、実際にEPUBデータを作成できたりも可能。英語ならPDFでドキュメントがみれる。詳しくはこちらを！
https://media.readthedocs.org/pdf/ebooklib/latest/ebooklib.pdf

Elasticsearchに入れて分析できるようにしたいのでそこに関しても今後追記するかも

