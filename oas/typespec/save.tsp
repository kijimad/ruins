namespace RuinsEditorApi.SaveData;

// ================== セーブデータ固有のスカラー型 ==================
// ほかの namespace と共通の型は完全修飾名で参照する

/** SHA-256チェックサム */
scalar Checksum extends string;

/** エンティティインデックス */
@minValue(0)
scalar EntityIndex extends uint32;

/** エンティティ世代番号 */
@minValue(0)
scalar EntityGeneration extends uint32;

/** エンティティ名。セーブデータでは空文字を許容する */
scalar EntityName extends string;

/** 説明文。セーブデータでは空文字を許容する */
scalar EntityDescription extends string;

/** プール最大値 (整数) */
scalar PoolMax extends integer;

/** プール現在値 (整数) */
scalar PoolCurrent extends integer;

/** プール最大値 (浮動小数点) */
scalar PoolFloatMax extends float64;

/** プール現在値 (浮動小数点) */
scalar PoolFloatCurrent extends float64;

/** APの毎ターン回復量 */
@minValue(0)
scalar Speed extends integer;

/** 能力値の基本値 */
scalar AbilityBase extends integer;

/** 能力値の装備・状態変化による修正値 */
scalar AbilityModifier extends integer;

/** 能力値の算出された現在値 */
scalar AbilityTotal extends integer;

/** カメラのスケール値 */
scalar CameraScale extends float64;

/** カメラ座標 */
scalar CameraPosition extends float64;

/** タイル座標 */
scalar TileCoord extends integer;

/** スプライトシート名。セーブデータではパターン制約を適用しない */
scalar SpriteSheetName extends string;

/** スプライトキー。セーブデータではパターン制約を適用しない */
scalar SpriteKey extends string;

/** スプライト描画深度 */
scalar SpriteDepth extends integer;

/** 光源の到達半径 (タイル単位)。無効時は0を許容する */
@minValue(0)
scalar LightRadius extends integer;

/** 能力値ボーナス */
scalar AbilityBonus extends integer;

/** 武器種別の識別子 */
scalar AttackTypeId extends string;

/** 攻撃種別の表示用ラベル */
scalar AttackTypeLabel extends string;

/** 現在の装弾数 */
@minValue(0)
scalar Magazine extends integer;

/** 弾薬の口径タグ。セーブデータでは任意文字列を許容する */
scalar AmmoTag extends string;

/** レシピ素材名 */
scalar MaterialName extends string;

/** 固定回復量 */
@minValue(0)
scalar HealNumeral extends integer;

/** ダメージ量 */
@minValue(0)
scalar DamageAmount extends integer;

/** 所持金 */
@minValue(0)
scalar Currency extends integer;

/** 装備スロット番号 */
@minValue(0)
scalar EquipmentSlotNumber extends integer;

// ================== セーブデータ最上位構造 ==================

/** セーブデータの最上位構造。ファイルに保存されるJSON全体を表す */
model SaveData {
  /** セーブデータのバージョン。現在は "1.0.0" のみ */
  version: "1.0.0";

  /** 保存日時 (RFC3339形式) */
  timestamp: utcDateTime;

  /** ワールドデータ */
  world: WorldSaveData;

  /** データ改ざん検知用SHA-256ハッシュ値 */
  checksum: Checksum;
}

/** ワールド全体のセーブデータ。プレイヤーと所持品のみを保存する */
model WorldSaveData {
  /** エンティティの配列。StableIDでソートされる */
  entities: EntitySaveData[];
}

/** 単一エンティティのセーブデータ */
model EntitySaveData {
  /** エンティティの安定ID */
  stable_id: StableID;

  /** コンポーネントのマップ。キーはコンポーネント型名で、値は型ごとに異なる */
  components: ComponentsMap;
}

/** エンティティの安定ID。セーブ/ロード間でエンティティを一意に識別する */
model StableID {
  /** エンティティインデックス */
  index: EntityIndex;

  /** 世代番号。同一インデックスの再利用を区別する */
  generation: EntityGeneration;
}

// ================== コンポーネントマップ ==================

/**
 * エンティティが持つコンポーネントのマップ。
 * 存在するコンポーネントのみがキーとして含まれる。
 * 保存対象: プレイヤー、バックパック内アイテム、装備中アイテム
 */
