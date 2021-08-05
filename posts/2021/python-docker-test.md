---
title: PythonのテストでElasticsearchをDockerで起動して、テストが終わったら止める
cover: https://pon-blog-media.s3-ap-northeast-1.amazonaws.com/media/python-docker-test.jpeg
date: 2021/05/25
id: python-docker-test
description: Pythonのテスト内でDockerを起動して、終わったらコンテナを止める機構を作ったので共有
tags:
    - Python
    - Docker
    - Elasticsearch
---

## Overview

こんにちは[pon](https://twitter.com/po3rin)です。

PythonでElasticsearchとの接続もテストしたい！しかし、これをすると開発者のローカルにElasticsearchをインストールさせる事になる。そこでElasticsearch on Dockerになるわけだが、手動でDocker起動して、インデックス作って、テストを走らせる...という手順は煩雑だし、初めて開発を始めるユーザーにとってテストを行うハードルが上がる。

そこで今回はPythonのテスト内でElasticsearch on Dockerを起動して、終わったらコンテナを止める機構を作ったのでメモがてら共有。

## EsManagerの作成

まずはElasticsearchのプロセスを管理するESManagerを作成する。

```py
import docker
from docker.models.containers import Container

class ESManager(ESManager):
    def __init__(self) -> None:
        self.client = docker.from_env()
        self.container = Container()

    def __enter__(self):
        self.client = docker.from_env()
        self.container = Container()
        return self

    def __exit__(self, exception_type, exception_value, traceback):
        self.stop()

    def run(self) -> None:
        try:
            c = self.client.containers.run(
                'docker.elastic.co/elasticsearch/elasticsearch:7.10.2',
                ports={
                    '9200/tcp': 9200,
                    '9300/tcp': 9300,
                },
                environment={
                    'discovery.type': 'single-node',
                    'ES_JAVA_OPTS': '-Xms1024m -Xmx1024m',
                },
                detach=True,
            )
        except Exception as e:
            raise ESManagerError('failed to run Elasticsearch container') from e

        timeout = 120
        stop_time = 3
        elapsed_time = 0
        while c.status != 'running' and elapsed_time < timeout:
            sleep(stop_time)
            elapsed_time += stop_time

    def stop(self) -> None:
        self.container.stop()
```

主に使うのは[Python Docker SDK](https://docker-py.readthedocs.io/en/stable/)で各種Docker操作がPythonからできて便利。

https://docker-py.readthedocs.io/en/stable/

```__enter__```と```__exit__```メソッドは```with```句で利用するための実装です。下記のように```with```と利用することでコンテナの止め忘れを防止できる。

```py
with ESManager() as es_manager:
    es_manager.run()

    ## なにやらの操作...
```

また、```run```ではElasticsearchが利用できるようになるのを待つ必要があるので簡単なwait機構を入れています。

## テスト用インデックス生成

続いてElasticsearchにインデックスを作る関数が必要なので作成する。ここでは自作ツールのeskeeperを利用して設定ファイルから一撃でインデックスを作成できるようにしてある。便利なので是非使って見て欲しい。

[![eskeeper](https://github-link-card.s3.ap-northeast-1.amazonaws.com/po3rin/eskeeper.png)](https://github.com/po3rin/eskeeper)

eskeeperではYAMLファイルにインデックス名とJSONでのインデックスの設定ファイルのパスを指定するだけであとは勝手にインデックスが作成される。

```yml
index:
  - name: index1
    mapping: eskeeper/mapping/index1.json

  - name: index2
    mapping: eskeeper/mapping/index2.json
```

実行

```sh
eskeeper < eskeeper/eskeeper.yml
```

このツールの作成経緯や実装の解説は記事にしているので是非読んでみて欲しい。

[IaCを意識したCLI開発のエッセンス](https://www.m3tech.blog/entry/iac-aware-cli)

一方でeskeeperはGo製なのでPython版がないのでsubprocessでコマンドを叩いてあげる必要があるので```subprocess```モジュールを利用している。

```py
import subprocess


def setup_index(es_host: str, config_path: str = 'eskeeper/es.yml') -> None:
    subprocess.run(f'eskeeper -e http://{es_host} < {config_path}', check=True, capture_output=True, shell=True, text=True)
```

## EsManagerを使ってテストする

これでES on Dockerをテスト内で起動、停止ができる。

```py
class TestES(unittest.TestCase):
    def setUp(self):
        super().setUp()

        # テストしたいESクライアントクラスはお好みで
        self.es_client = ESClient()

        # ES on Docker起動
        self.es_manager = ESManager()
        self.es_manager.run()

        #ESがAvailableになるのをpingして待つ...

        # インデックス作成
        setup_index(es_host="localhost:9200", config_path='eskeeper/eskeeper.yml')

    def test_search(self) -> None:
    	# 何かしら素敵なテスト


    def tearDown(self) -> None:
    	#後片付け
        super().tearDown()
        self.es_client.close()
    	self.es_manager.stop()
```

## まとめ

開発者がESを自分で準備しなくてもテストコマンド1発でES含めたテストができるので便利。
