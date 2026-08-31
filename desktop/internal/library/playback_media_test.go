package library

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMediaURLServesMP4PlaybackCacheForTransportStream(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.ts")
	if err := os.WriteFile(source, []byte("transport-stream-fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	tools := fixtureTools{metadata: Metadata{Duration: 42.5, Width: 1920, Height: 1080, HasVideo: true, HasAudio: true}}
	service, err := newServiceWithTools(filepath.Join(root, "data"), tools, tools)
	if err != nil {
		t.Fatal(err)
	}
	entry := service.Import([]string{source}).Added[0]
	playback := service.transportStreamPlaybackPath(entry.ID)
	if err := os.WriteFile(playback, []byte("mp4-playback-fixture"), 0o600); err != nil {
		t.Fatal(err)
	}

	url, err := service.MediaURL(entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer service.media.Close()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "video/mp4") || string(body) != "mp4-playback-fixture" {
		t.Fatalf("unexpected playback response: status=%d type=%q body=%q", response.StatusCode, response.Header.Get("Content-Type"), body)
	}
}
