package genspec

// Def はコンポーネント1種の登録エントリを表す。
type Def struct {
	// Field は EntitySpec / Components 構造体のフィールド名。型名と一致させる。
	Field string
	// Comment は EntitySpec / Components フィールドに付与する行末コメント。全エントリに付ける。
	Comment string
}

// Registry は全コンポーネントの登録表。出力順もこの順序に従う。
var Registry = []Def{
	// general ================
	{Field: "Name", Comment: "holds the display name"},
	{Field: "Description", Comment: "holds the description text"},

	// item ================
	{Field: "HP", Comment: "represents life force; death when it runs out"},
	{Field: "Consumable", Comment: "represents a consumable used up on use"},
	{Field: "WeightCapacity", Comment: "represents carry and storage weight capacity"},
	{Field: "Melee", Comment: "holds melee attack stats"},
	{Field: "Fire", Comment: "holds ranged attack stats and ammo"},
	{Field: "Value", Comment: "represents the base value of an item"},
	{Field: "Weight", Comment: "represents the weight of an item"},
	{Field: "Recipe", Comment: "holds the materials required for crafting"},
	{Field: "Wearable", Comment: "holds equipment stats"},
	{Field: "Abilities", Comment: "holds the entity's ability scores"},
	{Field: "Ammo", Comment: "holds ammo item stats"},
	{Field: "Stackable", Comment: "represents a stackable item with a count"},
	{Field: "Material", Comment: "marks a material for crafting or selling"},
	{Field: "LocationInBackpack", Comment: "marks being in the backpack"},
	{Field: "LocationEquipped", Comment: "marks being equipped"},
	{Field: "LocationOnField", Comment: "marks being on the field"},
	{Field: "LocationInStorage", Comment: "marks being in a container"},

	// field ================
	{Field: "Tile", Comment: "marks a tile entity"},
	{Field: "SoloAI", Comment: "holds solo AI settings"},
	{Field: "SquadAI", Comment: "holds squad member AI settings"},
	{Field: "Camera", Comment: "holds camera position and zoom"},
	{Field: "Position", Comment: "holds pixel coordinates on the field"},
	{Field: "GridElement", Comment: "holds grid coordinates on the field"},
	{Field: "SpriteRender", Comment: "holds sprite rendering info"},
	{Field: "BlockView", Comment: "marks blocking of vision"},
	{Field: "BlockPass", Comment: "marks being impassable"},
	{Field: "PassCost", Comment: "holds a tile's movement cost modifier"},
	{Field: "Door", Comment: "represents an openable door"},
	{Field: "Fixed", Comment: "marks a fixed object anchored in the world that cannot be picked up"},
	{Field: "Pushable", Comment: "marks being pushable; the base cube is the first user but the marker is generic"},
	{Field: "LightSource", Comment: "represents a light source"},
	{Field: "Interactable", Comment: "marks being interactable"},
	{Field: "VisualEffects", Comment: "manages the associated visual effect"},
	{Field: "TileTemperature", Comment: "holds a tile's temperature modifier"},

	// stage ================
	{Field: "StageBound", Comment: "holds the bound stage; used to identify stages you travel between"},
	{Field: "StageField", Comment: "holds per-stage field state; the current stage is looked up via CurrentStage"},
	{Field: "SeamlessBand", Comment: "holds the persistent overworld band and front state; its presence also marks the overworld"},
	{Field: "PortalConnection", Comment: "holds a portal's destination stage and landing coordinates"},
	{Field: "DungeonEntrance", Comment: "holds the ruin definition name a ruin entrance leads to"},
	{Field: "Suspended", Comment: "marks belonging to a non-current stage and being inactive"},

	// member ================
	{Field: "Player", Comment: "marks the player-controlled protagonist"},
	{Field: "Profession", Comment: "holds the chosen profession"},
	{Field: "Hunger", Comment: "holds the player's hunger"},
	{Field: "Wallet", Comment: "holds the player's funds"},
	{Field: "FactionAlly", Comment: "marks the ally faction"},
	{Field: "FactionEnemy", Comment: "marks the enemy faction"},
	{Field: "FactionNeutral", Comment: "marks the neutral faction"},
	{Field: "Boss", Comment: "marks a boss entity"},
	{Field: "Dialog", Comment: "holds dialogue data"},
	{Field: "Dead", Comment: "marks a dead state"},
	{Field: "TurnBased", Comment: "manages action points"},
	{Field: "HealthStatus", Comment: "holds per-body-part health status"},
	{Field: "Skills", Comment: "holds the skill set"},
	{Field: "CharModifiers", Comment: "aggregates effect multipliers"},

	// event ================
	{Field: "StateChangeRequest", Comment: "carries a state transition request"},
	{Field: "StatsChanged", Comment: "dirty flag marking that stats need recalculation"},
	{Field: "WeightDirty", Comment: "dirty flag marking that weight needs recalculation"},
	{Field: "ProvidesHealing", Comment: "holds HP healing properties"},
	{Field: "ProvidesNutrition", Comment: "holds hunger recovery properties"},
	{Field: "InflictsDamage", Comment: "holds damage-dealing properties"},

	// book ================
	{Field: "Book", Comment: "represents a readable book"},

	// battle ================
	{Field: "CommandTable", Comment: "holds the combat command table name for AI"},
	{Field: "DropTable", Comment: "holds the drop table name"},

	// squad ================
	{Field: "SquadMember", Comment: "marks a squad member"},

	// activity ================
	{Field: "Activity", Comment: "holds the running activity"},
	{Field: "LastActivity", Comment: "holds the latest activity result"},

	// singleton ================
	{Field: "GameLog", Comment: "singleton holding the game log storage"},
	{Field: "Dungeon", Comment: "singleton holding dungeon state"},
	{Field: "GameProgress", Comment: "singleton holding game progress data"},
	{Field: "TurnState", Comment: "singleton holding turn state"},
	{Field: "SpatialIndex", Comment: "singleton holding the spatial index"},
	{Field: "WeaponSelection", Comment: "singleton holding the selected weapon slot"},
	{Field: "GameTime", Comment: "singleton holding in-game time"},
	{Field: "VisionState", Comment: "singleton holding temporary vision-calculation state"},
	{Field: "UserSettings", Comment: "singleton holding global settings changed on the settings screen"},
}
