package library

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Ricori/finoka/desktop/internal/managedtools"
)

// preparePlaybackMedia returns a browser-playable local file. Chromium cannot
// play an MPEG transport stream directly, even when its H.264/AAC streams are
// otherwise supported. A TS source is therefore remuxed once into the managed
// video cache without re-encoding it.
func (s *Service) preparePlaybackMedia(id string, entry Entry) (string, error) {
	if !strings.EqualFold(filepath.Ext(entry.SourcePath), ".ts") {
		return s.playbackSourcePath(id), nil
	}

	destination := s.transportStreamPlaybackPath(id)
	if fileExists(destination) {
		return destination, nil
	}
	source := s.playbackSourcePath(id)
	if source == "" || !fileExists(source) {
		return "", errors.New("local media is missing; relink it first")
	}

	ffmpeg, err := managedtools.Find(s.root, "ffmpeg")
	if err != nil {
		return "", errors.New("ffmpeg is required to prepare TS video for playback")
	}
	partial := destination + ".part"
	_ = os.Remove(partial)
	command := exec.Command(ffmpeg,
		"-v", "error", "-y", "-fflags", "+genpts", "-i", source,
		"-map", "0:v:0", "-map", "0:a?", "-c", "copy", "-movflags", "+faststart", "-f", "mp4", partial,
	)
	configureMediaCommand(command)
	if output, runErr := command.CombinedOutput(); runErr != nil {
		_ = os.Remove(partial)
		return "", fmt.Errorf("prepare TS video for playback: %s", strings.TrimSpace(string(output)))
	}
	if err := os.Rename(partial, destination); err != nil {
		_ = os.Remove(partial)
		return "", err
	}
	_ = s.removeCachedMediaExcept(id, destination, entry.SourcePath)
	return destination, nil
}

func (s *Service) playbackSourcePath(id string) string {
	entry, err := s.entryByID(id)
	if err != nil {
		return ""
	}
	if strings.EqualFold(filepath.Ext(entry.SourcePath), ".ts") {
		if playback := s.transportStreamPlaybackPath(id); fileExists(playback) {
			return playback
		}
	}
	if cached := s.cachedPath(id); cached != "" {
		return cached
	}
	return entry.SourcePath
}

func (s *Service) transportStreamPlaybackPath(id string) string {
	return filepath.Join(s.cacheDir, id+".mp4")
}

func (s *Service) removeCachedMediaExcept(id string, keep ...string) error {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	entries, err := os.ReadDir(s.cacheDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var result error
	for _, entry := range entries {
		path := filepath.Join(s.cacheDir, entry.Name())
		if entry.IsDir() || cacheFileID(entry.Name()) != id || samePathAny(path, keep) {
			continue
		}
		result = errors.Join(result, os.Remove(path))
	}
	return result
}

func samePathAny(path string, candidates []string) bool {
	for _, candidate := range candidates {
		if samePath(path, candidate) {
			return true
		}
	}
	return false
}
