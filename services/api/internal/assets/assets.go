// Package assets embeds the data files the API needs: game configuration used by
// the game:init seeding and the achievements logic, bot avatars and the default photo.
package assets

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed game images
var FS embed.FS

// Game returns a file from game/.
func Game(name string) ([]byte, error) { return FS.ReadFile("game/" + name) }

// BotImages lists bot avatar file names sorted by name (Laravel File::files order).
func BotImages() []string {
	entries, err := fs.ReadDir(FS, "images/bots")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && e.Name()[0] != '.' {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}
