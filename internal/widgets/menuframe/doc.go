// Package menuframe はメニュー UI の部品庫。画面が組み合わせて使う部品と、その寸法を持つ。
//
// # 層の位置
//
// UI は theme(トークン)→ styled(一覧のデータ型)→ widgets/uicore(プリミティブ実体)→
// 本パッケージなどの部品 → 画面 states・menuloop の順に積む。
// 意匠(枠・行高・余白・色・テクスチャ)は部品と theme が持ち、画面はデータ・列の宣言・文言・
// 配置だけを書く。この積み順は depguard が守る。レイアウトエンジンを import できるのは
// プリミティブ実体だけに制限する。
//
// # 部品
//
//   - TabScreen: 見出し・タブ帯・一覧・下端固定フッタを持つ全画面モーダル。密な一覧用
//   - PanelScreen: 見出し・一覧・フッタを内容の高さで積む上端固定パネル。コマンドメニュー用
//   - PanelScreenDense: PanelScreen の密行版。キー一覧のような表を小さなパネルに収める
//   - PanelBox: パネルテクスチャの箱。独自配置の画面が意匠だけを部品へ合わせる
//   - InputBox: 1行入力欄の箱
//   - RenderList: 行データ Row と列宣言 styled.Col から、選択強調・ページ送り付きの行列を組む
//   - FormScreen: 見出し・入力欄・エラー・ヒントを画面中央へ縦に積む1項目の入力画面
//   - SplitScreen: 見出し・左の一覧・右の詳細枠・下の説明を持つ全画面。SplitList が左枠の一覧を組む
//   - TitleScreen: 背景を透かし、メニューを左下へ、版などの注記を右下へ置くタイトル画面
//
// # 寸法
//
// モーダルの矩形 ModalRect・WindowRect と、1ページに収められる行数 ListCapacity を持つ。
// カーソルの改ページと描画のページングが同じ値を使い、ずれない。
package menuframe
