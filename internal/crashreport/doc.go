// Package crashreport は panic 時にスタックトレースを構造化 JSON でファイルへ残す。
// Guard を必ず通る関数の先頭に defer で置くと、その呼び出し内の panic を捕らえて1件のファイルに
// 保存し、握りつぶさず再 panic して本来どおりプロセスを落とす。保存先は UserConfigDir の下で、
// desktop のみ実ファイルを作り WASM は何もしない。
package crashreport