model ComponentsMap {
  /** プレイヤーマーカー */
  Player?: MarkerComponent;

  /** 味方派閥マーカー */
  FactionAllyData?: MarkerComponent;

  /** バックパック内の位置。所有者エンティティ参照を含む */
  LocationInBackpack?: LocationInBackpackComponent;

  /** ステータス変更フラグ */
  StatsChanged?: MarkerComponent;

  /** 装備位置。エンティティ参照を含む */
  LocationEquipped?: LocationEquippedComponent;

  /** エンティティ名 */
  Name?: NameComponent;

  /** 説明文 */
  Description?: DescriptionComponent;

  /** 生命力 */
  HP?: HPComponent;

  /** 所持重量 */
  WeightCapacity?: WeightCapacityComponent;

  /** アクションポイントと速度 */
  TurnBased?: TurnBasedComponent;

  /** 能力値 */
  Abilities?: AbilitiesComponent;

  /** カメラ状態 */
  Camera?: CameraComponent;

  /** グリッド上の位置 */
  GridElement?: GridElementComponent;

  /** スプライト描画設定 */
  SpriteRender?: SpriteRenderComponent;

  /** 光源設定 */
  LightSource?: LightSourceComponent;

  /** 熱源設定 */
  HeatSource?: HeatSourceComponent;

  /** 防具設定 */
  Wearable?: WearableComponent;

  /** 売買価格 */
  Value?: ValueComponent;

  /** 近接攻撃設定 */
  Melee?: MeleeComponent;

  /** 遠距離攻撃設定 */
  Fire?: FireComponent;

  /** レシピ設定 */
  Recipe?: RecipeComponent;

  /** 弾薬設定 */
  Ammo?: AmmoComponent;

  /** 消費可能アイテム設定 */
  Consumable?: ConsumableComponent;

  /** 回復効果 */
  ProvidesHealing?: ProvidesHealingComponent;

  /** 栄養補給効果 */
  ProvidesNutrition?: ProvidesNutritionComponent;

  /** ダメージ効果 */
  InflictsDamage?: InflictsDamageComponent;

  /** 所持金 */
  Wallet?: WalletComponent;

  /** 戦闘コマンドテーブル名 */
  CommandTable?: CommandTableComponent;

  /** 部位ごとの健康状態 */
  HealthStatus?: HealthStatusComponent;

  /** スキルセット */
  Skills?: SkillsComponent;
}

// ================== マーカーコンポーネント ==================

/** マーカーコンポーネント。データを持たず、存在の有無だけが意味を持つ */
model MarkerComponent {}

// ================== データコンポーネント ==================

/** エンティティ名 */
model NameComponent {
  Name: EntityName;
}

/** 説明文 */
model DescriptionComponent {
  Description: EntityDescription;
}

/** 生命力。最大値と現在値を持つ */
model HPComponent {
  Max: PoolMax;
  Current: PoolCurrent;
}

/** 所持重量。最大値と現在値を持つ */
model WeightCapacityComponent {
  Max: PoolMax;
  Current: PoolCurrent;
}

/** 整数プール。最大値と現在値を持つ */
model IntPool {
  Max: PoolMax;
  Current: PoolCurrent;
}

/** 浮動小数点プール。最大値と現在値を持つ */
model FloatPool {
  Max: PoolFloatMax;
  Current: PoolFloatCurrent;
}

/** アクションポイントと速度 */
model TurnBasedComponent {
  /** アクションポイント */
  AP: IntPool;

  /** APの毎ターン回復量 */
  Speed: Speed;
}

/** 能力値 */
model AbilitiesComponent {
  Vitality: AbilityValue;
  Strength: AbilityValue;
  Sensation: AbilityValue;
  Dexterity: AbilityValue;
  Agility: AbilityValue;
  Defense: AbilityValue;
}

/** 単一能力値。基本値・修正値・合計値を持つ */
model AbilityValue {
  /** 固有の基本値 */
  Base: AbilityBase;

  /** 装備や状態変化による修正値 */
  Modifier: AbilityModifier;

  /** 算出された現在値 */
  Total: AbilityTotal;
}

/** カメラ状態 */
model CameraComponent {
  Scale: CameraScale;
  ScaleTo: CameraScale;
  X: CameraPosition;
  Y: CameraPosition;
  TargetX: CameraPosition;
  TargetY: CameraPosition;
}

