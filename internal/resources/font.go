package resources

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/assets"
)

// Font structure
type Font struct {
	Font       text.Face
	FaceSource *text.GoTextFaceSource // コピーが禁止されていて参照渡ししかできない
}

// UnmarshalTOML fills structure fields from TOML data
func (f *Font) UnmarshalTOML(i any) error {
	m, ok := i.(map[string]any)
	if !ok {
		return fmt.Errorf("fontのデコードに失敗: mapではありません: %T", i)
	}
	fontPath, ok := m["font"].(string)
	if !ok {
		return fmt.Errorf("fontのデコードに失敗: fontフィールドが文字列ではありません")
	}
	fontFile, err := assets.FS.Open(fontPath)
	if err != nil {
		return err
	}

	s, err := text.NewGoTextFaceSource(fontFile)
	if err != nil {
		return err
	}
	f.FaceSource = s

	font := &text.GoTextFace{
		Source: s,
		Size:   24,
	}
	f.Font = font

	return nil
}
