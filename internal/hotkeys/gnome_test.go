package hotkeys

import "testing"

func TestMergeBindingPathsKeepsOthers(t *testing.T) {
	got := mergeBindingPaths(
		[]string{"/custom/other/"},
		[]string{bindingPrefix + "lamha-area/"},
	)
	if len(got) != 2 || got[0] != "/custom/other/" || got[1] != bindingPrefix+"lamha-area/" {
		t.Fatalf("mergeBindingPaths() = %#v", got)
	}
}

func TestMergeBindingPathsIdempotent(t *testing.T) {
	ours := bindingPaths()
	got := mergeBindingPaths(ours, ours)
	if !sameStrings(got, ours) {
		t.Fatalf("mergeBindingPaths() = %#v, want %#v", got, ours)
	}
}
