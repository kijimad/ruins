package components

import "github.com/kijimaD/ruins/internal/oapi"

// testNeedDefs は components のテストが使う空腹と疲労の段階の表。値は raw の行と同じにする
var testNeedDefs = NeedDefs{
	Hunger: []HungerStage{
		{Level: oapi.HungerStageIdSatiated, Name: "Full", From: 0.95},
		{Level: oapi.HungerStageIdNormal, Name: "Normal", From: 0.66},
		{Level: oapi.HungerStageIdHungry, Name: "Hungry", From: 0.33, Condition: SeverityMinor},
		{Level: oapi.HungerStageIdStarving, Name: "Starving", From: 0, Condition: SeverityMedium},
	},
	Fatigue: []FatigueStage{
		{Level: oapi.FatigueStageIdRested, Name: "Rested", From: 0},
		{Level: oapi.FatigueStageIdNormal, Name: "Normal", From: 0.3},
		{Level: oapi.FatigueStageIdTired, Name: "Tired", From: 0.5, Condition: SeverityMinor},
		{Level: oapi.FatigueStageIdExhausted, Name: "Exhausted", From: 0.8, Condition: SeverityMedium},
	},
}
