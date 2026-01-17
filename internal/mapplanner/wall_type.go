package mapplanner

// WallType は壁の種類を表す
type WallType int

const (
	// WallTypeTop は上側の壁（下に床がある壁）
	WallTypeTop WallType = iota
	// WallTypeBottom は下側の壁（上に床がある壁）
	WallTypeBottom
	// WallTypeLeft は左側の壁（右に床がある壁）
	WallTypeLeft
	// WallTypeRight は右側の壁（左に床がある壁）
	WallTypeRight
	// WallTypeTopLeft は左上角の壁（右下に床がある壁）
	WallTypeTopLeft
	// WallTypeTopRight は右上角の壁（左下に床がある壁）
	WallTypeTopRight
	// WallTypeBottomLeft は左下角の壁（右上に床がある壁）
	WallTypeBottomLeft
	// WallTypeBottomRight は右下角の壁（左上に床がある壁）
	WallTypeBottomRight
	// WallTypeGeneric は汎用の壁（複雑なパターンまたは判定不可）
	WallTypeGeneric
)

// String は壁タイプを文字列で返す
func (wt WallType) String() string {
	switch wt {
	case WallTypeTop:
		return "Top"
	case WallTypeBottom:
		return "Bottom"
	case WallTypeLeft:
		return "Left"
	case WallTypeRight:
		return "Right"
	case WallTypeTopLeft:
		return "TopLeft"
	case WallTypeTopRight:
		return "TopRight"
	case WallTypeBottomLeft:
		return "BottomLeft"
	case WallTypeBottomRight:
		return "BottomRight"
	case WallTypeGeneric:
		return "Generic"
	default:
		return "Unknown"
	}
}
