package deepseek

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPrepareSearch(t *testing.T) {
	t.Parallel()

	got, err := PrepareSearch(SearchInput{Query: "DeepSeek site:github.com"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.Count != 10 || got.Request.NeedContent || got.MaxContentChars != 4000 {
		t.Fatalf("defaults: %+v", got)
	}

	count := 5
	maxContent := 0
	got, err = PrepareSearch(SearchInput{
		Query:           "  spaced query  ",
		Count:           &count,
		OriQuestion:     "original",
		Localization:    "203.0.113.8",
		NeedContent:     true,
		MaxContentChars: &maxContent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.Query != "  spaced query  " || got.Request.Count != 5 || got.Request.OriQuestion != "original" {
		t.Fatalf("normalized request: %+v", got.Request)
	}
	if got.Request.Localization != "203.0.113.8" || !got.Request.NeedContent || got.MaxContentChars != 0 {
		t.Fatalf("flags: %+v", got)
	}

	long := strings.Repeat("界", 129)
	if _, err := PrepareSearch(SearchInput{Query: long}); err == nil {
		t.Fatal("expected query length error")
	}
	if _, err := PrepareSearch(SearchInput{Query: "   "}); err == nil {
		t.Fatal("expected blank query error")
	}
	zero := 0
	if _, err := PrepareSearch(SearchInput{Query: "q", Count: &zero}); err == nil {
		t.Fatal("expected count error")
	}
	tooMany := 51
	if _, err := PrepareSearch(SearchInput{Query: "q", Count: &tooMany}); err == nil {
		t.Fatal("expected count upper bound error")
	}
	if _, err := PrepareSearch(SearchInput{Query: "q", OriQuestion: "   "}); err == nil {
		t.Fatal("expected ori_question error")
	}
	if _, err := PrepareSearch(SearchInput{Query: "q", Localization: "not-an-ip"}); err == nil {
		t.Fatal("expected localization error")
	}
	neg := -1
	if _, err := PrepareSearch(SearchInput{Query: "q", MaxContentChars: &neg}); err == nil {
		t.Fatal("expected max_content_chars error")
	}
}

func TestPrepareFetch(t *testing.T) {
	t.Parallel()

	got, err := PrepareFetch(FetchInput{URL: "https://example.com/a"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.URL != "https://example.com/a" || got.MaxChars != 0 {
		t.Fatalf("%+v", got)
	}

	limit := 12
	got, err = PrepareFetch(FetchInput{URL: "http://example.com", MaxChars: &limit})
	if err != nil || got.MaxChars != 12 {
		t.Fatalf("limit: %+v %v", got, err)
	}

	cases := []string{
		"",
		"   ",
		"example.com",
		"ftp://example.com",
		"https://user:pass@example.com/a",
		"http://",
		strings.Repeat("a", 8193),
	}
	for _, raw := range cases {
		if _, err := PrepareFetch(FetchInput{URL: raw}); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
	neg := -5
	if _, err := PrepareFetch(FetchInput{URL: "https://example.com", MaxChars: &neg}); err == nil {
		t.Fatal("expected max_chars error")
	}
	if n := utf8.RuneCountInString(strings.Repeat("界", 8192)); n != 8192 {
		t.Fatal(n)
	}
	if _, err := PrepareFetch(FetchInput{URL: "https://example.com/" + strings.Repeat("界", 8000)}); err != nil {
		t.Fatal(err)
	}
}
