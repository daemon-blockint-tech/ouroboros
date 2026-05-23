package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/daemon-blockint-tech/ouroboros/internal/endpoint"
	"github.com/daemon-blockint-tech/ouroboros/internal/exposure"
	"github.com/daemon-blockint-tech/ouroboros/internal/model"
	"github.com/daemon-blockint-tech/ouroboros/internal/output"
	"github.com/daemon-blockint-tech/ouroboros/internal/scanner"
	"github.com/daemon-blockint-tech/ouroboros/internal/store"
)

// runAgent runs periodic inventory scans and persists results to SQLite.
func runAgent(args []string) int {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)
	profile := fs.String("profile", "baseline", "scan profile: baseline|project|deep")
	interval := fs.Duration("interval", time.Hour, "time between scans")
	dbPath := fs.String("db", "", "SQLite path (default ~/.ouroboros/agent.db)")
	exposureCatalog := fs.String("exposure-catalog", "", "exposure catalog file or directory")
	semverMatch := fs.Bool("semver-match", true, "semver constraints in catalog versions")
	once := fs.Bool("once", false, "run one scan then exit")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ouroboros agent [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: %v\n", err)
		return 1
	}
	path := *dbPath
	if path == "" {
		path = store.PathDefault(home)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "agent: mkdir: %v\n", err)
		return 1
	}

	st, err := store.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: open db: %v\n", err)
		return 1
	}
	defer st.Close()

	var catalog *exposure.Catalog
	if *exposureCatalog != "" {
		catalog, err = exposure.Load(*exposureCatalog, 32<<20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "agent: catalog: %v\n", err)
			return 1
		}
		catalog.SemverMatch = *semverMatch
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	run := func() error {
		return agentScanOnce(ctx, st, *profile, catalog)
	}

	if *once {
		if err := run(); err != nil {
			fmt.Fprintf(os.Stderr, "agent: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Fprintf(os.Stderr, "ouroboros agent: db=%s interval=%s profile=%s\n", path, *interval, *profile)
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "agent: %v\n", err)
		return 1
	}
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return 0
		case <-ticker.C:
			if err := run(); err != nil {
				fmt.Fprintf(os.Stderr, "agent: scan: %v\n", err)
			}
		}
	}
}

func agentScanOnce(ctx context.Context, st *store.SQLite, profile string, catalog *exposure.Catalog) error {
	deviceID, _ := resolveDeviceID("")
	ep := endpoint.Current(deviceID)
	runID := newRunID()
	started := time.Now().UTC()
	scanID, err := st.BeginScan(ctx, runID, ep.DeviceID, profile, currentVersion(), started)
	if err != nil {
		return err
	}

	roots, _, err := resolveRoots(profile, nil, rootsOpts{})
	if err != nil {
		return err
	}

	emitter := output.New(io.Discard, os.Stderr, runID)
	base := model.Record{
		RecordType:     model.RecordTypePackage,
		SchemaVersion:  model.SchemaVersion,
		ScannerName:    model.ScannerName,
		ScannerVersion: currentVersion(),
		RunID:          runID,
		ScanTime:       started.Format(time.RFC3339Nano),
		Endpoint:       ep,
		Profile:        profile,
	}

	var stored int
	cfg := scanner.Config{
		Profile:      profile,
		Roots:        roots,
		Catalog:      catalog,
		BaseRecord:   base,
		Emitter:      emitter,
		Concurrency:  4,
		OnPackageObserved: func(r model.Record) error {
			stored++
			return st.UpsertComponent(ctx, ep.DeviceID, runID, r, time.Now().UTC())
		},
	}
	res, err := scanner.Run(ctx, cfg)
	if err != nil {
		return err
	}
	if err := st.FinishScan(ctx, scanID, time.Now().UTC()); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "agent: scan %s stored %d packages (%d emitted, %d dup)\n",
		runID, stored, res.RecordsEmitted, res.Duplicates)
	return nil
}
