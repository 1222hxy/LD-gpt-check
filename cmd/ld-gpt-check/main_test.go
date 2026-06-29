package main

import (
	"bytes"
	"context"
	"strings"
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

func TestPrintVersionIncludesCommitMetadata(t *testing.T) {
	oldCommit := gitCommit
	oldCommitDate := gitCommitDate
	oldModified := gitModified
	oldRecent := recentCommits
	defer func() {
		gitCommit = oldCommit
		gitCommitDate = oldCommitDate
		gitModified = oldModified
		recentCommits = oldRecent
	}()

	gitCommit = "1234567890abcdef"
	gitCommitDate = "2026-06-29T15:00:00+08:00"
	gitModified = "false"
	recentCommits = "1234567 First commit|89abcde Second commit"

	var out bytes.Buffer
	printVersion(&out)
	text := out.String()
	if !strings.HasPrefix(text, version+"\n") {
		t.Fatalf("version output should keep raw version as first line:\n%s", text)
	}
	for _, want := range []string{"commit: 1234567890ab (2026-06-29T15:00:00+08:00)", "recent commits:", "1234567 First commit", "89abcde Second commit"} {
		if !strings.Contains(text, want) {
			t.Fatalf("version output missing %q:\n%s", want, text)
		}
	}
}
