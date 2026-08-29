package brand

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type appstreamMeta struct {
	XMLName xml.Name `xml:"component"`
	URLs    []struct {
		Type string `xml:"type,attr"`
		Text string `xml:",chardata"`
	} `xml:"url"`
	Custom []struct {
		Key  string `xml:"key,attr"`
		Text string `xml:",chardata"`
	} `xml:"custom>value"`
	Releases []struct {
		Version string `xml:"version,attr"`
		URL     string `xml:"url"`
	} `xml:"releases>release"`
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func TestAppImageInfo(t *testing.T) {
	root := repoRoot(t)
	version := strings.TrimSpace(string(readFile(t, filepath.Join(root, "VERSION"))))
	desktop := string(readFile(t, filepath.Join(root, "data", "io.github.lamha.Lamha.desktop")))
	raw := readFile(t, filepath.Join(root, "data", "io.github.lamha.Lamha.metainfo.xml"))

	var meta appstreamMeta
	if err := xml.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("metainfo: %v", err)
	}

	if !strings.Contains(desktop, "X-AppImage-UpdateInformation="+UpdateInformation) {
		t.Fatalf("desktop file missing AppImage update information %q", UpdateInformation)
	}
	if !strings.Contains(desktop, "Exec=lamha") {
		t.Fatal("desktop file missing Exec=lamha")
	}
	if !strings.Contains(desktop, "Icon="+Name) {
		t.Fatalf("desktop file missing Icon=%s", Name)
	}

	if urlType(meta, "homepage") != SourceURL {
		t.Fatalf("metainfo homepage = %q, want %q", urlType(meta, "homepage"), SourceURL)
	}
	if urlType(meta, "bugtracker") != IssuesURL {
		t.Fatalf("metainfo bugtracker = %q, want %q", urlType(meta, "bugtracker"), IssuesURL)
	}
	if urlType(meta, "vcs-browser") != SourceURL {
		t.Fatalf("metainfo vcs-browser = %q, want %q", urlType(meta, "vcs-browser"), SourceURL)
	}
	if customValue(meta, "X-AppImage-UpdateInformation") != UpdateInformation {
		t.Fatalf("metainfo update information = %q, want %q", customValue(meta, "X-AppImage-UpdateInformation"), UpdateInformation)
	}
	if len(meta.Releases) == 0 {
		t.Fatal("metainfo has no releases")
	}
	latest := meta.Releases[0]
	if latest.Version != version {
		t.Fatalf("metainfo latest release %q does not match VERSION %q", latest.Version, version)
	}
	wantRelease := SourceURL + "/releases/tag/v" + version
	if latest.URL != wantRelease {
		t.Fatalf("metainfo latest release url = %q, want %q", latest.URL, wantRelease)
	}
	if UpdateURL != SourceURL+"/releases/latest" {
		t.Fatalf("UpdateURL = %q", UpdateURL)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func urlType(meta appstreamMeta, kind string) string {
	for _, u := range meta.URLs {
		if u.Type == kind {
			return strings.TrimSpace(u.Text)
		}
	}
	return ""
}

func customValue(meta appstreamMeta, key string) string {
	for _, v := range meta.Custom {
		if v.Key == key {
			return strings.TrimSpace(v.Text)
		}
	}
	return ""
}