/** グリッド上の位置 (タイル座標) */
model GridElementComponent {
  X: TileCoord;
  Y: TileCoord;
}

/** スプライト描画設定 */
model SpriteRenderComponent {
  SpriteSheetName: SpriteSheetName;
  SpriteKey: SpriteKey;
  AnimKeys?: SpriteKey[] | null;
  Depth: SpriteDepth;

  /** Ebitengine DrawImageOptions。レンダラ内部状態 */
  Options?: Record<unknown>;
}

/** 光源設定 */
model LightSourceComponent {
  /** 光の到達半径 (タイル単位) */
  Radius: LightRadius;

  /** RGBA色 */
  Color: RGBAColor;

  /** 有効/無効 */
  Enabled: RuinsEditorApi.LightEnabled;
}

/** 熱源設定 */
model HeatSourceComponent {
  /** 熱の到達半径。チェビシェフ距離、タイル単位 */
  Radius: RuinsEditorApi.HeatRadius;

  /** 毎ターン下げる低体温タイマーの量 */
  Warmth: RuinsEditorApi.HeatWarmth;
}

/** RGBA色。各チャネル0-255 */
model RGBAColor {
  R: RuinsEditorApi.ColorChannel;
  G: RuinsEditorApi.ColorChannel;
  B: RuinsEditorApi.ColorChannel;
  A: RuinsEditorApi.ColorChannel;
}

// ================== アイテム属性コンポーネント ==================

/** 防具設定 */
model WearableComponent {
  /** 防御力 */
  Defense: RuinsEditorApi.ArmorDefense;

  /** 装備部位 */
  EquipmentCategory: RuinsEditorApi.EquipmentCategory;

  /** 装備ボーナス */
  EquipBonus: EquipBonusData;

  /** 耐寒性能 */
  InsulationCold: RuinsEditorApi.InsulationCold;

  /** 耐暑性能 */
  InsulationHeat: RuinsEditorApi.InsulationHeat;
}

/** 装備ボーナス */
model EquipBonusData {
  Vitality: AbilityBonus;
  Strength: AbilityBonus;
  Sensation: AbilityBonus;
  Dexterity: AbilityBonus;
  Agility: AbilityBonus;
}

/** 売買価格 */
model ValueComponent {
  Value: RuinsEditorApi.ItemValue;
}

/** 近接攻撃設定 */
model MeleeComponent {
  Accuracy: RuinsEditorApi.Accuracy;
  Damage: RuinsEditorApi.BaseDamage;
  AttackCount: RuinsEditorApi.AttackCount;
  Element: RuinsEditorApi.Element;
  AttackCategory: AttackCategoryData;
  Cost: RuinsEditorApi.ActionCost;
  TargetType: TargetTypeData;
}

/** 遠距離攻撃設定 */
model FireComponent {
  Accuracy: RuinsEditorApi.Accuracy;
  Damage: RuinsEditorApi.BaseDamage;
  AttackCount: RuinsEditorApi.AttackCount;
  Element: RuinsEditorApi.Element;
  AttackCategory: AttackCategoryData;
  Cost: RuinsEditorApi.ActionCost;
  TargetType: TargetTypeData;

  /** 現在の装弾数 */
  Magazine: Magazine;

  /** 最大装弾数 */
  MagazineSize: RuinsEditorApi.MagazineSize;

  /** リロード完了に必要な総工数 */
  ReloadEffort: RuinsEditorApi.ReloadEffort;

  /** 使用する弾薬の口径タグ */
  AmmoTag: AmmoTag;

  /** 装填中の弾薬によるダメージ修正値 */
  LoadedDamageBonus: RuinsEditorApi.DamageBonus;

  /** 装填中の弾薬による命中修正値 */
  LoadedAccuracyBonus: RuinsEditorApi.AccuracyBonus;
}

/** レシピ設定 */
model RecipeComponent {
  Inputs: RecipeInputData[];
}

/** レシピ素材 */
model RecipeInputData {
  Name: MaterialName;
  Amount: RuinsEditorApi.MaterialAmount;
}

/** 弾薬設定 */
model AmmoComponent {
  AmmoTag: AmmoTag;
  DamageBonus: RuinsEditorApi.DamageBonus;
  AccuracyBonus: RuinsEditorApi.AccuracyBonus;
}

