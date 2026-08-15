// Package vrt はビジュアルリグレッションテストの共通基盤を提供する。
//
// 公開APIは役割で5群に分かれる。
//
// # ホスト・ロック・リソース
//
//   - RunTestMain: TestMain から呼ぶ。ebiten ループ内で全テストを走らせ ebiten.NewImage 等を使えるようにする
//   - WithUILock: ebitenui のグローバル状態への並行アクセスを直列化する
//   - SharedUIResources: 共有のUIリソースを一度だけ読み込んで返す
//
// # ワールド構築
//
//   - InitVRTWorld: 固定シードの素の決定的ワールドを作る
//
// # ゴールデン比較。ピクセル差分で pass/fail を判定する
//
//   - AssertContainerGolden: ウィジェットのコンテナを固定する
//   - AssertScreenGolden: 任意の画面描画を固定する
//   - AssertStateGolden: ステートスタックを全段描画して固定する。世界の上にUIを重ねた自然な実画面に使う
//
// # 画像生成。比較はせず保存用のPNGを返す
//
//   - RenderPNG: ステートを構築して描き、PNGを返す。draw が nil なら全段描画、非 nil なら任意レンダラ
//
// # 補助
//
//   - States: ステート列をビルダー関数へ変換するアダプタ
package vrt
