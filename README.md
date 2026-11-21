![coverage](https://kijimad.github.io/ruins/cov/coverage.svg)

![Ruins](docs/steam/generated/library_header.png)

A turn-based survival roguelike. Your ship crashed on a frozen planet, and only your cargo cube survived. Head north with the cube, scavenge ruins to fend off cold and hunger, and repair the escape core to leave the planet.

- [Steam](https://store.steampowered.com/app/4791810/Ruins/)
- [Play in the browser](https://kijimad.github.io/ruins/)
- [Test report](https://kijimad.github.io/ruins/cov/)
- [Raw data schema](internal/rawschema/schema.cue)
- [Godoc](https://pkg.go.dev/github.com/kijimaD/ruins)

## Screenshots

| | | | |
|---|---|---|---|
| <img src="internal/states/testdata/TestGolden_CharacterJob.png" width="200" /><br>CharacterJob | <img src="internal/states/testdata/TestGolden_CharacterNaming.png" width="200" /><br>CharacterNaming | <img src="internal/states/testdata/TestGolden_Character_Abilities.png" width="200" /><br>Character_Abilities | <img src="internal/states/testdata/TestGolden_Character_Basic.png" width="200" /><br>Character_Basic |
| <img src="internal/states/testdata/TestGolden_Character_Effects.png" width="200" /><br>Character_Effects | <img src="internal/states/testdata/TestGolden_Character_Equipment.png" width="200" /><br>Character_Equipment | <img src="internal/states/testdata/TestGolden_Character_Health.png" width="200" /><br>Character_Health | <img src="internal/states/testdata/TestGolden_Character_Skills.png" width="200" /><br>Character_Skills |
| <img src="internal/states/testdata/TestGolden_ChoiceMenuHeaders.png" width="200" /><br>ChoiceMenuHeaders | <img src="internal/states/testdata/TestGolden_ChoiceMenuMany.png" width="200" /><br>ChoiceMenuMany | <img src="internal/states/testdata/TestGolden_ComponentDebug.png" width="200" /><br>ComponentDebug | <img src="internal/states/testdata/TestGolden_CraftMenu_Armor.png" width="200" /><br>CraftMenu_Armor |
| <img src="internal/states/testdata/TestGolden_CraftMenu_Consumables.png" width="200" /><br>CraftMenu_Consumables | <img src="internal/states/testdata/TestGolden_CraftMenu_Weapons.png" width="200" /><br>CraftMenu_Weapons | <img src="internal/states/testdata/TestGolden_DebugMenu.png" width="200" /><br>DebugMenu | <img src="internal/states/testdata/TestGolden_Dungeon.png" width="200" /><br>Dungeon |
| <img src="internal/states/testdata/TestGolden_DungeonMenu.png" width="200" /><br>DungeonMenu | <img src="internal/states/testdata/TestGolden_EquipCompare_Detail.png" width="200" /><br>EquipCompare_Detail | <img src="internal/states/testdata/TestGolden_EquipCompare_List.png" width="200" /><br>EquipCompare_List | <img src="internal/states/testdata/TestGolden_EquipSelect.png" width="200" /><br>EquipSelect |
| <img src="internal/states/testdata/TestGolden_EquipSelectFlow.png" width="200" /><br>EquipSelectFlow | <img src="internal/states/testdata/TestGolden_FeedFuelMenu.png" width="200" /><br>FeedFuelMenu | <img src="internal/states/testdata/TestGolden_GameOver.png" width="200" /><br>GameOver | <img src="internal/states/testdata/TestGolden_ItemActionDetail.png" width="200" /><br>ItemActionDetail |
| <img src="internal/states/testdata/TestGolden_ItemAction_Drop.png" width="200" /><br>ItemAction_Drop | <img src="internal/states/testdata/TestGolden_ItemAction_Eat.png" width="200" /><br>ItemAction_Eat | <img src="internal/states/testdata/TestGolden_ItemAction_Inspect.png" width="200" /><br>ItemAction_Inspect | <img src="internal/states/testdata/TestGolden_ItemAction_Read.png" width="200" /><br>ItemAction_Read |
| <img src="internal/states/testdata/TestGolden_ItemAction_Use.png" width="200" /><br>ItemAction_Use | <img src="internal/states/testdata/TestGolden_ItemAction_VerbJump.png" width="200" /><br>ItemAction_VerbJump | <img src="internal/states/testdata/TestGolden_ItemMarker.png" width="200" /><br>ItemMarker | <img src="internal/states/testdata/TestGolden_KeyHelp.png" width="200" /><br>KeyHelp |
| <img src="internal/states/testdata/TestGolden_LoadMenu.png" width="200" /><br>LoadMenu | <img src="internal/states/testdata/TestGolden_LookAround.png" width="200" /><br>LookAround | <img src="internal/states/testdata/TestGolden_LookAround_Away.png" width="200" /><br>LookAround_Away | <img src="internal/states/testdata/TestGolden_MainMenu.png" width="200" /><br>MainMenu |
| <img src="internal/states/testdata/TestGolden_MainMenuOpenSettings.png" width="200" /><br>MainMenuOpenSettings | <img src="internal/states/testdata/TestGolden_MainMenu_LoadFailed.png" width="200" /><br>MainMenu_LoadFailed | <img src="internal/states/testdata/TestGolden_Message.png" width="200" /><br>Message | <img src="internal/states/testdata/TestGolden_Overworld.png" width="200" /><br>Overworld |
| <img src="internal/states/testdata/TestGolden_OverworldMap.png" width="200" /><br>OverworldMap | <img src="internal/states/testdata/TestGolden_Overworld_1_Morning.png" width="200" /><br>Overworld_1_Morning | <img src="internal/states/testdata/TestGolden_Overworld_2_Day.png" width="200" /><br>Overworld_2_Day | <img src="internal/states/testdata/TestGolden_Overworld_3_Evening.png" width="200" /><br>Overworld_3_Evening |
| <img src="internal/states/testdata/TestGolden_Overworld_4_Night.png" width="200" /><br>Overworld_4_Night | <img src="internal/states/testdata/TestGolden_Overworld_5_Midnight.png" width="200" /><br>Overworld_5_Midnight | <img src="internal/states/testdata/TestGolden_Overworld_6_Dawn.png" width="200" /><br>Overworld_6_Dawn | <img src="internal/states/testdata/TestGolden_Overworld_Indoor.png" width="200" /><br>Overworld_Indoor |
| <img src="internal/states/testdata/TestGolden_Overworld_Snowfall.png" width="200" /><br>Overworld_Snowfall | <img src="internal/states/testdata/TestGolden_Overworld_Winter.png" width="200" /><br>Overworld_Winter | <img src="internal/states/testdata/TestGolden_Overworld_Winter_Remains.png" width="200" /><br>Overworld_Winter_Remains | <img src="internal/states/testdata/TestGolden_PersistentMessage.png" width="200" /><br>PersistentMessage |
| <img src="internal/states/testdata/TestGolden_PickupMenuStorage.png" width="200" /><br>PickupMenuStorage | <img src="internal/states/testdata/TestGolden_Remains.png" width="200" /><br>Remains | <img src="internal/states/testdata/TestGolden_Remains_Fadeout.png" width="200" /><br>Remains_Fadeout | <img src="internal/states/testdata/TestGolden_RunClear.png" width="200" /><br>RunClear |
| <img src="internal/states/testdata/TestGolden_RunStats_Environment.png" width="200" /><br>RunStats_Environment | <img src="internal/states/testdata/TestGolden_RunStats_Table.png" width="200" /><br>RunStats_Table | <img src="internal/states/testdata/TestGolden_SaveMenu.png" width="200" /><br>SaveMenu | <img src="internal/states/testdata/TestGolden_SettingsMenu.png" width="200" /><br>SettingsMenu |
| <img src="internal/states/testdata/TestGolden_SettingsMenuClose.png" width="200" /><br>SettingsMenuClose | <img src="internal/states/testdata/TestGolden_Shooting.png" width="200" /><br>Shooting | <img src="internal/states/testdata/TestGolden_ShopMenu_Buy.png" width="200" /><br>ShopMenu_Buy | <img src="internal/states/testdata/TestGolden_ShopMenu_Sell.png" width="200" /><br>ShopMenu_Sell |
| <img src="internal/states/testdata/TestGolden_StatusBadges.png" width="200" /><br>StatusBadges | <img src="internal/states/testdata/TestGolden_StorageMenu_Retrieve.png" width="200" /><br>StorageMenu_Retrieve | <img src="internal/states/testdata/TestGolden_StorageMenu_Store.png" width="200" /><br>StorageMenu_Store | |


Every image is generated by the golden tests.

## Controls

### Exploration

**Movement**

| Key | Action |
| --- | --- |
| ↑ / ↓ / ← / → | Move. Movement is in 4 directions; rotate the view to walk diagonally |
| . (period) | Wait a turn |

**Actions**

| Key | Action |
| --- | --- |
| Enter | Interact with what is underfoot or next to you |
| Space | Open the interaction menu |
| G | Pick up items |
| S | Attack in a direction |
| F | Shooting mode. Tab / Shift + Tab switch targets, Enter fires, R reloads |
| 1-5 | Switch hand slot |
| B | Sleep |

**Item verbs**. Each opens the item screen for that verb.

| Key | Action |
| --- | --- |
| Shift + X | Inspect |
| D | Drop |
| Shift + D | Disassemble |
| E | Eat or drink |
| R | Read |
| T | Use |

**Screens**

| Key | Action |
| --- | --- |
| M | Menu |
| L | Field info |
| N | Overworld map, only on the overworld |
| ? | Key help |

**Camera**

| Key | Action |
| --- | --- |
| Z / C | Rotate the view 45 degrees left / right |
| PageDown / PageUp | Zoom out / in |
| Mouse wheel | Zoom |

### Menus

| Key | Action |
| --- | --- |
| ↑ / ↓ | Move between items |
| ← / → | Switch tabs, or move left and right in a grid |
| Tab / Shift + Tab | Next / previous tab |
| X | Item details, in item lists |
| Enter | Select |
| Escape | Cancel or go back |
| ? | Key help |

## Development

See the Dockerfile for dependencies.

```
$ make help
```

## Design documents

Generated from the frontmatter in `docs/design`. Filter them with `go run . designdoc list`.

| status | Document | Progress | tags |
|---|---|---|---|
| draft | [灯りと武器と道具に残量を持たせて有限にする](docs/design/261119224218.md) | 2/17 | item, gamedesign |


## References

Code and games this project drew on.

- https://github.com/x-hgg-x/sokoban-go
  - The project started as a copy of this and was reshaped from there
  - Reference for using ECS
- https://github.com/x-hgg-x/goecsengine
  - Reference for game states
- https://bfnihtly.bracketproductions.com/
  - Reference for data-driven factories and the game log
- https://krkrz.github.io/krkr2doc/kag3doc/contents/
  - Reference for the markup a sound novel needs
- https://ebitengine.org/en/examples/raycasting.html
  - Reference for raycasting
- https://cataclysmdda.org/
  - Reference for roguelike systems
- KONAMI's Power Pro Kun Pocket 10, 11 and 12, for game systems
  - Little of it remains after a change of direction

Assets.

Fonts.

- http://jikasei.me/font/jf-dotfont/
- https://github.com/googlefonts/morisawa-biz-ud-gothic

Images.

- https://www.pixilart.com
- https://pixabay.com/photos/forest-fog-woods-trees-mystical-3394066/
- https://pixabay.com/photos/beer-drink-alcohol-heineken-bar-5940890/
- https://pixabay.com/photos/lost-places-monastery-past-masonry-4019367/
