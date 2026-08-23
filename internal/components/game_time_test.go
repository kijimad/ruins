package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestTimeOfDay_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tod  TimeOfDay
		want string
	}{
		{TimeDawn, "Dawn"},
		{TimeMorning, "Morning"},
		{TimeDay, "Noon"},
		{TimeEvening, "Evening"},
		{TimeNight, "Night"},
		{TimeMidnight, "Midnight"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.tod.String())
		})
	}
}

func TestTimeOfDay_String_不正な値はpanicする(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = TimeOfDay(99).String()
	})
}

// TestGameTime_startsAtNoon は新規ゲームの GameTime が経過0・昼・1日目から始まることを確認する。
// TotalTurns は経過ターンで0始まり。昼開始は startOffsetTurns が時刻導出で表す
func TestGameTime_startsAtNoon(t *testing.T) {
	t.Parallel()

	gt := &GameTime{} // 新規ゲームの初期値
	assert.Equal(t, consts.Turn(0), gt.TotalTurns, "経過ターンは0始まり")
	assert.Equal(t, TimeDay, gt.GetTimeOfDay(), "昼から始まる")
	assert.Equal(t, 1, gt.GetDayNumber(), "1日目から始まる")
	assert.Equal(t, 10, gt.GetTemperatureModifier(), "昼の気温修正")
}

// TestGameTime_GetTimeOfDay は経過ターンから時間帯を導けることを確認する。
// 経過0は昼で、そこから夕・夜・深夜・翌朝と進む
func TestGameTime_GetTimeOfDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   TimeOfDay
	}{
		{"経過0は昼", 0, TimeDay},
		{"経過249は昼", 249, TimeDay},
		{"経過250は夕", 250, TimeEvening},
		{"経過499は夕", 499, TimeEvening},
		{"経過500は夜", 500, TimeNight},
		{"経過749は夜", 749, TimeNight},
		{"経過750は深夜", 750, TimeMidnight},
		{"経過999は深夜", 999, TimeMidnight},
		{"経過1000は夜明け（2日目）", 1000, TimeDawn},
		{"経過1249は夜明け", 1249, TimeDawn},
		{"経過1250は朝", 1250, TimeMorning},
		{"経過1499は朝", 1499, TimeMorning},
		{"経過1500は昼（2日目）", 1500, TimeDay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetTimeOfDay())
		})
	}
}

// TestGameTime_GetDayNumber は経過ターンから経過日数を導けることを確認する。
// 日付は暦の夜明けで繰り上がる。昼開始なので翌日の夜明け、経過1000、で2日目に変わる
func TestGameTime_GetDayNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   int
	}{
		{"経過0は1日目", 0, 1},
		{"経過999は1日目", 999, 1},
		{"経過1000は2日目", 1000, 2},
		{"経過2500は3日目", 2500, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetDayNumber())
		})
	}
}

// TestGameTime_GetTemperatureModifier は経過ターンに対応する時間帯の気温修正値を確認する
func TestGameTime_GetTemperatureModifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   int
	}{
		{"昼は+10°C", 0, 10},
		{"夕は+5°C", 250, 5},
		{"夜は-5°C", 500, -5},
		{"深夜は-10°C", 750, -10},
		{"夜明けは+0°C", 1000, 0},
		{"朝は+5°C", 1250, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetTemperatureModifier())
		})
	}
}

// TestGameTime_GetSeasonalTemperature は経過日数に対応する季節の世界温度ベースを確認する。
// 春開始の正弦波で、春秋が中点、夏がピーク、冬が底になる。TotalTurns=(日-1)*1500 で当該日の始まりを指す。
func TestGameTime_GetSeasonalTemperature(t *testing.T) {
	t.Parallel()

	const turnsPerDay consts.Turn = 1500
	tests := []struct {
		name     string
		day      int
		expected int
	}{
		{"1日目は春の中点", 1, -4},
		{"9日目は夏のピーク", 9, 22},
		{"17日目は秋の中点", 17, -4},
		{"25日目は冬の底", 25, -30},
		{"33日目は翌年の春の中点", 33, -4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: consts.Turn(tt.day-1) * turnsPerDay}
			assert.Equal(t, tt.day, gt.GetDayNumber(), "前提: 当該日を指す")
			assert.Equal(t, tt.expected, gt.GetSeasonalTemperature())
		})
	}
}

// TestGameTime_GetSeason は経過日数に対応する季節を確認する。1年を4等分し春開始で巡る。
func TestGameTime_GetSeason(t *testing.T) {
	t.Parallel()

	const turnsPerDay consts.Turn = 1500
	tests := []struct {
		name     string
		day      int
		expected Season
	}{
		{"1日目は春", 1, SeasonSpring},
		{"8日目も春", 8, SeasonSpring},
		{"9日目は夏", 9, SeasonSummer},
		{"17日目は秋", 17, SeasonAutumn},
		{"25日目は冬", 25, SeasonWinter},
		{"33日目は翌年の春", 33, SeasonSpring},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: consts.Turn(tt.day-1) * turnsPerDay}
			assert.Equal(t, tt.day, gt.GetDayNumber(), "前提: 当該日を指す")
			assert.Equal(t, tt.expected, gt.GetSeason())
		})
	}
}

func TestGameTime_Advance(t *testing.T) {
	t.Parallel()

	gt := &GameTime{TotalTurns: 0}
	gt.Advance()
	assert.Equal(t, 1, int(gt.TotalTurns))

	gt.Advance()
	assert.Equal(t, 2, int(gt.TotalTurns))
}

// TestGameTime_AdvanceToNextTimeOfDay は次の時間帯の開始ターンまで進むことを確認する。
// start は経過ターン。区切りは turnsPerTimeOfDay ごとなので、次の区切りへ丸める
func TestGameTime_AdvanceToNextTimeOfDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		start     consts.Turn
		wantTurns consts.Turn
	}{
		{"区切りの途中から次の区切りへ", 100, 250},
		{"区切りちょうどから次の区切りへ", 250, 500},
		{"区切りの途中から次の区切りへ2", 800, 1000},
		{"日をまたぐ区切りへ", 1300, 1500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.start}
			gt.AdvanceToNextTimeOfDay()
			assert.Equal(t, int(tt.wantTurns), int(gt.TotalTurns))
		})
	}
}

// TestGameTime_AdvanceToNextTimeOfDay_常に1つ進む はどの経過ターンから呼んでも
// 時間帯がちょうど1つ進むことを保証する。昼開始のオフセットは一定なので前後の差は変わらない
func TestGameTime_AdvanceToNextTimeOfDay_常に1つ進む(t *testing.T) {
	t.Parallel()

	for start := consts.Turn(0); start < turnsPerDay*2; start += 37 {
		gt := &GameTime{TotalTurns: start}
		before := gt.GetTimeOfDay()
		gt.AdvanceToNextTimeOfDay()
		after := gt.GetTimeOfDay()
		// 時間帯の総数は1日のターン数を時間帯あたりのターン数で割った値
		timeOfDayCount := int(turnsPerDay / turnsPerTimeOfDay)
		want := TimeOfDay((int(before) + 1) % timeOfDayCount)
		assert.Equal(t, want, after, "start=%d では1つ次の時間帯になるべき", start)
	}
}
