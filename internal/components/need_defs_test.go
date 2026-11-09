package components

// testNeedDefs は components のテストが使う空腹と疲労の段階の表。値は raw の行と同じにする
var testNeedDefs = NewNeedDefs(
	[]HungerStage{
		{Level: HungerSatiated, From: 0.95},
		{Level: HungerNormal, From: 0.66},
		{Level: HungerHungry, From: 0.33, Condition: SeverityMinor},
		{Level: HungerStarving, From: 0, Condition: SeverityMedium},
	},
	[]FatigueStage{
		{Level: FatigueRested, From: 0},
		{Level: FatigueNormal, From: 0.3},
		{Level: FatigueTired, From: 0.5, Condition: SeverityMinor},
		{Level: FatigueExhausted, From: 0.8, Condition: SeverityMedium},
	},
)
