// Package framedbg は枠付き背景の即時描画を提供する。
//
// HUD やゲームプレイ上のパネル、ログ領域のように、毎フレーム screen へ直接描く背景に使う。
// メニューの保持描画は internal/ui の Canvas 系が担うので、そちらとは別系統。用途で分離している。
//
//   - Style: 枠線・背景・上下辺の立体線を表す描画スタイル。
//   - Draw: Style に従って枠付き背景を screen へ即時描画する。
//   - PanelStyle: ゲーム内パネルの既定スタイル。
package framedbg
