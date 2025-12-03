package chibiki

// UnitType is a helper, but we will treat Target as string for simplicity in JSON
type UnitType string

const (
	TypeGround   UnitType = "ground"
	TypeFlying   UnitType = "flying"
	TypeBuilding UnitType = "building"
	TypeSpell    UnitType = "spell"
)

type UnitStats struct {
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	Elixir   int     `json:"elixir"`
	HP       float64 `json:"hp"`
	Damage   float64 `json:"damage"`
	HitSpeed float64 `json:"hit_speed"`
	Speed    float64 `json:"speed"`
	Range    float64 `json:"range"`
	Target   string  `json:"target_type"` // e.g. "ground", "all"
	Ability  string  `json:"ability"`
}

type Entity struct {
	ID           string    `json:"id"`
	Key          string    `json:"key"`
	OwnerID      string    `json:"owner"`
	Team         int       `json:"team"` // 0 = Bottom (Player), 1 = Top (Enemy)
	X            float64   `json:"x"`
	Y            float64   `json:"y"`
	HP           float64   `json:"hp"`
	MaxHP        float64   `json:"max_hp"`
	Stats        UnitStats `json:"-"` // Don't send static stats every frame
	LastAttack   float64   `json:"-"`
	TargetID     string    `json:"-"`
	StunnedUntil float64   `json:"-"`
}
