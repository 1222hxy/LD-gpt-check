package i18n

import "testing"

func TestNormalizeDefaultsToChinese(t *testing.T) {
	if got := Normalize(""); got != ZH {
		t.Fatalf("Normalize empty = %q", got)
	}
	if got := Normalize("unknown"); got != ZH {
		t.Fatalf("Normalize unknown = %q", got)
	}
	if got := Normalize("en-US"); got != EN {
		t.Fatalf("Normalize en-US = %q", got)
	}
}

func TestDetectEnvOverridesConfig(t *testing.T) {
	t.Setenv("LD_GPT_CHECK_LANG", "en")
	if got := Detect("zh-CN"); got != EN {
		t.Fatalf("Detect = %q", got)
	}
}

func TestLocalizerFallback(t *testing.T) {
	l := New(EN)
	if got := l.S("config_not_logged_in"); got != "Login status: not logged in" {
		t.Fatalf("english message = %q", got)
	}
	if got := l.S("missing_key"); got != "missing_key" {
		t.Fatalf("missing key = %q", got)
	}
}

func TestParseBoolInput(t *testing.T) {
	if got, ok := ParseBoolInput("是", false); !ok || !got {
		t.Fatalf("ParseBoolInput yes = %v %v", got, ok)
	}
	if got, ok := ParseBoolInput("no", true); !ok || got {
		t.Fatalf("ParseBoolInput no = %v %v", got, ok)
	}
	if _, ok := ParseBoolInput("maybe", false); ok {
		t.Fatal("expected invalid bool input")
	}
}
