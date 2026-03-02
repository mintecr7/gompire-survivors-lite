package world

type WeaponKind int

const (
	WeaponWhip WeaponKind = iota
	WeaponSpear
	WeaponNova
	WeaponFang
)

type WeaponDef struct {
	Name         string
	AttackStyle  WeaponAttackStyle
	DamageMul    float32
	CooldownMul  float32
	RangeMul     float32
	DropWeight   int
	DropRadius   float32
	AttackRadius float32 // used by radial attack styles
}

type WeaponAttackStyle int

const (
	AttackSingle WeaponAttackStyle = iota
	AttackPierce
	AttackRadial
)

var weaponDefs = map[WeaponKind]WeaponDef{
	WeaponWhip: {
		Name:        "Whip",
		AttackStyle: AttackSingle,
		DamageMul:   1.00,
		CooldownMul: 1.00,
		RangeMul:    1.00,
		DropWeight:  30,
		DropRadius:  9,
	},
	WeaponSpear: {
		Name:        "Spear",
		AttackStyle: AttackPierce,
		DamageMul:   0.85,
		CooldownMul: 0.85,
		RangeMul:    1.15,
		DropWeight:  26,
		DropRadius:  8,
	},
	WeaponNova: {
		Name:         "Blood Nova",
		AttackStyle:  AttackRadial,
		DamageMul:    0.70,
		CooldownMul:  1.20,
		RangeMul:     0.9,
		AttackRadius: 135,
		DropWeight:   16,
		DropRadius:   11,
	},
	WeaponFang: {
		Name:        "Fang Dagger",
		AttackStyle: AttackSingle,
		DamageMul:   1.35,
		CooldownMul: 1.30,
		RangeMul:    0.80,
		DropWeight:  22,
		DropRadius:  7,
	},
}

var weaponOrder = []WeaponKind{
	WeaponWhip,
	WeaponSpear,
	WeaponNova,
	WeaponFang,
}

func weaponDef(kind WeaponKind) WeaponDef {
	if d, ok := weaponDefs[kind]; ok {
		return d
	}
	return weaponDefs[WeaponWhip]
}

func (w *World) randomWeaponKind() WeaponKind {
	total := 0
	for _, kind := range weaponOrder {
		d := weaponDefs[kind]
		total += d.DropWeight
	}
	if total <= 0 {
		return WeaponWhip
	}
	roll := w.randIntn(total)
	acc := 0
	for _, kind := range weaponOrder {
		d := weaponDefs[kind]
		acc += d.DropWeight
		if roll < acc {
			return kind
		}
	}
	return WeaponWhip
}

func weaponDropChance(kind EnemyKind) float32 {
	switch kind {
	case EnemyTank:
		return 0.42
	case EnemyRunner:
		return 0.22
	default:
		return 0.10
	}
}
