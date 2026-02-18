package tasks

import (
	"testing"
)

func TestMatchesTags_NoFilter(t *testing.T) {
	if !matchesTags([]string{"install"}, nil, nil) {
		t.Error("expected match with no filter")
	}
	// Task with no tags also runs when there is no filter
	if !matchesTags(nil, nil, nil) {
		t.Error("expected tagless task to run with no filter")
	}
}

func TestMatchesTags_FilterMatch(t *testing.T) {
	if !matchesTags([]string{"install", "nginx"}, []string{"nginx"}, nil) {
		t.Error("expected match on 'nginx'")
	}
}

func TestMatchesTags_FilterNoMatch(t *testing.T) {
	if matchesTags([]string{"install"}, []string{"deploy"}, nil) {
		t.Error("expected no match: task has 'install', filter wants 'deploy'")
	}
}

func TestMatchesTags_SkipTags(t *testing.T) {
	if matchesTags([]string{"install"}, nil, []string{"install"}) {
		t.Error("expected task to be skipped")
	}
}

func TestMatchesTags_SkipTakesPrecedence(t *testing.T) {
	// Both filter and skip include the same tag – skip wins.
	if matchesTags([]string{"nginx"}, []string{"nginx"}, []string{"nginx"}) {
		t.Error("expected skip to take precedence over filter")
	}
}

func TestMatchesTags_NoTagsWithFilter(t *testing.T) {
	// Task has no tags but a tag filter is active → skip it.
	if matchesTags(nil, []string{"deploy"}, nil) {
		t.Error("expected tagless task to be skipped when filter is active")
	}
}

func TestExpandVars_Basic(t *testing.T) {
	result, err := expandVars("echo {{.version}}", map[string]interface{}{"version": "1.2.3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo 1.2.3" {
		t.Errorf("expected 'echo 1.2.3', got %q", result)
	}
}

func TestExpandVars_NoVars(t *testing.T) {
	result, err := expandVars("echo hello", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", result)
	}
}

func TestExpandVars_EmptyString(t *testing.T) {
	result, err := expandVars("", map[string]interface{}{"k": "v"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestMergeVars(t *testing.T) {
	a := map[string]interface{}{"x": "1", "y": "original"}
	b := map[string]interface{}{"y": "override", "z": "3"}
	merged := mergeVars(a, b)
	if merged["x"] != "1" {
		t.Errorf("expected x=1, got %v", merged["x"])
	}
	if merged["y"] != "override" {
		t.Errorf("expected y=override (b wins), got %v", merged["y"])
	}
	if merged["z"] != "3" {
		t.Errorf("expected z=3, got %v", merged["z"])
	}
}
