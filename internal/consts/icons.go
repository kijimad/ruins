package consts

// Nerd Fonts アイコン定数
// https://www.nerdfonts.com/cheat-sheet
const (
	// 矢印
	IconArrowLeft  = "\uf060"
	IconArrowRight = "\uf061"
	IconArrowUp    = "\uf062"
	IconArrowDown  = "\uf063"

	// カーソル
	IconCursor = "\uf054" // fa-chevron-right

	// 操作
	IconEnter    = "\uf148"
	IconCheck    = "\uf00c"
	IconClose    = "\uf00d"
	IconSave     = "\uf0c7"
	IconKeyEnter = "\U000f0311" // md-keyboard_return
	IconKeyEsc   = "\U000f12b7" // md-keyboard_esc
	IconKeySpace = "\U000f1050" // md-keyboard_space
	IconKeyTab   = "\U000f0312" // md-keyboard_tab
	IconKeyShift = "\U000f0636" // md-apple_keyboard_shift
	IconKeyHelp  = "\U000f078b" // md-help_box。? キーのキーキャップ表記
	IconKeyDot   = "\U000f09df" // md-circle_small。. キーのキーキャップ表記

	// UI
	IconHome     = "\uf015"
	IconSettings = "\uf013"
	IconMenu     = "\uf0c9"
	IconSearch   = "\uf002"

	// 単位
	IconDegree   = "\u2103"     // 摂氏記号 ℃
	IconCurrency = "\U000f1aaf" // 通貨記号 nf-md-heating_coil
	IconKg       = "\u338f"     // ㎏
	IconG        = "g"          // グラム。単一の合字がないため素の g を使う
	IconMg       = "\u338e"     // ㎎

	// ゲーム
	IconHeart    = "\uf004"
	IconStar     = "\uf005"
	IconShield   = "\uf132"
	IconSword    = "\uf71c"
	IconSkull    = "\uf54c"
	IconFlask    = "\uf0c3"
	IconFire     = "\uf06d"
	IconBolt     = "\uf0e7"
	IconLeaf     = "\uf06c"
	IconDroplet  = "\uf043"
	IconMoon     = "\uf186"
	IconSun      = "\uf185"
	IconUser     = "\uf007"
	IconUsers    = "\uf0c0"
	IconBag      = "\uf290"
	IconMap      = "\uf279"
	IconCompass  = "\uf14e"
	IconFlag     = "\uf024"
	IconClock    = "\uf017"
	IconWarning  = "\uf071"
	IconInfo     = "\uf129"
	IconQuestion = "\uf128"
	IconCube     = "\uf1b2" // fa-cube
)

// IconKeyAlphaBoxBase は md-alpha_a_box。英字キーキャップの先頭で、a からの差分を足す。
// a から z まで連続配置であることは実グリフのコードポイントで確認済み
const IconKeyAlphaBoxBase rune = 0xF0B08

// IconKeyDigitBoxes は数字キーのキーキャップグリフ md-numeric_N_box。
// 系列のコードポイントは等差でなく5の前後で刻みが乱れるため、算術でなく実測の表で持つ。
// 等差を仮定すると 5 だけ outline 系の別グリフを拾い、白抜きの箱が1つ混ざる
var IconKeyDigitBoxes = [10]string{
	"\U000f03a1", // 0
	"\U000f03a4", // 1
	"\U000f03a7", // 2
	"\U000f03aa", // 3
	"\U000f03ad", // 4
	"\U000f03b1", // 5
	"\U000f03b3", // 6
	"\U000f03b6", // 7
	"\U000f03b9", // 8
	"\U000f03bc", // 9
}
