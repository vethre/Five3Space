package upsidedown

// Meta-progression system for roguelite mechanics
// Handles permanent upgrades, run modifiers, and character classes

import (
	"encoding/json"
	"main/internal/data"
)

// ========================================
// PERMANENT UPGRADES (Meta-Shop)
// ========================================

type UpgradeType string

const (
	UpgradeMaxHealth     UpgradeType = "max_health"
	UpgradeMaxSanity     UpgradeType = "max_sanity"
	UpgradeStartFlares   UpgradeType = "start_flares"
	UpgradeLightRadius   UpgradeType = "light_radius"
	UpgradeResourceSpawn UpgradeType = "resource_spawn"
	UpgradeMoveSpeed     UpgradeType = "move_speed"
	UpgradeSanityRegen   UpgradeType = "sanity_regen"
	UpgradeDamageResist  UpgradeType = "damage_resist"
)

type Upgrade struct {
	Type        UpgradeType `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	MaxLevel    int         `json:"maxLevel"`
	BaseCost    int         `json:"baseCost"`    // Ember shards per level
	BonusPerLvl float64     `json:"bonusPerLvl"` // Percentage bonus per level
}

var Upgrades = map[UpgradeType]Upgrade{
	UpgradeMaxHealth: {
		Type:        UpgradeMaxHealth,
		Name:        "Hardened Body",
		Description: "+5% max health per level",
		MaxLevel:    10,
		BaseCost:    50,
		BonusPerLvl: 5.0,
	},
	UpgradeMaxSanity: {
		Type:        UpgradeMaxSanity,
		Name:        "Mental Fortitude",
		Description: "+5% max sanity per level",
		MaxLevel:    10,
		BaseCost:    50,
		BonusPerLvl: 5.0,
	},
	UpgradeStartFlares: {
		Type:        UpgradeStartFlares,
		Name:        "Prepared",
		Description: "+1 starting flare per level",
		MaxLevel:    5,
		BaseCost:    100,
		BonusPerLvl: 1.0,
	},
	UpgradeLightRadius: {
		Type:        UpgradeLightRadius,
		Name:        "Inner Light",
		Description: "+10% light radius per level",
		MaxLevel:    5,
		BaseCost:    75,
		BonusPerLvl: 10.0,
	},
	UpgradeResourceSpawn: {
		Type:        UpgradeResourceSpawn,
		Name:        "Scavenger's Luck",
		Description: "+5% resource spawn rate per level",
		MaxLevel:    5,
		BaseCost:    80,
		BonusPerLvl: 5.0,
	},
	UpgradeMoveSpeed: {
		Type:        UpgradeMoveSpeed,
		Name:        "Swift Feet",
		Description: "+3% movement speed per level",
		MaxLevel:    5,
		BaseCost:    60,
		BonusPerLvl: 3.0,
	},
	UpgradeSanityRegen: {
		Type:        UpgradeSanityRegen,
		Name:        "Calm Mind",
		Description: "+10% sanity regen near light per level",
		MaxLevel:    5,
		BaseCost:    70,
		BonusPerLvl: 10.0,
	},
	UpgradeDamageResist: {
		Type:        UpgradeDamageResist,
		Name:        "Thick Skin",
		Description: "+5% damage resistance per level",
		MaxLevel:    5,
		BaseCost:    90,
		BonusPerLvl: 5.0,
	},
}

// ========================================
// RUN MODIFIERS
// ========================================

type ModifierID string

const (
	ModVoidSurge   ModifierID = "void_surge"
	ModDimLight    ModifierID = "dim_light"
	ModQuickDecay  ModifierID = "quick_decay"
	ModHunterMoon  ModifierID = "hunter_moon"
	ModGhostlyMist ModifierID = "ghostly_mist"
	ModBloodMoon   ModifierID = "blood_moon"
)

type RunModifier struct {
	ID              ModifierID `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	EmberMultiplier float64    `json:"emberMultiplier"` // Reward multiplier
	// Effects (applied during run)
	SpawnRateMod    float64 `json:"spawnRateMod"`    // Demogorgon spawn rate multiplier
	LightRestoreMod float64 `json:"lightRestoreMod"` // Light orb effectiveness
	SanityDrainMod  float64 `json:"sanityDrainMod"`  // Sanity drain multiplier
	EnemySightMod   float64 `json:"enemySightMod"`   // Enemy detection range
	EnemySpeedMod   float64 `json:"enemySpeedMod"`   // Enemy movement speed
	ResourceMod     float64 `json:"resourceMod"`     // Resource spawn multiplier
}

