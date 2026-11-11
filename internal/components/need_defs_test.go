package components

import "github.com/kijimaD/ruins/internal/rawschema"

// testNeedDefs は components のテストが使う空腹と疲労の段階の表。値は raw の行と同じにする
var testNeedDefs = NeedDefs{
	Hunger: []HungerStage{
		{Level: rawschema.HungerStageIdSatiated, Name: "Full", From: 0.95},
		{Level: rawschema.HungerStageIdNormal, Name: "Normal", From: 0.66},
		{Level: rawschema.HungerStageIdHungry, Name: "Hungry", From: 0.33, Condition: SeverityMinor},
		{Level: rawschema.HungerStageIdStarving, Name: "Starving", From: 0, Condition: SeverityMedium},
	},
	Fatigue: []FatigueStage{
		{Level: rawschema.FatigueStageIdRested, Name: "Rested", From: 0},
		{Level: rawschema.FatigueStageIdNormal, Name: "Normal", From: 0.3},
		{Level: rawschema.FatigueStageIdTired, Name: "Tired", From: 0.5, Condition: SeverityMinor},
		{Level: rawschema.FatigueStageIdExhausted, Name: "Exhausted", From: 0.8, Condition: SeverityMedium},
	},
}
