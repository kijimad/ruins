// Package menuframe はメニュー UI の部品庫。画面が組み合わせて使う部品と、その寸法を持つ。
//
// # 層の位置
//
// UI は theme(トークン)→ styled(一覧のデータ型)→ widgets/internal/ui(プリミティブ実体)→
// 本パッケージなどの部品 → widgets/ui(画面向けファサード)→ 画面 states・menuloop の順に積む。
// 意匠(枠・行高・余白・色・テクスチャ)は部品と theme が持ち、画面はデータ・列の宣言・文言・
// 配置だけを書く。プリミティブ実体は internal 可視性で画面から遮断され、レイアウトエンジンの
// import は depguard が実体パッケージだけに制限する。
//
// # 部品
//
//   - TabScreen: 見出し・タブ帯・一覧・下端固定フッタを持つ全画面モーダル。密な一覧用
//   - PanelScreen: 見出し・一覧・フッタを内容の高さで積む上端固定パネル。コマンドメニュー用
//   - PanelScreenDense: PanelScreen の密行版。キー一覧のような表を小さなパネルに収める
//   - PanelBox: パネルテクスチャの箱。独自配置の画面が意匠だけを部品へ合わせる
//   - InputBox: 1行入力欄の箱
//   - RenderList: 行データ Row と列宣言 styled.Col から、選択強調・ページ送り付きの行列を組む
//
// # 寸法
//
// モーダルの矩形 ModalRect・CenterWindowRect と、1ページに収められる行数 ListCapacity を持つ。
// カーソルの改ページと描画のページングが同じ値を使い、ずれない。
package menuframe