var RunModifiers = map[ModifierID]RunModifier{
	ModVoidSurge: {
		ID:              ModVoidSurge,
		Name:            "Void Surge",
		Description:     "The Upside Down hungers. +50% demogorgon spawns, +100% ember shards",
		EmberMultiplier: 2.0,
		SpawnRateMod:    1.5,
		LightRestoreMod: 1.0,
		SanityDrainMod:  1.0,
		EnemySightMod:   1.0,
		EnemySpeedMod:   1.0,
		ResourceMod:     1.0,
	},
	ModDimLight: {
		ID:              ModDimLight,
		Name:            "Dim Light",
		Description:     "Light fades faster, but resources are more common. +50% resources, -50% light restore",
		EmberMultiplier: 1.3,
		SpawnRateMod:    1.0,
		LightRestoreMod: 0.5,
		SanityDrainMod:  1.0,
		EnemySightMod:   1.0,
		EnemySpeedMod:   1.0,
		ResourceMod:     1.5,
	},
	ModQuickDecay: {
		ID:              ModQuickDecay,
		Name:            "Quick Decay",
		Description:     "Your mind slips faster. 2x sanity drain, 1.5x ember shards",
		EmberMultiplier: 1.5,
		SpawnRateMod:    1.0,
		LightRestoreMod: 1.0,
		SanityDrainMod:  2.0,
		EnemySightMod:   1.0,
		EnemySpeedMod:   1.0,
		ResourceMod:     1.0,
	},
	ModHunterMoon: {
		ID:              ModHunterMoon,
		Name:            "Hunter's Moon",
		Description:     "They see further, but move slower. +50% sight, -20% speed",
		EmberMultiplier: 1.4,
		SpawnRateMod:    1.0,
		LightRestoreMod: 1.0,
		SanityDrainMod:  1.0,
		EnemySightMod:   1.5,
		EnemySpeedMod:   0.8,
		ResourceMod:     1.0,
	},
	ModGhostlyMist: {
		ID:              ModGhostlyMist,
		Name:            "Ghostly Mist",
		Description:     "Thick fog obscures all. Reduced visibility for everyone",
		EmberMultiplier: 1.6,
		SpawnRateMod:    1.0,
		LightRestoreMod: 0.7,
		SanityDrainMod:  1.3,
		EnemySightMod:   0.7,
		EnemySpeedMod:   1.0,
		ResourceMod:     1.0,
	},
	ModBloodMoon: {
		ID:              ModBloodMoon,
		Name:            "Blood Moon",
		Description:     "The nightmare realm bleeds through. Everything is harder, rewards are great",
		EmberMultiplier: 3.0,
		SpawnRateMod:    2.0,
		LightRestoreMod: 0.5,
		SanityDrainMod:  1.5,
		EnemySightMod:   1.2,
		EnemySpeedMod:   1.2,
		ResourceMod:     0.7,
	},
}

// ========================================
// CHARACTER CLASSES
// ========================================

type ClassID string

const (
	ClassSurvivor   ClassID = "survivor"
	ClassScout      ClassID = "scout"
	ClassPsychic    ClassID = "psychic"
	ClassPyromancer ClassID = "pyromancer"
)

