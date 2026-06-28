package questions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	SchemaVersion = "1"
	DefaultSuite  = "candy_21"
)

type Bank struct {
	SchemaVersion string     `json:"schema_version"`
	Questions     []Question `json:"questions"`
}

type Question struct {
	ID      string   `json:"id"`
	Version string   `json:"version"`
	Title   string   `json:"title"`
	Prompt  string   `json:"prompt"`
	Tags    []string `json:"tags,omitempty"`
	Grader  Grader   `json:"grader"`
}

type Grader struct {
	Type             string  `json:"type"`
	Expected         string  `json:"expected,omitempty"`
	Tolerance        float64 `json:"tolerance,omitempty"`
	IndependentMatch bool    `json:"independent_match,omitempty"`
	Pattern          string  `json:"pattern,omitempty"`
	CaseSensitive    bool    `json:"case_sensitive,omitempty"`
	TrimSpace        bool    `json:"trim_space,omitempty"`
}

type GradeResult struct {
	OK              bool
	ExpectedAnswer  string
	ExtractedAnswer string
	FailureReason   string
}

type LoadOptions struct {
	File                  string
	URL                   string
	CacheDir              string
	AllowHTTP             bool
	FallbackOnRemoteError bool
}

func Builtin() []Question {
	return []Question{{
		ID:      DefaultSuite,
		Version: "1",
		Title:   "糖果形状口味保证题",
		Prompt: `不使用任何外部工具回答以下问题：

在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形，不同的形状靠手感可以分辨）。现已知不同口味的糖和不同形状的数量统计如下表。参赛者需要在活动前决定摸出的糖果数目，那么，最少取出多少个糖果才能保证手中同时拥有不同形状的苹果味和桃子味的糖？（同时手中有圆形苹果味匹配五角星桃子味糖果，或者有圆形桃子味匹配五角星苹果味糖果都满足要求）

          苹果味 桃子味 西瓜味
圆形        7      9      8
五角星形    7      6      4`,
		Tags: []string{"math", "pigeonhole"},
		Grader: Grader{
			Type:             "number",
			Expected:         "21",
			IndependentMatch: true,
		},
	}}
}

func Load(ctx context.Context, opts LoadOptions) ([]Question, error) {
	questions := Builtin()
	if opts.File != "" {
		q, err := LoadFile(opts.File)
		if err != nil {
			return nil, err
		}
		questions = merge(questions, q)
	}
	if opts.URL != "" {
		q, err := LoadRemote(ctx, opts.URL, opts.CacheDir, opts.AllowHTTP)
		if err != nil {
			if !opts.FallbackOnRemoteError {
				return nil, err
			}
		} else {
			questions = merge(questions, q)
		}
	}
	if err := Validate(questions); err != nil {
		return nil, err
	}
	sort.Slice(questions, func(i, j int) bool { return questions[i].ID < questions[j].ID })
	return questions, nil
}

func LoadFile(path string) ([]Question, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

func LoadRemote(ctx context.Context, rawURL, cacheDir string, allowHTTP bool) ([]Question, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid question url %q", rawURL)
	}
	if u.Scheme != "https" {
		localHTTP := allowHTTP && u.Scheme == "http" && (strings.HasPrefix(u.Host, "localhost") || strings.HasPrefix(u.Host, "127.0.0.1"))
		if !localHTTP {
			return nil, fmt.Errorf("question url must use https")
		}
	}
	if cacheDir == "" {
		cacheDir = DefaultCacheDir()
	}
	cachePath := filepath.Join(cacheDir, cacheName(rawURL)+".json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ld-gpt-check/0.1")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err == nil && resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			b, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			if readErr != nil {
				return nil, readErr
			}
			q, parseErr := Parse(b)
			if parseErr != nil {
				return nil, parseErr
			}
			_ = os.MkdirAll(cacheDir, 0700)
			_ = os.WriteFile(cachePath, b, 0600)
			return q, nil
		}
		err = fmt.Errorf("question url returned HTTP %d", resp.StatusCode)
	}

	if b, readErr := os.ReadFile(cachePath); readErr == nil {
		return Parse(b)
	}
	return nil, err
}

func Parse(b []byte) ([]Question, error) {
	var bank Bank
	if err := json.Unmarshal(b, &bank); err != nil {
		return nil, err
	}
	if bank.SchemaVersion != "" && bank.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported question schema_version %q", bank.SchemaVersion)
	}
	if err := Validate(bank.Questions); err != nil {
		return nil, err
	}
	return bank.Questions, nil
}

func Validate(qs []Question) error {
	seen := map[string]bool{}
	for _, q := range qs {
		if strings.TrimSpace(q.ID) == "" {
			return errors.New("question id is required")
		}
		if seen[q.ID] {
			return fmt.Errorf("duplicate question id %q", q.ID)
		}
		seen[q.ID] = true
		if strings.TrimSpace(q.Version) == "" {
			return fmt.Errorf("question %s version is required", q.ID)
		}
		if strings.TrimSpace(q.Title) == "" {
			return fmt.Errorf("question %s title is required", q.ID)
		}
		if strings.TrimSpace(q.Prompt) == "" {
			return fmt.Errorf("question %s prompt is required", q.ID)
		}
		if err := validateGrader(q); err != nil {
			return err
		}
	}
	return nil
}

