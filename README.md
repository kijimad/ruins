ローグライク。

- [play](https://kijimad.github.io/ruins/)
- [godoc](https://kijimad.github.io/ruins/godoc/pkg/github.com/kijimaD/ruins/internal/)

## Play Images

<img src="./vrtimages/MainMenu.png" width="50%" /><img src="./vrtimages/Dungeon.png" width="50%" />

<img src="./vrtimages/Town.png" width="50%" /><img src="./vrtimages/InventoryMenu.png" width="50%" />

<img src="./vrtimages/CraftMenu.png" width="50%" /><img src="./vrtimages/EquipMenu.png" width="50%" />

<img src="./vrtimages/LoadMenu.png" width="50%" /><img src="./vrtimages/SaveMenu.png" width="50%" />

<img src="./vrtimages/GameOver.png" width="50%" /><img src="./vrtimages/Message.png" width="50%" />

<img src="./vrtimages/DebugMenu.png" width="50%" /><img src="./vrtimages/ShopMenu.png" width="50%" />

各画像は全自動でキャプチャされる。最新なことをCIで保証している。

## キーボード操作

### ダンジョン探索
- **W** - 上
- **S** - 下
- **A** - 左
- **D** - 右
- **C / PageDown** - ズームアウト
- **E / PageUp** - ズームイン
- **マウスホイール** - ズーム操作

### メニューナビゲーション
- **↑ / ↓** - 項目の上下移動
- **← / →** - グリッド表示時の左右移動
- **Tab** - 次の項目へ移動
- **Shift + Tab** - 前の項目へ移動
- **Enter** - 項目選択・決定
- **Escape** - キャンセル・戻る

## 開発

依存関係はDockerfileを参考にする。

```
$ make
run                  実行する。スクショのキーを指定している
test                 テストを実行する
build                ビルドする
vrt                  各ステートでスクショを取得する
fmt                  フォーマットする
lint                 Linterを実行する
aseprite             asepriteでパッキングする
toolsinstall         開発ツールをインストールする
check                一気にチェックする
memp                 実行毎に保存しているプロファイルを見る
pprof                サーバ経由で取得したプロファイルを見る。起動中でなければならない
help                 ヘルプを表示する
```

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

画像。

- https://www.pixilart.com
