package main

import "testing"

func TestSublimeContains(t *testing.T) {
	tests := []struct {
		text   string
		substr string
		pass   bool
	}{
		{"hello", "lo", true},
		{"abcdefg", "cf", true},
		{"abcdefg", "a", true},
		{"abcdefg", "b", true},
		{"abcdefg", "cfa", false},
		{"abcdefg", "aa", false},
		{"世界", "a", false},
		{"Hello 世界", "界", true},
		{"Hello 世界", "elo", true},
	}
	for _, v := range tests {
		res := SublimeContains(v.text, v.substr)
		if res != v.pass {
			t.Fatalf("Failed: %v - res:%v", v, res)
		}
	}
}

func TestCleanPath(t *testing.T) {
	tests := []struct {
		orig   string
		expect string
	}{
		// {"C:\\hello", "C:/hello"}, // Only works in windows
		{"", "."},
		{"//../foo", "/foo"},
		{"/../../", "/"},
		{"/hello/world/..", "/hello"},
		{"/..", "/"},
		{"/foo/..", "/"},
		{"/-/foo", "/-/foo"},
	}
	for _, v := range tests {
		res := cleanPath(v.orig)
		if res != v.expect {
			t.Fatalf("Clean path(%v) expect(%v) but got(%v)", v.orig, v.expect, res)
		}
	}
}