func Select(qs []Question, ids string) ([]Question, error) {
	if strings.TrimSpace(ids) == "" {
		ids = DefaultSuite
	}
	index := map[string]Question{}
	for _, q := range qs {
		index[q.ID] = q
	}
	var selected []Question
	seen := map[string]bool{}
	for _, id := range strings.Split(ids, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if seen[id] {
			continue
		}
		q, ok := index[id]
		if !ok {
			return nil, fmt.Errorf("unknown suite %q; available: %s", id, strings.Join(IDs(qs), ", "))
		}
		selected = append(selected, q)
		seen[id] = true
	}
	if len(selected) == 0 {
		return nil, errors.New("no suites selected")
	}
	return selected, nil
}

func IDs(qs []Question) []string {
	out := make([]string, 0, len(qs))
	for _, q := range qs {
		out = append(out, q.ID)
	}
	sort.Strings(out)
	return out
}

func Grade(q Question, answer string) GradeResult {
	expected := q.Grader.Expected
	if expected == "" {
		expected = q.Grader.Pattern
	}
	result := GradeResult{ExpectedAnswer: expected, FailureReason: "unknown"}
	switch q.Grader.Type {
	case "number":
		extracted, ok := extractNumber(answer, q.Grader.Expected, q.Grader.IndependentMatch)
		result.ExtractedAnswer = extracted
		if !ok {
			result.FailureReason = "no_answer"
			return result
		}
		if q.Grader.IndependentMatch {
			result.OK = true
			break
		}
		want, err1 := strconv.ParseFloat(q.Grader.Expected, 64)
		got, err2 := strconv.ParseFloat(extracted, 64)
		if err1 != nil || err2 != nil {
			result.FailureReason = "parse_error"
			return result
		}
		tol := q.Grader.Tolerance
		if tol < 0 {
			tol = 0
		}
		result.OK = abs(got-want) <= tol
	case "exact":
		got := answer
		want := q.Grader.Expected
		if q.Grader.TrimSpace {
			got = strings.TrimSpace(got)
			want = strings.TrimSpace(want)
		}
		if !q.Grader.CaseSensitive {
			got = strings.ToLower(got)
			want = strings.ToLower(want)
		}
		result.ExtractedAnswer = strings.TrimSpace(answer)
		result.OK = got == want
	case "regex":
		re, err := regexp.Compile(q.Grader.Pattern)
		if err != nil {
			result.FailureReason = "parse_error"
			return result
		}
		m := re.FindStringSubmatch(answer)
		if len(m) == 0 {
			result.FailureReason = "no_answer"
			return result
		}
		if len(m) > 1 {
			result.ExtractedAnswer = m[1]
		} else {
			result.ExtractedAnswer = m[0]
		}
		result.OK = true
	}
	if result.OK {
		result.FailureReason = ""
	} else if result.FailureReason == "unknown" {
		result.FailureReason = "wrong_answer"
	}
	return result
}

func PromptHash(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

func DefaultCacheDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "question-cache"
	}
	return filepath.Join(filepath.Dir(exe), "question-cache")
}

func validateGrader(q Question) error {
	switch q.Grader.Type {
	case "number":
		if strings.TrimSpace(q.Grader.Expected) == "" {
			return fmt.Errorf("question %s number grader expected is required", q.ID)
		}
		if _, err := strconv.ParseFloat(q.Grader.Expected, 64); err != nil {
			return fmt.Errorf("question %s number grader expected must be numeric", q.ID)
		}
	case "exact":
		if q.Grader.Expected == "" {
			return fmt.Errorf("question %s exact grader expected is required", q.ID)
		}
	case "regex":
		if q.Grader.Pattern == "" {
			return fmt.Errorf("question %s regex grader pattern is required", q.ID)
		}
		if _, err := regexp.Compile(q.Grader.Pattern); err != nil {
			return fmt.Errorf("question %s regex grader pattern is invalid: %w", q.ID, err)
		}
	default:
		return fmt.Errorf("question %s grader type must be number, exact, or regex", q.ID)
	}
	return nil
}

func merge(base, extra []Question) []Question {
	byID := map[string]Question{}
	for _, q := range base {
		byID[q.ID] = q
	}
	for _, q := range extra {
		byID[q.ID] = q
	}
	out := make([]Question, 0, len(byID))
	for _, q := range byID {
		out = append(out, q)
	}
	return out
}

func cacheName(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(sum[:])
}

func extractNumber(answer, expected string, independent bool) (string, bool) {
	if independent {
		pattern := `(^|[^0-9.-])(` + regexp.QuoteMeta(expected) + `)([^0-9.]|$)`
		re := regexp.MustCompile(pattern)
		m := re.FindStringSubmatch(answer)
		if len(m) >= 3 {
			return m[2], true
		}
		return "", false
	}
	re := regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)
	m := re.FindString(answer)
	return m, m != ""
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
