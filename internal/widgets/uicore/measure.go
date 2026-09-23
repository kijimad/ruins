package uicore

import (
	"math"
	"sync"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// textMu は ebiten text/v2 の測定・描画と、測定結果キャッシュへのアクセスを直列化する。
// GoTextFaceSource の遅延グリフキャッシュ runeToBoolMap はスレッド安全でなく、共有フェイスへ
// 同時に測定・描画するとキャッシュが壊れる。本番の描画は単一ゴルーチンなのでロックは常に無競合で、
// 並列テストのときだけ直列化が効く。ebiten text へ触れるのは MeasureText と
// EbitenCanvas.DrawText の2箇所なので、この mutex で全経路を覆う。
var textMu sync.Mutex

// measureCache は (文字列, フェイス) ごとの測定結果を覚える。Text.Draw が位置決めのため毎フレーム
// MeasureText を呼ぶが、結果は決定的なので一度測れば使い回せる。フェイスは起動時に生成した
// *GoTextFace か *MultiFace のポインタ実体で、比較可能かつ不変なのでキーに使える。
//
// 動的文字列でキーが際限なく増えないよう上限で丸ごと捨てる。捨てても再測定で正しく埋め直すので、
// キャッシュミスが増えるだけで結果は変わらない。
var measureCache = map[measureKey][2]int{}

type measureKey struct {
	s    string
	face text.Face
}

const measureCacheMax = 4096

// MeasureText は face で描いたときの s の送り幅と高さを画素で返す。
//
// フォントの送り幅はほぼ整数だが、わずかに端数を持つ。切り捨てると境界をまたぐたびに
// 1px 揺れ、列の幅や中央寄せの位置がずれる。四捨五入で境界から遠ざけ、丸め方を
// この1箇所に固定する。寸法を内容から決めたい箇所はすべてここを通す。
//
// フェイスが nil なら測れないので 0 を返す。呼び出し側は左上寄せへ倒すなどの
// 退避をとる。
func MeasureText(s string, face text.Face) (int, int) {
	if face == nil {
		return 0, 0
	}
	key := measureKey{s: s, face: face}
	textMu.Lock()
	defer textMu.Unlock()
	if wh, ok := measureCache[key]; ok {
		return wh[0], wh[1]
	}
	w, h := text.Measure(s, face, 0)
	wh := [2]int{int(math.Round(w)), int(math.Round(h))}
	if len(measureCache) >= measureCacheMax {
		clear(measureCache)
	}
	measureCache[key] = wh
	return wh[0], wh[1]
}

// MeasureTextWidth は MeasureText の幅だけを返す。列幅や送り幅の算出に使う。
func MeasureTextWidth(s string, face text.Face) int {
	w, _ := MeasureText(s, face)
	return w
}

// FitWidth は中身に合わせた箱の幅を返す。最も広い中身に extra を足し、lower を下回らせない。
//
// extra は中身の外側に足す幅で、左右のパディングなら2つぶん、列と列の間隔ならその1つぶんを渡す。
// lower は箱として痩せすぎる下限で、不要なら 0 を渡す。中身がひとつも無ければ lower になる。
//
// 内容ぴったりの幅を決める計算はこれ1つにする。表の列・選択肢の塊・ボタン・バッジ・
// キーキャップと、同じ規則を要る箇所が散らばっており、別々に書くと下限や余白の扱いがずれる。
func FitWidth(contents []int, extra, lower int) int {
	widest := 0
	for _, c := range contents {
		widest = max(widest, c)
	}
	return max(widest+extra, lower)
}

// LineHeight は face の自然な行送りを画素で返す。アセンダとディセンダを含む字面の
// 高さから取るので、固定値のように字面に対して間延びしない。
func LineHeight(face text.Face) int {
	// アセンダとディセンダを両方持つ字を測り、フェイスの縦の広がりを代表させる
	_, h := MeasureText("Ag", face)
	return h
}
