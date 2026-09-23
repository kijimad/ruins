// Package uicore は UI プリミティブの実体。保持型ツリーの Widget と、その配置・描画を提供する。
//
// # 層の位置と可視性
//
// ここは Atomic Design の atom にあたる。import できるパッケージは depguard の
// ui_core_inbound_guard が列挙し、画面が名指しできるシンボルは lintrule の
// TestScreenLayerUICoreSurface が許可制で絞る。
//
// 面は2つに分ける。配置もできる Widget は部品が扱い、描くだけの Drawable を画面へ見せる。
// 画面に Layout を見せると絶対座標で画面を組めてしまい、レイアウトエンジンを迂回する経路が
// 型に開く。置き場所を決めるのは部品の仕事にする。
//
// # 責務
//
//   - 状態はインスタンスが所有する。パッケージレベルの可変状態を持たないので、
//     複数の UI を並行に構築・更新しても競合しない
//   - 配置の計算は furex の flexbox へ委譲する。Container と FlexColumn が唯一の接続点で、
//     depguard がレイアウトエンジンの import をこのパッケージだけに制限する
//   - 行分割 WrapText は UAX#14 の segmenter へ委譲し、日本語も英語も同じ規則で折り返す
//   - 描画は Canvas の裏に閉じる。テストは記録用の実装で ebiten 無しに
//     レイアウトとテキストを検証できる
//
// # ウィジェットの使い分け
//
// 葉は子を持たず自分の矩形だけを描く。「何を描くか」で選ぶ。
//
//   - Text: 1行の文字
//   - Graphic: 画像1枚。矩形へ縦横比を保って収める。一覧のアイコンに使う
//   - NineSlice: テクスチャを9スライスで伸ばす。意匠画像の枠付き背景。選択バー・タイトルバー・入力枠
//   - Box: 単色の塗りと枠。BoxStyle.Radius で角丸にできる。theme 色のパネル背景など、
//     テクスチャを持たない箱はこちら
//
// 入れ物は子を主軸方向に並べる。用途でコンストラクタを選ぶ。
//
//   - VBox: 縦積み。各行は固定の rowH
//   - Row: 横並び。列ごとの幅を指定する
//   - Panel: 背景 style 付きの縦積み。独自配置の画面が意匠だけ部品へ合わせるのに使う
//   - FlexColumn: 伸縮する縦配置。Grow で余りを埋め Height で固定する。項目数で伸びる画面はこちら
//
// Group は「配置済み」の子を束ねるだけで再配置しない。子を並べる Container との違いはここにある。
// FlexColumn で個別配置した内容へ Box の背景層を重ねる、といった合成に使う。
//
// 単色背景の入り口は2つ。子を持つ入れ物の背景は Container.SetStyle、個別配置した内容の背後へ
// 層として敷くなら Box を Group に入れる。どちらも意匠は BoxStyle で表す。テクスチャ背景は
// SetBackgroundNineSlice か NineSlice 葉を使う。
package uicore
