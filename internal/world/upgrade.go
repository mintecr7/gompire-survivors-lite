package world

type UpgradeKind int

const (
	UpDamage UpgradeKind = iota
	UpAttackSpeed
	UpMagnet
)

type UpgradeOption struct {
	Kind  UpgradeKind
	Title string
	Desc  string
}

type UpgradeMenu struct {
	Active  bool
	Options [2]UpgradeOption
	Pending int // how many level-up choices still need to be picked
}

func (w *World) openUpgradeMenuIfNeeded() {
	if w.Upgrade.Pending <= 0 || w.Upgrade.Active {
		return
	}

	// v0.1: always present two fixed choices
	pool := []UpgradeOption{
		{
			Kind:  UpDamage,
			Title: "1) +Damage",
			Desc:  "Increase damage by +10",
		},
		{
			Kind:  UpAttackSpeed,
			Title: "2) Faster Attack",
			Desc:  "Reduce attack cooldown by 15%",
		},
		{
			Kind:  UpMagnet,
			Title: "Magnet",
			Desc:  "Increase XP pickup radius by +15",
		},
	}

	// pick 2 distinct options from pool
	// First pick:
	i := w.rng.Intn(len(pool))
	first := pool[i]
	pool = append(pool[:i], pool[i+1:]...)

	// Second pick:
	j := w.rng.Intn(len(pool))
	second := pool[j]

	// Assign to menu. We want keys 1 and 2 to always correspond.
	// Ensure the titles match "1)" and "2)".
	first.Title = "1) " + stripPrefix(first.Title)
	second.Title = "2) " + stripPrefix(second.Title)

	w.Upgrade.Options[0] = first
	w.Upgrade.Options[1] = second
	w.Upgrade.Active = true
}

func (w *World) applyUpGradeChoice(choice int) {
	if !w.Upgrade.Active {
		return
	}

	if choice < 0 || choice > 1 {
		return
	}

	opt := w.Upgrade.Options[choice]

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
	case UpMagnet:
		w.Player.XPMagnet += 15
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

func stripPrefix(s string) string {
	if len(s) >= 3 && (s[0] == '1' || s[0] == '2') && s[1] == ')' {
		return s[3:]
	}
	return s
}
