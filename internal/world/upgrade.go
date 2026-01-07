package world

import "fmt"

type UpgradeKind int

const (
	UpDamage UpgradeKind = iota
	UpAttackSpeed
)

type UpgradeOption struct {
	Kind  UpgradeKind
	Title string
	Desc  string
}

type UpgradeMenu struct {
	Active  bool
	Option  [2]UpgradeOption
	Pending int // how many level-up choices still need to be picked
}

func (w *World) openUpgradeMenuIfNeeded() {
	if w.Upgrade.Pending <= 0 || w.Upgrade.Active {
		return
	}

	// v0.1: always present two fixed choices
	w.Upgrade.Option[0] = UpgradeOption{
		Kind:  UpDamage,
		Title: "1) +Damage",
		Desc:  fmt.Sprintf("Increase damage by +%.0f", 10.0),
	}

	w.Upgrade.Option[1] = UpgradeOption{
		Kind:  UpAttackSpeed,
		Title: "2) Faster Attack",
		Desc:  "Reduce attack cooldown by 15%",
	}

	w.Upgrade.Active = true
}

func (w *World) applyUpGradeChoice(choice int) {
	if !w.Upgrade.Active {
		return
	}

	if choice < 0 || choice > 1 {
		return
	}

	opt := w.Upgrade.Option[choice]

	switch opt.Kind {
	case UpDamage:
		w.Player.Damage += 10
	case UpAttackSpeed:
		// Lower cooldown means faster attacks. Clamp to avoid going to 0
		w.Player.AttackCooldown = maxf(0.12, w.Player.AttackCooldown*0.85)
		// If timer is loner than new cooldown, clamp it too
		if w.Player.AttackTimer > w.Player.AttackCooldown {
			w.Player.AttackTimer = w.Player.AttackCooldown
		}
	}

	// consume one pending upgrade choice
	if w.Upgrade.Pending > 0 {
		w.Upgrade.Pending--
	}

	// If more pending (e.g., big XP pickup caused multiple levels), show again
	if w.Upgrade.Pending > 0 {
		w.Upgrade.Active = false
		w.openUpgradeMenuIfNeeded()
		return
	}

	w.Upgrade.Active = false
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
