// Package menuframe は画面レベルの UI 足場を組み立てる。
//
// タブ付きモーダル NewTabScreen、内容サイズに縮む小パネル NewPanelScreen、
// それらがログ領域と重ならないための基準を与える logTopY・CenterWindowRect を持つ。
//
// 入力データは states の Fetch が用意し、本パッケージはその見た目の器だけを組む。
// ゲーム世界の直描きには関わらない。
package menuframe
