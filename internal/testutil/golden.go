package testutil

import (
	"bytes"
	"image/png"
)

// PNGPixelEqual は2つの PNG バイト列を復号し、全画素が完全一致するかを返す。goldie.WithEqualFn へ
// 渡して、CPU 描画の golden をバイトでなく画素で比較するのに使う。png.Encode の圧縮は Go の
// バージョンで出力バイトが変わりうるが、復号後の画素は変わらない。バイト比較だと toolchain 更新の
// たびに全 golden が割れるのを、画素比較にして防ぐ。VRT の許容付き比較と違い、CPU 描画は決定的な
// ので許容ゼロの厳密一致にし、実際の描画変化は必ず捉える。
func PNGPixelEqual(actual, expected []byte) bool {
	a, err := png.Decode(bytes.NewReader(actual))
	if err != nil {
		return false
	}
	e, err := png.Decode(bytes.NewReader(expected))
	if err != nil {
		return false
	}
	ab, eb := a.Bounds(), e.Bounds()
	if ab != eb {
		return false
	}
	for y := ab.Min.Y; y < ab.Max.Y; y++ {
		for x := ab.Min.X; x < ab.Max.X; x++ {
			ar, ag, abl, aa := a.At(x, y).RGBA()
			er, eg, ebl, ea := e.At(x, y).RGBA()
			if ar != er || ag != eg || abl != ebl || aa != ea {
				return false
			}
		}
	}
	return true
}
