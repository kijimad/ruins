![coverage](https://kijimad.github.io/ruins/cov/coverage.svg)

![Ruins](docs/steam/generated/library_header.png)

ローグライク。

- [Steam](https://store.steampowered.com/app/4791810/Ruins/)
- [Play](https://kijimad.github.io/ruins/)
- [Test Report](https://kijimad.github.io/ruins/cov/)
- [Raw Spec](https://kijimad.github.io/ruins/raw-spec/)
- [Godoc](https://pkg.go.dev/github.com/kijimaD/ruins)

## Play Images

<!-- VRT_IMAGES -->

各画像はゴールデンテストで自動生成される。

## キーボード操作

### ダンジョン探索

**移動**

| キー | 動作 |
| --- | --- |
| ↑ / ↓ / ← / → | 上下左右の移動 |
| Shift + 矢印2方向 | 斜め移動 |
| . (ピリオド) | その場で待機 |

**行動**

| キー | 動作 |
| --- | --- |
| Enter | 足元・隣接の対象と相互作用する |
| Space | インタラクションメニューを開く |
| G | 足元のアイテムを拾う |
| F | 射撃モード |
| 1〜5 | 武器スロットの切り替え |

**動詞コマンド**（アイテム操作画面を開く）

| キー | 動作 |
| --- | --- |
| Shift + X | 調べる |
| D | 置く |
| E | 食べる・飲む |
| R | 読む |
| T | 使う |

**画面・情報**

| キー | 動作 |
| --- | --- |
| M | ダンジョンメニュー |
| L | フィールド情報 |
| N | オーバーワールド地図（オーバーワールドにいるときのみ） |

**カメラ操作**

| キー | 動作 |
| --- | --- |
| Z / C | カメラを45度ずつ左回り / 右回りに回転 |
| PageDown | ズームアウト |
| PageUp | ズームイン |

### メニューナビゲーション

| キー | 動作 |
| --- | --- |
| ↑ / ↓ | 項目の上下移動 |
| ← / → | グリッド表示時の左右移動 |
| Tab | 次の項目へ移動 |
| Shift + Tab | 前の項目へ移動 |
| Enter | 項目選択・決定 |
| Escape | キャンセル・戻る |

## 開発

依存関係はDockerfileを参考にする。

```
$ make help
```

## 設計ドキュメントの状況

`docs/design` の frontmatter から自動生成される。`go run . designdoc list` で絞り込める。

<!-- DESIGN_STATUS -->

## Reference

ゲーム作成で参考にしたコード等。

- https://github.com/x-hgg-x/sokoban-go
  - 最初にコピペして作成をはじめ、改変していった
  - ECSの使い方まわりで参考にした
- https://github.com/x-hgg-x/goecsengine
  - ゲームステートまわりで参考にした
- https://bfnihtly.bracketproductions.com/
  - 設定ファイルによるファクトリ、ゲームログまわりを参考にした
- https://krkrz.github.io/krkr2doc/kag3doc/contents/
  - サウンドノベルに必要な記法を参考にした
- https://ebitengine.org/en/examples/raycasting.html
  - レイキャストの実装の参考にした
- https://cataclysmdda.org/
  - ローグライクシステムの参考にした
- ゲームシステム面で、KONAMIのビデオゲーム『パワプロクンポケット』シリーズ10・11・12を参考にした
  - 途中の方針転換で、あまり残っていない

使用した素材類。

フォント。

- http://jikasei.me/font/jf-dotfont/
- https://github.com/googlefonts/morisawa-biz-ud-gothic
- https://font.download/font/augustus

画像。

- https://www.pixilart.com
- https://pixabay.com/photos/forest-fog-woods-trees-mystical-3394066/
- https://pixabay.com/photos/beer-drink-alcohol-heineken-bar-5940890/
- https://pixabay.com/photos/lost-places-monastery-past-masonry-4019367/
