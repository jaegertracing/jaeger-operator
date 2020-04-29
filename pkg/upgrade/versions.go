package upgrade

var (
	upgrades = map[string]upgradeFunction{
		"1.15.0": upgrade1_15_0,
		"1.17.0": upgrade1_17_0,
	}
)
