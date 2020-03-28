# チームで使うPython3開発環境の比較検討と構築手法

ずっとPython2をチームで使ったけど、開発方法の改善期間を設けたので、ここでPython3に上げることにました。また、メンバーの開発環境がバラバラで、使っているパッケージの共有もできていなかったので、ここらで見直した。今回はPythonの環境構築の方法をまとめたものと、実際の構築手順や、パッケージの共有方法を書きます。

## 要件

* 全員の開発環境をパッケージ等も含めて統一したい
* pythonのバージョン変えてテストとかはしない
* Docker使うでもいいけどDebugはしたい
* 初心者もなるべくハマることのない構築手順が良い

## 候補の洗いだし
どれもエディターはVSCodeを使用想定(社内普及率が高い為)

* pyenv
* venv
* pipenv
* Anaconda
* Docker + ptvsd

## pyenv
バージョン指定して処理系をとってきて共存させられる。複数のバージョン切り替えしてテストしたいだけなら便利。pyenvのインストールは色んな人が色々書いているので、pyenvを使いこなす前に色んなブログ等を見てしまうと、最終的にハマるパターンが多い。pyenvをちゃんと使うための学習コストが少しかかる。これだけではプロジェクト毎に閉じた環境は作れない為、追加でまた色々必要になる。

公式ソース https://github.com/pyenv/pyenv
参考構築手順 https://qiita.com/koooooo/items/b21d87ffe2b56d0c589b

## venv
環境分離ツール。pythonバージョンやパッケージ状態をプロジェクト毎に閉じた環境が作れる。Python3.3からvenvという名前で標準で取り込まれており無駄なインストールが不要。各種ツールのサポートが充実しているのが強み(VSCodeなど)。パッケージ管理の共有は.txtベースと古風。プロジェクト内の環境を```activate```や```deactivate```でON,OFFできる.逆に毎回これを打たなきゃいけないのは面倒。Python 3.x.x 下に組み込まれた機能なので，Python自体のバージョンは管理できない。

公式ドキュメント https://docs.python.jp/3/library/venv.html
参考構築手順 https://qiita.com/fiftystorm36/items/b2fd47cf32c7694adc2e

## pipenv
Pipenvは、Python公式が正式に推薦するPythonパッケージングツール。pythonのバージョンやパッケージも含めた仮想環境がすぐに作れる。pipとvenvが連動している為、venvの上位互換感がある。node.jsでいうとこのnpmみたいなものも作ってくれる。仮想環境はプロジェクトフォルダの外の見えないとこに作ってくれる。色々便利にできるが故にpipenvコマンドの使い方を覚えるのに学習コストがかかる。

公式ドキュメント http://pipenv-ja.readthedocs.io/ja/translate-ja/#
参考構築手順 https://torina.top/detail/458/

## Anaconda
データ分析に使えるライブラリとかツールが一括で使えるようになる(もちろん使わないパッケージも入る)。pythonバージョンも切り替えれたりと便利。Anacondaが中で持っているツールがOSの持っているツールを覆い隠すため、今まで使っていたツールとバッティングしたりする。pyenvでAnacondaを使うと言う手もある

公式ドキュメント https://docs.anaconda.com/
参考構築手順 https://qiita.com/berry-clione/items/24fb5d97e4c458c0fc28

## Docker + ptvsd
docker上で動くpythonにVSCodeからデバッグできる。ローカル開発環境作りたいだけならvenvだけでいい気がする。コードに```import ptvsd```入れたりと少し手順が多め。dockerの学習時間も必要になる。

構築手順　https://blog.stedplay.com/how-to-remote-debug-python-on-docker-in-vscode/

## venv + VSCodeによる環境構築
以上から、メンバーと環境を共有するなら、学習コストがほぼゼロで分かりやすいvenvが良いのではないか。よって今回はvenv+VSCodeによる環境構築手順を書きます。

### 現在の環境確認

```bash
$ python -V
Python 2.7.10

$ python3 -V
Python 3.6.3
```

もしPython3がまだ入ってない場合はHomebrewでインストールしておく(venvが3系に組み込まれている為)

```bash
$ sudo brew install python3
```

### 新しい環境を作成

venvを使って閉じた環境を作っていきます。[]の中の名前はお好みで

```bash
# 任意の場所で
mkdir [project name]
cd [project name]
python3 -m venv [env name]
```

これで環境設定系が全て[env name]に入ります。

```bash
$ ls
[env name]
```

### 環境設定のON OFF

#### ON

```
$ source [env name]/bin/activate
```
プロンプトの頭に環境名が表示される。実際に```python -V```でバージョンが変わったことを確認してみてください。

ちなみにシェルがbashでない場合(fishとかcshとか)の場合は下記のコマンドになる

|シェル  |仮想環境有効化コマンド  |
|---|---|
|bash/zsh|$ source [env name]/bin/activate|
|fish|$ . [env name]/bin/activate.fish|
|csh/tcsh|$ source [env name]/bin/activate.csh|

#### OFF

```bash
([newvenvname])$ deactivate
```

プロンプトの頭に環境名が消える。実際に```python -V```でバージョンが元に戻ることを確認してみてください。

### VSCodeと連携

VSCodeはまだOSのPythonのバージョンをみにいくので```.vscode/setting.json```を作りそこに参照するpythonのバージョンを教えてあげる。VSCodeの基本設定 -> 設定 -> ワークスペースの設定でも設定できる

```json
{
    "python.pythonPath": "${workspaceFolder}/bin/python"
}
```

### packageをメンバーと共有する方法
環境が構築できても、今回はメンバーとパッケージ等を共有する必要がある。
pip freeze コマンドを実行すると、インストールしたパッケージの一覧が出力されるので、この情報をファイルに保存して、プログラムのソースコードと一緒にバージョン管理するのが一般的のよう。その際は```requirements.txt```というものを作りそれをメンバーに配るのが一般的らしい。

参考ページ [サードパーティ製パッケージと venv](http://bootcamp-text.readthedocs.io/textbook/6_venv.html)

```bash
$ pip freeze > requirements.txt
```
プロジェクトをGitにあげる際には[env name]は重いので```.gitignore```に入れる。

```txt:.gitignore
/[env name]/
```

メンバーはvenvで作った仮想環境がactivateになっている状態で下記を実行すればパッケージがインストールされる

```
$ pip install -r requirements.txt
```

以上！！
なんか間違ってたら教えて下さい。すごい勢いで直します。

## 参考
判断する上で非常に参考になった記事。
[pyenvが必要かどうかフローチャート](https://qiita.com/shibukawa/items/0daab479a2fd2cb8a0e7)