// ================== アイテム効果コンポーネント ==================

/** 消費可能アイテム設定 */
model ConsumableComponent {
  UsableScene: RuinsEditorApi.UsableScene;
  TargetType: TargetTypeData;
}

/**
 * 回復効果。Amounterインターフェースを具体型に分解してシリアライズする
 */
model ProvidesHealingComponent {
  amount: HealingAmountData;
}

/** 回復量データ。typeフィールドで計算方式を判別する */
model HealingAmountData {
  /** 回復量の計算方式 */
  type: HealingAmountType;

  /** 最大HPに対する回復割合 (0.0-1.0)。typeが"ratio"の場合に使用する */
  ratio?: RuinsEditorApi.HealRatio;

  /** 固定回復量。typeが"numeral"の場合に使用する */
  numeral?: HealNumeral;
}

/** 栄養補給効果 */
model ProvidesNutritionComponent {
  Amount: RuinsEditorApi.NutritionAmount;
}

/** ダメージ効果 */
model InflictsDamageComponent {
  Amount: DamageAmount;
}

/** 所持金 */
model WalletComponent {
  Currency: Currency;
}

// ================== エンティティ参照コンポーネント ==================

/**
 * バックパック位置コンポーネント。
 * Ownerフィールドはエンティティ参照のため、StableIDに変換してシリアライズする
 */
model LocationInBackpackComponent {
  /** 所有者のStableID */
  OwnerRef: StableID;
}

/**
 * 装備位置コンポーネント。
 * Ownerフィールドはエンティティ参照のため、StableIDに変換してシリアライズする
 */
model LocationEquippedComponent {
  /** 装備者のStableID */
  OwnerRef: StableID;

  /** 装備スロット番号 */
  EquipmentSlot: EquipmentSlotNumber;
}

// ================== 共通データ型 ==================

/** 対象指定 */
model TargetTypeData {
  TargetGroup: RuinsEditorApi.TargetGroup;
  TargetNum: RuinsEditorApi.TargetNum;
}

/** 攻撃種別 */
model AttackCategoryData {
  /** 武器種別の識別子 */
  Type: AttackTypeId;

  /** 近接/遠距離の区分 */
  Range: AttackRangeType;

  /** 表示用ラベル */
  Label: AttackTypeLabel;
}

// ================== セーブデータ固有のenum ==================

/** 攻撃射程。types.tspに対応するenumがないためセーブ固有で定義する */
enum AttackRangeType {
  MELEE,
  RANGED,
}

/** 回復量の計算方式。types.tspのHealingValueTypeとは値が異なるためセーブ固有で定義する */
enum HealingAmountType {
  /** 最大値に対する割合で回復する */
  ratio,

  /** 固定値で回復する */
  numeral,
}

/** 戦闘コマンドテーブル名 */
model CommandTableComponent {
  /** コマンドテーブル名 */
  Name: string;
}

// ================== 健康状態コンポーネント ==================

/** 部位ごとの健康状態 */
model HealthStatusComponent {
  /** 部位ごとの健康情報の配列 */
  Parts: BodyPartHealth[];

  /** 平熱からの体温のずれ。摂氏。0 が平熱で、寒さで負、暑さで正へ動く */
  BodyTempOffset: float64;
}

/** 1つの部位の健康状態 */
model BodyPartHealth {
  /** この部位に付与されている状態の配列 */
  Conditions?: HealthCondition[];
}

/** 部位に付与される1つの状態 */
model HealthCondition {
  /** 状態の種類 */
  Type: string;

  /** 重症度 */
  Severity: integer;

  /** 進行度タイマー */
  Timer: float64;

  /** この状態によるステータスへの影響 */
  Effects?: StatEffect[];
}

/** ステータスへの影響 */
model StatEffect {
  /** 影響を受けるステータスの種類 */
  Stat: string;

  /** 修正値 */
  Value: integer;
}

// ================== スキルコンポーネント ==================

/** スキルセット */
model SkillsComponent {
  /** スキルIDをキーとしたスキルデータのマップ */
  Data: Record<SkillEntry>;
}

/** 個別スキルの状態 */
model SkillEntry {
  /** スキル値 */
  Value: integer;

  /** 経験値の最大値 */
  ExpMax: integer;

  /** 蓄積経験値 */
  ExpCurrent: integer;
}
