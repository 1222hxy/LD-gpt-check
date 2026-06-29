package main

import (
	"context"
	"testing"

	"github.com/1222hxy/LD-gpt-check/internal/i18n"
	"github.com/1222hxy/LD-gpt-check/internal/wizard"
)

func TestNoArgsStartsWizard(t *testing.T) {
	old := runWizard
	oldUpdate := runAutoUpdateCheck
	defer func() {
		runWizard = old
		runAutoUpdateCheck = oldUpdate
	}()
	runAutoUpdateCheck = func(ctx context.Context, lang i18n.Lang) bool { return false }

	called := false
	runWizard = func(ctx context.Context, opts wizard.Options) error {
		called = true
		if opts.Version != version {
			t.Fatalf("version = %q", opts.Version)
		}
		if opts.Lang != i18n.ZH {
			t.Fatalf("lang = %q", opts.Lang)
		}
		return nil
	}

	if err := run(context.Background(), nil, i18n.ZH); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected wizard to start")
	}
}

func TestHelpDoesNotStartWizard(t *testing.T) {
	old := runWizard
	oldUpdate := runAutoUpdateCheck
	defer func() {
		runWizard = old
		runAutoUpdateCheck = oldUpdate
	}()
	runAutoUpdateCheck = func(ctx context.Context, lang i18n.Lang) bool {
		t.Fatal("help should not check for updates")
		return false
	}

	runWizard = func(ctx context.Context, opts wizard.Options) error {
		t.Fatal("wizard should not start for help")
		return nil
	}

	if err := run(context.Background(), []string{"help"}, i18n.ZH); err != nil {
		t.Fatal(err)
	}
}
