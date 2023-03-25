---
title: cruft実践入門 ~cookiecutter templateの変更に追従する~
cover: img/gopher.png
date: 2023/03/05
id: m3-cruft
description: Go is a programming language
tags:
    - golang
    - markdown
draft: true
---

## Overview

エムスリーエンジニアリンググループ AI・機械学習チームでソフトウェアエンジニアをしている中村([po3rin](https://twitter.com/po3rin)) です。検索とGoが好きです。

AI・機械学習チームでは開発の効率化のため、プロジェクトの雛形を自動的に生成するcookiecutterのプロジェクトtemplateを利用しています。

https://www.m3tech.blog/entry/kubernetes-api-project-template

しかし、templateから作成したプロジェクトはtemplateが新しくなった場合に、その変更に追従するのが困難になります。そこで、今回は[cruft](https://github.com/cruft/cruft/)というツールを導入して、最新cookiecuttertemplateへの追従を楽にできるようにしました。今回はcruftの紹介と、それをどのように導入したかをお話しします。

<!-- more -->

[:contents]

## cookiecutterの課題

cookiecutterを使ったプロジェクト作成を行った後に、template側に重要な更新が入った場合、そのtemplateから作ったプロジェクトにその変更を加えたい時があります。例えばツールのバージョンアップや新しいセキュリティチェックをCIに導入した場合、templateの更新はもちろん、そのtemplateから作った全てのプロジェクトにその変更を加えていく必要があります。

![](./cookiecutter.png)

どのプロジェクトに変更を追加したかを管理するのは非常に困難で、弊チームではスプレッドシートを使って変更を適用したプロジェクトにチェックを入れてもらい、まだ変更を取り込んでいないプロジェクトを管理するなどの手作業が発生していました。

そこでcruftの登場です。

## cruftとは

cruftを使用すると、templateの最新状態への追従や、変更のチェックなどを簡単に行えます。cookiecutter template機能と完全に互換性があるので、cookiecutterを使っているチームはすぐに取り入れることができます。

リポジトリはこちらです。
https://github.com/cruft/cruft/

このツールを導入することで、cookiecutterで作った既存のプロジェクトが最新のtemplateへ追従するのを楽にしました。

## 既存プロジェクトへのcruft導入

cookicutter templateから作った既存プロジェクトにcruftを導入する方法を解説します。まずは下記を実行します。

```
cruft link https://rendezvous.m3.com/ai/cookiecutter-m3gokart
```

そうすると、cookiecutterからプロジェクトを作る時と同じように、templateの値についての質問されるので、プロジェクトを作った時と同じ値を回答します。そうすると下記のように`cruft.json`が作成されます。

```json
{
  "template": "https://github.com/m3dev/cookiecutter-gokart",
  "commit": "25b2ea60fd1b3145908b750fc0e42e130913c7c5",
  "checkout": null,
  "context": {
    "cookiecutter": {
      "project_name": "xxx",
      "package_name": "xxx",
      // ...
      "_template": "https://github.com/m3dev/cookiecutter-gokart"
    }
  },
  "directory": null,
}
```

確実にdiffが出てしまうファイルに関しては`cruft.json`に`skip`フィールドを追加することができます。

```json
{
  "template": "https://github.com/m3dev/cookiecutter-gokart",
  ...
  "skip": [
    "xxx",
    ".venv",
    "poetry.lock",
    "poetry.toml"
  ]
}
```

この状態で下記を実行すると`cruft.json`の`commit`が最新のcookiecutterのcommitに追従していることを確認できます。

```sh
cruft check
```

もし、上記コマンドに失敗した場合は、`cruft.json`の`commit`とcookiecutterの最新コミットと差分があるので、その差分をチェックし、可能ならそのまま取り込みます。

```
cruft update
```

```cruft check```に失敗している場合、CUIでどのアクションを行うかを問われます。`cruft.json`の`commit`から最新のコミットまでの変更を確認したい場合は[v]、変更を適用したい場合は[y]を選択します。適用したくない場合は[s]を選択します。これで簡単にcookiecutterの最新のcommitに追従できます。

`cruft.json`の`commit`から最新のコミットまでの変更ではなく、最新cookiecutterと現在のプロジェクトの差分を丸々見たい場合は下記を実行します。

```
cruft diff
```

古い既存プロジェクトに導入する場合は複数のdiffが出るので、diff単位で確認しながら適用したいdiffだけを適用する下記のコマンドが便利です。

```sh
cruft diff | git apply - && git add -p
```

## CIの設定

cruftを使って常に最新の状態を確認して必要なdiffを取り込んだかをチェックするために、弊チームでは`cruft check`を行うCIを設定しました。弊社ではGitLabを利用しているので、GitLabでの設定方法を主に説明します。

CIで`cruft check`するための設定を`.gitlab-ci.yml`追加します。この際にマージリクエストにコメントを残したいので[gitlab-comment](https://github.com/yuyaban/gitlab-comment)を導入しています。この際に`GITLAB_ACCESS_TOKEN`を作っておきます。


```yml
.cruft_check:
  image: <<何かしら素敵なpython image>>
  before_script:
    # gitのセットアップなど...
    # ...

    - pip install --upgrade pip
    - pip install cruft
    - wget -q https://github.com/yuyaban/gitlab-comment/releases/download/v0.2.3/gitlab-comment_0.2.3_linux_amd64.tar.gz
    - tar -zxvf gitlab-comment_0.2.3_linux_amd64.tar.gz
    - chmod +x gitlab-comment
    - mv gitlab-comment /usr/bin/gitlab-comment
    - rm gitlab-comment_0.2.3_linux_amd64.tar.gz
  script:
    - |
      export GITLAB_ACCESS_TOKEN=$CRUFT_GITLAB_TOKEN
      cruft check || exit_code=$?
      if [ $exit_code -ne 0 ]; then
        export GITLAB_ACCESS_TOKEN=$CRUFT_GITLAB_TOKEN
        gitlab-comment post -k cruft_check -u 'Comment.HasMeta && Comment.Meta.TemplateKey == "cruft_check"' --var target:"${CI_JOB_NAME}"
        exit 1
      fi

cruft:
  stage: test
  extends: .cruft_check
  only:
    - merge_requests
```

このCIの設定は既存プロジェクトにするに導入できるように、template化してチーム内で利用できるようにしてあります。詳しくはGitLab CIのtemplate基盤を構築した際に書いた下記のブログをご覧ください。

https://www.m3tech.blog/entry/gitlab-include

そして、コメントアップデートするために`gitlab-comment.yaml`を追加します。これで`gitlab-comment`がどのコメントを更新すれば良いかを判別できます。

```yml
---
post:
  default: |
    hello gitlab-comment !
  cruft_check:
    # update: 'Comment.HasMeta && Comment.Meta.TemplateKey == "cruft_check"'
    template: |
      最新のcookiecutterに追従できていません。ローカルで ```cruft update``` を実行して、最新のcookiecutterに追従しましょう。
      https://rendezvous.m3.com/ai/torenia/-/blob/main/README.md#最新cookiecutterへの追従
```

これでCIで`cruft check`をして、もし最新版に追従していない場合はマージリクエストにコメントが残ります。cruftの運用ドキュメントなどへのリンクも置いておくと便利です。

![](./cruft-comment.png)

`cruft diff`の結果をコメントに残すことも考えましたが、弊チームの運用では、全てのプロジェクトに確実に差分が出てしまうので、毎回マージリクエストのたびにコメントが投稿されてしまいます。そのため今回はCIに`cruft diff`の結果をアップロードする運用は見送りました。

## cookiecutter template自体へのcruft導入

cookiecutter templateにはデフォルトでcruftの設定が作成されると便利です。そこで、templateに`cruft.json`やCIの設定を追加しておきます。

しかし、ここで`cruft.json`に設定した`commit`フィールドはどんどん古くなっていくので、cookiecutterからプロジェクトを作成した段階で最新のコミットにするようにPost-Generate Hooksを設定しました。Post-Generate Hooksはプロジェクト作成時に実行されるスクリプトを定義できる機能です。

Pre/Post-Generate Hooks機能に関するドキュメントはこちら

https://cookiecutter.readthedocs.io/en/1.7.2/advanced/hooks.html

プロジェクト作成時に`cruft update`が走るようにするコード例は下記になります。`cruft update`コマンドに`-s`オプションをつけることで、コミットIDの更新だけを行ってくれます。

```
import subprocess
from pathlib import Path
import shutil

rootpath: Path = Path.cwd()

# setup cruft 
subprocess.check_call(['poetry', 'run', 'cruft', 'update', '-s'])
```

これをドキュメントの通り`hooks`ディレクトリに入れておけばcookiecutterからのプロジェクト作成時に新しいコミットIDに書き換えてくれます。

## チームへの導入

cruftをチームへ導入する際の障壁はcookiecutterで作った既存の全てのプロジェクトに導入する必要があることです。ここをパッと自動化することは難しいため(何か素敵なアイデアがあれば教えてほしい)、チームのみんなに協力してもらう必要があります。

そのために僕が行った取り組みは下記です。

* 既存プロジェクトへの導入のためのドキュメントを書く
* 参考として導入事例のプロジェクトを一個作る
* 簡単にCIをセットアップできるようにGitLab CI テンプレートを用意
* チームで週1で行っている技術共有会でcruftを紹介。運用方法について合意を取る。

このように大量のプロジェクトに変更を行う場合は、チームメンバーに協力してもらうための障壁を落とす必要があります。今までセキュリティチェックツールなどを導入する場合には同じようなことをしていましたが、cruft導入を突破してしまえば楽なのでがんばりましょう。

## まとめ

cruftの導入によって、最新版cookiecutterへの追従や変更の確認を簡単に行う方法を紹介し、実際にチームに導入する方法なども紹介しました。

## We're hiring!

エムスリーではエンジニアの開発体験をゴリゴリ改善する仲間を募集しています！！我こそは！という方はぜひ以下からご応募ください！

[https://jobs.m3.com/product/:embed:cite]
