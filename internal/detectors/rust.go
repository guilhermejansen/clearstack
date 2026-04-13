package detectors

// Rust cargo detectors.
//
// `target/` is safe to delete but only when the parent is an actual crate
// (has Cargo.toml) and the project is dormant — rebuilding is expensive.

func init() {
	Register(&SimpleDirDetector{
		CategoryID:    "rust_target",
		DirName:       "target",
		Desc:          "Rust build output (cargo target/)",
		SafetyLevel:   SafetySafe,
		Strategy:      StrategyTrash,
		NeedsDormancy: true,
		Require:       []string{"Cargo.toml"},
	})
}
