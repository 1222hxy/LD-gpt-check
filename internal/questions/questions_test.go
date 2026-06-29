package questions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinCandyQuestionIsOriginalAndGrades21(t *testing.T) {
	q := Builtin()[0]
	if q.ID != DefaultSuite {
		t.Fatalf("id = %q", q.ID)
	}
	for _, want := range []string{
		"不使用任何外部工具回答以下问题：",
		"在一个黑色的袋子里放有三种口味的糖果",
		"圆形苹果味匹配五角星桃子味糖果",
		"五角星形    7      6      4",
	} {
		if !strings.Contains(q.Prompt, want) {
			t.Fatalf("builtin prompt missing %q", want)
		}
	}
	if !Grade(q, "最少需要取出 **21 个**。").OK {
		t.Fatal("expected independent 21 to pass")
	}
	if Grade(q, "答案是 121。").OK {
		t.Fatal("121 must not pass as independent 21")
	}
}

func TestExactAndRegexGraders(t *testing.T) {
	exact := Question{
		ID: "exact", Version: "1", Title: "Exact", Prompt: "p",
		Grader: Grader{Type: "exact", Expected: "Answer", TrimSpace: true},
	}
	if !Grade(exact, " answer \n").OK {
		t.Fatal("expected case-insensitive trimmed exact match")
	}

	regexQ := Question{
		ID: "regex", Version: "1", Title: "Regex", Prompt: "p",
		Grader: Grader{Type: "regex", Pattern: `答案[:：]\s*(\d+)`},
	}
	got := Grade(regexQ, "最终答案：42")
	if !got.OK || got.ExtractedAnswer != "42" {
		t.Fatalf("regex grade = %#v", got)
	}
}

func TestParseQuestionBank(t *testing.T) {
	data := []byte(`{
	  "schema_version": "1",
	  "questions": [{
	    "id": "custom_1",
	    "version": "1",
	    "title": "Custom",
	    "prompt": "Question?",
	    "grader": {"type": "number", "expected": "3", "independent_match": true}
	  }]
	}`)
	qs, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 1 || qs[0].ID != "custom_1" {
		t.Fatalf("questions = %#v", qs)
	}
}

func TestLoadFallsBackWhenDefaultRemoteFails(t *testing.T) {
	qs, err := Load(context.Background(), LoadOptions{
		URL:                   "https://127.0.0.1:1/questions.json",
		CacheDir:              t.TempDir(),
		FallbackOnRemoteError: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 1 || qs[0].ID != DefaultSuite {
		t.Fatalf("questions = %#v", qs)
	}
}

func TestLoadRemoteNoCacheDoesNotReadCachedQuestionBank(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, cacheName(server.URL)+".json")
	if err := os.WriteFile(cachePath, []byte(`{
	  "schema_version": "1",
	  "questions": [{
	    "id": "cached",
	    "version": "1",
	    "title": "Cached",
	    "prompt": "Cached?",
	    "grader": {"type": "number", "expected": "1"}
	  }]
	}`), 0600); err != nil {
		t.Fatal(err)
	}

	cached, err := LoadRemote(context.Background(), server.URL, cacheDir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(cached) != 1 || cached[0].ID != "cached" {
		t.Fatalf("cached questions = %#v", cached)
	}

	if _, err := LoadRemoteNoCache(context.Background(), server.URL, true); err == nil {
		t.Fatal("expected no-cache remote load to fail instead of reading cache")
	}
}
