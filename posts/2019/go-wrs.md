---
title: 任意の重みに従ってランダムに値を返す「Weighted Random Selection」をGoで実装する！
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/weight.jpeg
date: 2019/08/19
id: go-wrs
description: 今回は Go で 「Weighted Random Selection」 の実装方法を紹介します。
tags:
    - Go
    - Algorithm
---

## Weighted Random Selection とは

とある重み(確率分布)を元に要素をランダムに選択するやつです。numpyで言うと ```numpy.random.choice``` に当たります。下記は第一引数 ５([0,1,2,3,4]) から3つを確率分布pでランダムに選択する関数です。

```python
>>> np.random.choice(5, 3, p=[0.1, 0, 0.3, 0.6, 0])
array([3, 3, 0])
```

ランダムな選択に重複を許可しない場合は引数に ```replace=False``` を指定します。

```python
>>> np.random.choice(5, 3, replace=False, p=[0.1, 0, 0.3, 0.6, 0])
array([2, 3, 0])
```

今回はGoでこの処理を行う際の実装を紹介します。

## Go による Weighted Random Selection

今回は最もシンプルな Linear Scan アルゴリズムで実装します。やることは[０~weightの合計値]の間でランダムに基準となる値を選び、基準からweightを順に引いていき、０以下になったらそれが選択されます。

![img1](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2019/1566172800/8ee51ac1-0368-0b9b-43ff-1d112763b3af.png)

早速実装していきます。下記はvの中からwの確率分布に従って1つだけ値を取得する関数です。

```go
// 0 ~ max までの範囲でランダムに値を返す
var randGenerator = func(max float64) float64 {
	rand.Seed(time.Now().UnixNano())
	r := rand.Float64() * max
	return r
}

func weightedChoiceOne(v int, w []float64) float64 {
	// v を slice　に変換
    // ex) 5 -> [0, 1, 2, 3, 4]
	vs := make([]int, 0, v)
	for i := 0; i < v; i++ {
		vs = append(vs, i)
	}

    // weightの合計値を計算
	var sum float64
	for _, v := range w {
		sum += v
	}

    // weightの合計から基準値をランダムに選ぶ
	r := randGenerator(sum)

    // weightを基準値から順に引いていき、0以下になったらそれを選ぶ
	for j, v := range vs {
		r -= w[j]
		if r < 0 {
			return v
		}
    }
    // should return error...
	return 0
}
```

最後の```return 0``` はたまたま１つも選ばれなかった(基準がweightの合計と丁度一致した時など)場合に到達します。確率的にほとんどありませんが、ここはこの関数を使う状況によってエラーハンドリングや空で返すなどの対策が考えられます。

上記のコードを少し変更すれば選ぶ数の指定 & 重複排除も実装できます。ポイントは選ばれた物をスライスから排除しておくことです。


```go
func weightedChoice(v, size int, w []float64) ([]float64, error) {
	// v を slice　に変換
    // ex) 5 -> [0, 1, 2, 3, 4]
	vs := make([]int, 0, v)
	for i := 0; i < v; i++ {
		vs = append(vs, i)
	}

    // weightの合計値を計算
	var sum float64
	for _, v := range w {
		sum += v
	}

	result := make([]float64, 0, size)
	for i := 0; i < size; i++ {
		r := randGenerator(sum)

		for j, v := range vs {
			r -= w[j]
			if r < 0 {
				result = append(result, float64(v))

                // weightの合計値から選ばれたアイテムのweightを引く
				sum -= w[j]

                // 選択されたアイテムと重みを排除
				w = append(w[:j], w[j+1:]...)
				vs = append(vs[:j], vs[j+1:]...)

				break
			}
		}
	}
	return result, nil
}

```

選択されたアイテムと重みの削除のコードが少し特殊に見えますが、下記の公式Wikiを参考に実装しています。
[https://github.com/golang/go/wiki/SliceTricks#delete](https://github.com/golang/go/wiki/SliceTricks#delete)

これを使えば与えられたweightにしtがってランダムに値を返します。

```go
func main() {
	r1 := weightedChoiceOne(5, []float64{0.1, 0.1, 0.2, 0.9, 0.1})
	r2, _ := weightedChoice(5, 4, []float64{0.1, 0.9, 0.2, 0.3, 0.1})
	fmt.Println(r1) // 3
	fmt.Println(r2) // [1 3 2 0]
}
```

これで Weighted Random Selection が実装できました！ 今回は最もシンプルな Linear Scan での実装を紹介しましたが、そのほかのアルゴリズムはこのサイトが勉強になります。https://blog.bruce-hill.com/a-faster-weighted-random-choice

コードは Go Playground にあげておきます。
https://play.golang.org/p/-vqQEvwCi44

