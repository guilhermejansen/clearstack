package main

import (
	"fmt"

	"github.com/guilhermejansen/clearstack/internal/config"
	"github.com/guilhermejansen/clearstack/internal/detectors"
	"github.com/guilhermejansen/clearstack/internal/engine"
	"github.com/guilhermejansen/clearstack/internal/journal"
	"github.com/guilhermejansen/clearstack/internal/trash"
)

// buildEngineOptions is the slice of options shared by all commands that
// need a live engine instance.
type buildEngineOptions struct {
	// Categories restricts the detector set for this invocation.
	Categories []detectors.Category
	// WithCleaner wires a Cleaner + Journal when true (Clean/Undo).
	WithCleaner bool
}

// loadConfigAndEngine is the single entry point commands use to go from the
// process environment to a ready-to-use Engine.
func loadConfigAndEngine(opts buildEngineOptions) (*engine.Engine, *journal.Journal, error) {
	cfg, err := config.Load(globalFlags.ConfigPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	if globalFlags.Profile != "" {
		next, err := config.ApplyProfile(*cfg, globalFlags.Profile)
		if err != nil {
			return nil, nil, err
		}
		cfg = &next
	}

	minAge, err := cfg.Dormancy.ParseMinAge()
	if err != nil {
		return nil, nil, err
	}
	dp := engine.DormancyPolicy{
		MinAge:   minAge,
		CheckGit: cfg.Dormancy.CheckGit,
	}

	var j *journal.Journal
	var cleaner *engine.Cleaner
	if opts.WithCleaner {
		jj, jerr := journal.Open()
		if jerr != nil {
			return nil, nil, fmt.Errorf("journal: %w", jerr)
		}
		j = jj
		cleaner = engine.NewCleaner(trash.New(), j, engine.NewSafety(cfg.Safety.WhitelistPaths...), detectors.Default)
	}

	eng, err := engine.New(engine.Config{
		Registry:   detectors.Default,
		Safety:     engine.NewSafety(cfg.Safety.WhitelistPaths...),
		Dormancy:   dp,
		Categories: opts.Categories,
		Cleaner:    cleaner,
		Journal:    j,
	})
	if err != nil {
		return nil, nil, err
	}
	return eng, j, nil
}