type CharacterClass struct {
	ID          ClassID `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	UnlockCost  int     `json:"unlockCost"` // Ember shards to unlock (0 = free)
	// Stat modifiers
	HealthMod      float64 `json:"healthMod"`
	SanityMod      float64 `json:"sanityMod"`
	SpeedMod       float64 `json:"speedMod"`
	LightMod       float64 `json:"lightMod"`
	StartingFlares int     `json:"startingFlares"`
	SanityRegenMod float64 `json:"sanityRegenMod"`
	FlareDuration  float64 `json:"flareDuration"` // Multiplier
}

var CharacterClasses = map[ClassID]CharacterClass{
	ClassSurvivor: {
		ID:             ClassSurvivor,
		Name:           "Survivor",
		Description:    "Balanced stats. The default choice for those entering the void",
		UnlockCost:     0,
		HealthMod:      1.0,
		SanityMod:      1.0,
		SpeedMod:       1.0,
		LightMod:       1.0,
		StartingFlares: 0,
		SanityRegenMod: 1.0,
		FlareDuration:  1.0,
	},
	ClassScout: {
		ID:             ClassScout,
		Name:           "Scout",
		Description:    "Lightning fast, but fragile. +30% speed, -15% health",
		UnlockCost:     200,
		HealthMod:      0.85,
		SanityMod:      1.0,
		SpeedMod:       1.3,
		LightMod:       1.0,
		StartingFlares: 0,
		SanityRegenMod: 1.0,
		FlareDuration:  1.0,
	},
	ClassPsychic: {
		ID:             ClassPsychic,
		Name:           "Psychic",
		Description:    "Connected to the light. 2x sanity regen, but enemies sense you better",
		UnlockCost:     300,
		HealthMod:      1.0,
		SanityMod:      1.2,
		SpeedMod:       0.95,
		LightMod:       1.3,
		StartingFlares: 0,
		SanityRegenMod: 2.0,
		FlareDuration:  1.0,
	},
	ClassPyromancer: {
		ID:             ClassPyromancer,
		Name:           "Pyromancer",
		Description:    "Master of fire. Starts with 3 flares, but they burn 50% shorter",
		UnlockCost:     250,
		HealthMod:      0.9,
		SanityMod:      1.0,
		SpeedMod:       1.0,
		LightMod:       1.2,
		StartingFlares: 3,
		SanityRegenMod: 1.0,
		FlareDuration:  0.5,
	},
}

// ========================================
// PLAYER META DATA
// ========================================

type PlayerMeta struct {
	EmberShards     int                 `json:"emberShards"`
	UpgradeLevels   map[UpgradeType]int `json:"upgradeLevels"`
	UnlockedClasses map[ClassID]bool    `json:"unlockedClasses"`
	SelectedClass   ClassID             `json:"selectedClass"`
	TotalRuns       int                 `json:"totalRuns"`
	BestSurvival    float64             `json:"bestSurvival"`
	TotalKills      int                 `json:"totalKills"`
	HighestWave     int                 `json:"highestWave"`
}

func NewPlayerMeta() *PlayerMeta {
	return &PlayerMeta{
		EmberShards:     0,
		UpgradeLevels:   make(map[UpgradeType]int),
		UnlockedClasses: map[ClassID]bool{ClassSurvivor: true},
		SelectedClass:   ClassSurvivor,
		TotalRuns:       0,
		BestSurvival:    0,
		TotalKills:      0,
		HighestWave:     0,
	}
}

// Calculate stat bonuses from upgrades
func (pm *PlayerMeta) GetUpgradeBonus(upgradeType UpgradeType) float64 {
	level := pm.UpgradeLevels[upgradeType]
	upgrade := Upgrades[upgradeType]
	return float64(level) * upgrade.BonusPerLvl / 100.0 // Return as multiplier
}

func (pm *PlayerMeta) GetUpgradeCost(upgradeType UpgradeType) int {
	level := pm.UpgradeLevels[upgradeType]
	upgrade := Upgrades[upgradeType]
	return upgrade.BaseCost * (level + 1)
}

func (pm *PlayerMeta) CanAffordUpgrade(upgradeType UpgradeType) bool {
	upgrade := Upgrades[upgradeType]
	currentLevel := pm.UpgradeLevels[upgradeType]
	if currentLevel >= upgrade.MaxLevel {
		return false
	}
	return pm.EmberShards >= pm.GetUpgradeCost(upgradeType)
}

func (pm *PlayerMeta) PurchaseUpgrade(upgradeType UpgradeType) bool {
	if !pm.CanAffordUpgrade(upgradeType) {
		return false
	}
	cost := pm.GetUpgradeCost(upgradeType)
	pm.EmberShards -= cost
	pm.UpgradeLevels[upgradeType]++
	return true
}

func (pm *PlayerMeta) CanAffordClass(classID ClassID) bool {
	if pm.UnlockedClasses[classID] {
		return false // Already unlocked
	}
	class := CharacterClasses[classID]
	return pm.EmberShards >= class.UnlockCost
}

func (pm *PlayerMeta) PurchaseClass(classID ClassID) bool {
	if !pm.CanAffordClass(classID) {
		return false
	}
	class := CharacterClasses[classID]
	pm.EmberShards -= class.UnlockCost
	pm.UnlockedClasses[classID] = true
	return true
}

// ========================================
// RUN CONFIG (per-run settings)
// ========================================

type RunConfig struct {
	ActiveModifiers []ModifierID `json:"activeModifiers"`
	EndlessMode     bool         `json:"endlessMode"`
	SelectedClass   ClassID      `json:"selectedClass"`
}

func (rc *RunConfig) GetCombinedModifiers() RunModifier {
	combined := RunModifier{
		EmberMultiplier: 1.0,
		SpawnRateMod:    1.0,
		LightRestoreMod: 1.0,
		SanityDrainMod:  1.0,
		EnemySightMod:   1.0,
		EnemySpeedMod:   1.0,
		ResourceMod:     1.0,
	}

	for _, modID := range rc.ActiveModifiers {
		if mod, ok := RunModifiers[modID]; ok {
			combined.EmberMultiplier *= mod.EmberMultiplier
			combined.SpawnRateMod *= mod.SpawnRateMod
			combined.LightRestoreMod *= mod.LightRestoreMod
			combined.SanityDrainMod *= mod.SanityDrainMod
			combined.EnemySightMod *= mod.EnemySightMod
			combined.EnemySpeedMod *= mod.EnemySpeedMod
			combined.ResourceMod *= mod.ResourceMod
		}
	}

	return combined
}

// ========================================
// EMBER SHARD CALCULATION
// ========================================

func CalculateEmberShards(survivalTime float64, score int, kills int, survived bool, modifier float64) int {
	base := int(survivalTime * 2) // 2 shards per second survived
	base += score / 20            // 1 shard per 20 points
	base += kills * 10            // 10 shards per kill

	if survived {
		base *= 2 // Double rewards for survival
	}

	return int(float64(base) * modifier)
}

// ========================================
// SERIALIZE/DESERIALIZE FOR STORAGE
// ========================================

func (pm *PlayerMeta) ToJSON() string {
	data, _ := json.Marshal(pm)
	return string(data)
}

func PlayerMetaFromJSON(jsonStr string) *PlayerMeta {
	if jsonStr == "" {
		return NewPlayerMeta()
	}
	pm := NewPlayerMeta()
	json.Unmarshal([]byte(jsonStr), pm)
	return pm
}

// ========================================
// WAVE SYSTEM (Endless Mode)
// ========================================

type Wave struct {
	Number          int     `json:"number"`
	DemogorgonCount int     `json:"demogorgonCount"`
	SpeedMod        float64 `json:"speedMod"`
	HasBoss         bool    `json:"hasBoss"`
	BossHealth      int     `json:"bossHealth"`
}

func GenerateWave(waveNum int) Wave {
	wave := Wave{
		Number:          waveNum,
		DemogorgonCount: 2 + waveNum,                   // Escalating count
		SpeedMod:        1.0 + float64(waveNum)*0.05,   // 5% faster each wave
		HasBoss:         waveNum%5 == 0 && waveNum > 0, // Boss every 5 waves
		BossHealth:      100 + (waveNum/5)*50,          // Boss gets tankier
	}
	return wave
}

// Store integration helpers
func LoadPlayerMeta(store *data.Store, userID string) *PlayerMeta {
	if userID == "" || userID == "guest" {
		return NewPlayerMeta()
	}
	user, ok := store.GetUser(userID)
	if !ok {
		return NewPlayerMeta()
	}
	// Store meta in user's extra data field (we'll add this)
	return PlayerMetaFromJSON(user.UpsideDownMeta)
}

func SavePlayerMeta(store *data.Store, userID string, meta *PlayerMeta) {
	if userID == "" || userID == "guest" {
		return
	}
	store.UpdateUpsideDownMeta(userID, meta.ToJSON())
}
