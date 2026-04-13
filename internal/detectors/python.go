package detectors

// Python detectors.
//
// __pycache__ is always safe to delete (regenerated on next import). We
// intentionally leave .venv at caution so it only appears when a user opts
// in via the aggressive profile or explicit --categories flag.

func init() {
	Register(&SimpleDirDetector{
		CategoryID:  "pycache",
		DirName:     "__pycache__",
		Desc:        "Python bytecode cache",
		SafetyLevel: SafetySafe,
		Strategy:    StrategyHardDelete,
	})
	Register(&SimpleDirDetector{
		CategoryID:  "pytest_cache",
		DirName:     ".pytest_cache",
		Desc:        "pytest cache",
		SafetyLevel: SafetySafe,
		Strategy:    StrategyHardDelete,
	})
	Register(&SimpleDirDetector{
		CategoryID:  "mypy_cache",
		DirName:     ".mypy_cache",
		Desc:        "mypy type-check cache",
		SafetyLevel: SafetySafe,
		Strategy:    StrategyHardDelete,
	})
	Register(&SimpleDirDetector{
		CategoryID:  "ruff_cache",
		DirName:     ".ruff_cache",
		Desc:        "Ruff linter cache",
		SafetyLevel: SafetySafe,
		Strategy:    StrategyHardDelete,
	})
	Register(&SimpleDirDetector{
		CategoryID:  "tox_cache",
		DirName:     ".tox",
		Desc:        "tox environments",
		SafetyLevel: SafetySafe,
		Strategy:    StrategyTrash,
	})
	// .venv is regenerable but expensive — caution level, dormant-only.
	Register(&SimpleDirDetector{
		CategoryID:    "venv_dormant",
		DirName:       ".venv",
		Desc:          "Dormant Python virtualenv (.venv)",
		SafetyLevel:   SafetyCaution,
		Strategy:      StrategyTrash,
		NeedsDormancy: true,
		Require:       []string{"pyproject.toml"},
	})
}
