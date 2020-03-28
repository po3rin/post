# 超小技!! Makefile に help をつけて「こいつ...できる!」と言われたい

開発時のめんどくさいコマンドを打ってる時間は極力減らしたいですよね。そこでMakefileなのですが、Makefileのコマンドが多くなったり、複雑になればなるほど「このコマンドなんだっけ」となります。

そこで今回は簡単に Makefile に help をつける方法を紹介します。これをつければチームメンバーに格の違いを見せつけることができます(多分

##Go言語での開発用のMakefileを例に

今回はGo言語での開発で使う Makefile の基本型を例に紹介します。もちろんGo言語をインストールしてなくても help は動くのでコピペで試せます。

手順は2つ
- target に help を追加。 grepと正規表現を使ったコマンドをその下に書きます。
- target の右隣に ## をつけてその後ろに説明書きを追加します。

実際のMakefileは下のようになります。ちなみにコマンド行の行頭にはtabでインデントしないとエラーになるので注意(僕は VScode のインデント設定がスペース4になっていてハマった淡い記憶がある)。grepコマンドの行頭の @ は実行時にコマンドを非表示にしてくれます。

```bash:Makefile

GOBUILD=go build
GOCLEAN=go clean
GOTEST=go test
GOGET=go get
BINARY_NAME=tryhelp
BINARY_UNIX=$(BINARY_NAME)_unix

all: test build ## go test & go build
.PHONY: build
build: ## build go binary
	$(GOBUILD) -o $(BINARY_NAME) -v
.PHONY: test
test: ## go test
	$(GOTEST) -v ./...
.PHONY: clean
clean: ## remove go bainary
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
.PHONY: run
run: ## run go bainary
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)
.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
```

実行してみましょう。

```bash

$ make help
all                            go test & go build
build                          build go binary
clean                          remove go bainary
help                           Display this help screen
run                            run go bainary
test                           go test
```

まごう事なき help です。これで新しくチームに入ったルーキーも道に迷わずに済みそうです。

printf内の値30を変更することで、ターゲットと説明の幅を調整する。
```| sort```を削除すると、makefileに書いてある順番で出てくる。

今回使った正規表現はこちらが勉強になる
[正規表現の基本](https://qiita.com/sea_ship/items/7c8811b5cf37d700adc4)

