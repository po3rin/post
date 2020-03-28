# DockerでPython3.6の環境構築！matplotlibインストールで詰まった話とかも

## DockerでPython3.6環境構築！

前使っていたMacのPython環境が行き当たりばったりのインストールで
よくわかんなくなってしまったので、最近はやりのDockerとやらを試してみた。

まずは本家サイトからダウンロード。僕はStableの方を入れました！
> https://docs.docker.com/docker-for-mac/install/

下記コマンドでダウンロードできたか確認。

```
$ docker --version
Docker version 17.12.0-ce, build c97c6d6
$ which docker
/usr/local/bin/docker
```

いーね。

## Docker Hubからイメージを入手

Dockerは実行環境や操作方法をまとめて1つのパッケージにし、それを「Dockerイメージ」として保存／配布している。
今回は公式がだしてる```library/python```を引っ張ってくる。```docker pull python:<バージョン>```で指定のPythonのイメージをもってこれる。僕は3.6が使いたかったので、下記を実行。新幹線の中だったからちょっと時間かかった。

```
$ sudo docker pull python:3.6
3.6: Pulling from library/python
f49cf87b52c1: Pull complete
7b491c575b06: Pull complete
b313b08bab3b: Pull complete
51d6678c3f0e: Pull complete
09f35bd58db2: Pull complete
0f9de702e222: Pull complete
73911d37fcde: Pull complete
99a87e214c92: Pull complete
Digest: sha256:98149ed5f37f48ea3fad26ae6c0042dd2b08228d58edc95ef0fce35f1b3d9e9f
Status: Downloaded newer image for python:3.6
```

成功したっぽい。```docker image ls```で確認できる

```
$ docker image ls
REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
python              3.6                 c1e459c00dc3        6 weeks ago         692MB
```

おお、サイズも。いつ作られたかも見れる。

## コンテナ起動

さっそくコンテナを起動する。```--name <コンテナ名前>```のオプションでコンテナに名前つけれます。

```
docker run -it --name pytest python:3.6 /bin/bash
```

はいれた！！pythonがちゃんと3.6使えるのか確認

```
# python --version
Python 3.6.2
```

おお、Python2.7系しか入ってなかったのに、3.6が使えるぞ！
今までpyenvでやらなんやらで行ってたので、これは楽かも。

```
# pip --version
pip 9.0.1 from /usr/local/lib/python3.6/site-packages (python 3.6)
```

ついでにpipもはいってた。ざす。
pipでnumpyやらなんやらいれてもホスト環境が汚れることはありません。

## Vimのインストール
Docker | docker コンテナの中で vim が使えない！ってなった。基本僕は簡単なコードならvimで編集してたので、入れとく。

```
# apt-get update
# apt-get install vim
```

これでvimが使える様になった。

## いざPython実行

早速、pythonのコードを書いて実行しよう。
/home/test.pyを作る。

```
# cd home
# vi test.py
```

コードは以下の通り
ちなみに```a```押すと入力モードに入り、ESCキー押して```:qw```で保存終了できる

```python:test.py
#-*- coding: utf-8 -*-
print("hello, python3.6")
```

これでいざ実行！

```
# python test.py
hello, python3.6
```
ログでた。よろしい。

## 終了するには
ホスト環境戻るには下記コマンド。

```
# exit
```

実際にPython3.6が本当にコンテナ上だけのものだったのかをホスト環境で確認してみた。

```
$ python --version
Python 2.7.10
```

戻ってる！俺たち、元の世界に戻ってきたんだ！！

## 再びコンテナ起動

再びコンテナに入りたいときは```docker exec```使います。

```
# docker exec -it pytest /bin/bash
```

## matplotlib インストールで "no display name and no $DISPLAY environment variable"
ちょっとつまづいたので、共有しとく。matplotlibを使ったスクリプトを実行したら下記のエラーが出た

```
_tkinter.TclError: no display name and no $DISPLAY environment variable
```

matplotlibnの設定をいじる必要があるそう。以下で設定ファイルさがす。

```
# find / -name matplotlibrc
/usr/local/lib/python3.6/site-packages/matplotlib/mpl-data/matplotlibrc
```

でてきた。こいつの中身を開いて

```
backend : TKAgg
```

を

```
backend : Agg
```

に変えてあげる。僕は ```matplotlibrc```の41行目くらいにあった
これでエラーが消えました。

以上です。

