// Package menuframe は画面レベルの UI 足場を組み立てる。
//
// タブ付きモーダル NewTabScreen、内容サイズに縮む小パネル NewPanelScreen、
// それらがログ領域と重ならないための基準を与える LogTopY・CenterWindowRect を持つ。
// これらは以前 states と overlay に散在していた。states をデータ提供へ純化し、
// overlay パッケージを overlay 契約と詳細窓構築に絞るため、画面足場の責務をここへ集約する。
//
// 入力データは states の Fetch が用意し、本パッケージはその見た目の器だけを組む。
// ゲーム世界の直描きには関わらない。
package menuframe
