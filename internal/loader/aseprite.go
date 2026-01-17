package loader

// AsepriteJSON は Aseprite が生成する JSON スプライトシートフォーマット
type AsepriteJSON struct {
	Frames []AsepriteFrame `json:"frames"`
	Meta   AsepriteMeta    `json:"meta"`
}

// AsepriteFrame は1つのスプライトフレーム情報
type AsepriteFrame struct {
	Filename         string       `json:"filename"`
	Frame            AsepriteRect `json:"frame"`
	Rotated          bool         `json:"rotated"`
	Trimmed          bool         `json:"trimmed"`
	SpriteSourceSize AsepriteRect `json:"spriteSourceSize"`
	SourceSize       AsepriteSize `json:"sourceSize"`
	Duration         int          `json:"duration"`
}

// AsepriteRect は矩形の位置とサイズ
type AsepriteRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// AsepriteSize はサイズ
type AsepriteSize struct {
	W int `json:"w"`
	H int `json:"h"`
}

// AsepriteMeta はメタデータ
type AsepriteMeta struct {
	App     string       `json:"app"`
	Version string       `json:"version"`
	Image   string       `json:"image"`
	Format  string       `json:"format"`
	Size    AsepriteSize `json:"size"`
	Scale   string       `json:"scale"`
}
