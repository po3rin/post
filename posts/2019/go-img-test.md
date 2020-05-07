---
title: Goによる画像処理テストパターンの考察とまとめ
cover: https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/media/camera.jpeg
date: 2019/12/01
id: go-img-test
description: 僕が他の画像処理パッケージのコードリーディングをしてまとめた画像処理テストの実装パターンを紹介します。
tags:
    - Go
    - Image
---

この記事は [Go2 Advent Calendar 2019](https://qiita.com/advent-calendar/2019/go2)の1日目の記事です。

## 画像のテストパターン

### 1ピクセルずつ愚直にテスト

標準パッケージやOSSなどの様々なパッケージではRGBAの値を1ピクセルごと調べています。

```go
// in go/src/image/draw/draw_test.go
func eq(c0, c1 color.Color) bool {
	r0, g0, b0, a0 := c0.RGBA()
	r1, g1, b1, a1 := c1.RGBA()
	return r0 == r1 && g0 == g1 && b0 == b1 && a0 == a1
}

func TestDraw(t *testing.T) {
	// ...

	// 画像が処理されているかを1ピクセルごと調べる
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !eq(dst.At(x, y), golden.At(x, y)) {
				// test fail
			}
		}
	}
}
```

しかし愚直に1ピクセルごと調べていくと時間がかかるので、標準パッケージでは、その前に画像の大きさが一致するかや、1ピクセルだけチェックしてRGBAがあっているかを確認しています。下記は```image/draw```パッケージのテスト実装です。

```go
// 画像の大きさが一致するか
if !b.Eq(golden.Bounds()) {
	// fail
}

// (8,8)のRGBAがあっているかだけをチェック
if image.Pt(8, 8).In(r) {
	if !eq(dst.At(8, 8), test.expected) {
		t.Errorf("draw %v %s: at (8, 8) %v versus %v", r, test.desc, dst.At(8, 8), test.expected)
		continue
	}
}
```

上の例で1ピクセルだけチェックするのは、そもそも全く画像処理できてなかったパターン(全く予期せぬ画像ができてしまう時など)を前もって弾く為です。これである程度重いテストの前にテストを失敗させることができます。

### ストライドを使った効率化
さらに面白いテストのパターンもあります。下記は画像処理アルゴリズムのコレクションパッケージ [github.com/anthonynsimon/bild](https://github.com/anthonynsimon/bild) のテストで使われているequal関数です。

```go
func RGBAImageEqual(a, b *image.RGBA) bool {
	if !a.Rect.Eq(b.Rect) {
		return false
	}

	for y := 0; y < a.Bounds().Dy(); y++ {
		for x := 0; x < a.Bounds().Dx(); x++ {
			pos := y*a.Stride + x*4
			if a.Pix[pos+0] != b.Pix[pos+0] {
				return false
			}
			if a.Pix[pos+1] != b.Pix[pos+1] {
				return false
			}
			if a.Pix[pos+2] != b.Pix[pos+2] {
				return false
			}
			if a.Pix[pos+3] != b.Pix[pos+3] {
				return false
			}
		}
	}
	return true
}
```

画像データの構造に着目してストライドの値を使ってRGBAを保持するスライスに直でアクセスしています。もちろん先ほどのテストパターンより高速です。画像のデータ構造とストライドに関しては下記の記事が大変勉強になります。

[画像データの構造](http://neareal.net/index.php?ComputerGraphics/ImageProcessing/TheStructureOfImageData)

### bytes.Equalでテスト

一方で ```image.RGBA.Pix``` は ```[]uint8``` なので ```bytes.Equal``` で一発でテストできます。画像処理フィルタパッケージ「disintegration/gift」ではこのように画像のテストがされています。

```go
func checkBoundsAndPix(b1, b2 image.Rectangle, pix1, pix2 []uint8) bool {
	if !b1.Eq(b2) {
		return false
	}
	if !bytes.Equal(pix1, pix2) {
		return false
	}
	return true
}
```

実はここまでに紹介した例よりもこちらの方が高速です。一方で、この方法だとどのピクセルが間違っているのかの情報が失われてしまいます。。その為、「パフォーマンス」と「テスト失敗時の情報の詳細度」のシーソーゲームです。テスト失敗時の情報の詳細度を全く気にしないのであれば ```reflect.DeepEqual``` でもいけます。速度はほとんど ```bytes.Equal``` を使ったテストと同じです。

### カラーモードに適したテスト

当然、画像のカラーモード次第で更に最適化したテスト実装があります。下記はグレースケールの画像のequal関数です。グレースケールならこれで十分でしょう。

```go
func GrayImageEqual(a, b *image.Gray) bool {
	if !a.Rect.Eq(b.Rect) {
		return false
	}

	for i := 0; i < len(a.Pix); i++ {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}
```


### テストパターン別のベンチマーク

先ほどパフォーマンスの話が出たので先ほど紹介したパターンをそれぞれベンチマークしてみましょう。ベンチマークは460px × 460px の同じ画像かをチェックするテストです。コードは [こちら](https://github.com/po3rin/go_playground/tree/master/try-img-test) にあるので興味のある方はどうぞ。

```bach
go test -bench=. ./...
goos: darwin
goarch: amd64
pkg: github.com/po3rin/try-img-test
BenchmarkEqNormal-12              84	  13948999 ns/op   //1ピクセルずつ愚直にテスト
BenchmarkEqWithStride-12         686	   1717456 ns/op   //ストライドを使った効率化
BenchmarkEqWithBytes-12         1070	   1113299 ns/op   //bytes,Equalを使ったテスト
BenchmarkEqWithReflect-12       1059	   1115034 ns/op   //reflect.DeepEqualを使ったテスト
PASS
```

1ピクセルずつ愚直に検査が当然遅いですね。ストライドを使った効率化したテストはパフォーマンスと失敗時の情報の詳細度のバランス的に良さそうです。テスト失敗時にどの程度の詳細度で情報が欲しいかで選んでいくと良いでしょう。


## テストケースの準備

画像をテストする方法を決めたらあとは期待するデータを準備するだけです。github.com/disintegration/gift では下記のようにテストデータを準備しています。

```go
testData := []struct {
	desc           string
	w, h           int
	r              Resampling
	srcb, dstb     image.Rectangle
	srcPix, dstPix []uint8
}{
	{
		"resize to fit (1, 1, nearest)",
		1, 1, NearestNeighborResampling,
		image.Rect(-1, -1, 4, 4),
		image.Rect(0, 0, 1, 1),
		[]uint8{
			0x00, 0x01, 0x02, 0x03, 0x04,
			0x05, 0x06, 0x07, 0x08, 0x09,
			0x0a, 0x0b, 0x0c, 0x0d, 0x0e,
			0x0f, 0x10, 0x11, 0x12, 0x13,
			0x14, 0x15, 0x16, 0x17, 0x18,
		},
		[]uint8{0x0c},
	},
	// ...
}
```

上のように ```[]uint8``` を準備しても良いですが、もっと大きな画像を扱う場合はどうしましょう。また、欲しい画像がコロコロ変わる場合にいちいち ```[]uint8``` をテストケースに詰め直すのも面倒な作業です。その為、場合によっては下記のように goldenfile を準備したテストを行うのが便利です。その際には goldenfile を生成するフラグを準備しておくと良いでしょう。

```go
var genGoldenFiles = flag.Bool("gen_golden_files", false, "whether to generate the TestXxx golden files.")

func TestResizePNG(t *testing.T) {
	tests := []struct {
		name           string
		goldenFilename string
	}{
		{
			name:           "x1.0",
			goldenFilename: "testdata/resize_golden_1.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// 何かしら面白い画像生成
			got := Convert()

			// goldenfile 生成フラッグが有効だったら goldenfileを生成して終了
			if *genGoldenFiles {
				goldenFile, _ := os.Create(tt.goldenFilename)
				defer goldenFile.Close()
				_ = png.Encode(goldenFile, got)
				return
			}

			// goldenfileから期待する画像を取得
			f, _ = os.Open(tt.goldenFilename)
			defer f.Close()
			want, _, _ := image.Decode(f)

			// 欲しい画像ができているかテスト
			if !reflect.DeepEqual(convertRGBA(got), convertRGBA(want)) {
				t.Errorf("actual image differs from golden image")
				return
			}
		})
	}
}
```

このテストは下記のようにフラグを使うことでテストではなくgoldenfile生成を行ってくれるようになります。

```bash
go test -gen_golden_files ./...
```

実際に github.com/golang/image パッケージではこのように goldenfile と生成フラグを使ったテストが実装されています。goldenfile生成には独自でフラグを用意する他にもビルドフラグで切り替えるパターンもあります。

また、画像処理のテストにおけるgoldenfileはPNGであることが望まれます。JPEG自体がlossy(情報が欠落する)な非可逆(元に戻せない)圧縮方式なので、一度image.ImageをJPEGに変換してしまうと、元の画像に復元することはできないからです。

![img1](https://pon-blog-media.s3.ap-northeast-1.amazonaws.com/2019/1575158400/3b8e8eab-ff95-9fec-a7c8-7e37cc6f6989.png)

## まとめ
いろんな画像処理パッケージのテストをのぞいて、画像処理のテストのパターンをまとめました。意外にもたくさんのパターンがあって驚きました。僕がまだ考えついていないテスト実装パターンがあると思うので、もっと良いテスト方法があったらぜひ教えてください！

