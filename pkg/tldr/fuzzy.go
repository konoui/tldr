package tldr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/sahilm/fuzzy"
)

// CmdInfo contains name, platform, language
type CmdInfo struct {
	Name     string   `json:"name"`
	Platform []string `json:"platform"`
	Language []string `json:"language"`
}

// Cmds a slice of CmdInfo
type Cmds []*CmdInfo

// CmdsIndex structure of index.json
type CmdsIndex struct {
	Commands Cmds `json:"commands"`
}

func (c Cmds) String(i int) string {
	return c[i].Name
}

// Len return length of Commands for fuzzy interface
func (c Cmds) Len() int {
	return len(c)
}

// Filter fuzzy search commands by query
func (c Cmds) Filter(query string) Cmds {
	cmds := Cmds{}
	results := fuzzy.FindFrom(query, c)
	for _, r := range results {
		// Note: replace highfun with space as command name in index file joined highfun
		// e.g.) git-checkout -> git checkout
		cmdName := strings.Replace(c[r.Index].Name, "-", " ", -1)
		c[r.Index].Name = cmdName
		cmds = append(cmds, c[r.Index])
	}

	return cmds
}

// Search fuzzy search commands by query. This is wrapped Filtter
func (c Cmds) Search(args []string) Cmds {
	// Note: We should replace space with highfun as a index file format is joined with highfun
	// e.g.) git checkout -> git-checkout.md
	query := strings.Join(args, "-")
	return c.Filter(query)
}

// LoadIndexFile load command index file
func (t *Tldr) LoadIndexFile() (*CmdsIndex, error) {
	f, err := os.Open(filepath.Join(t.path, t.indexFile))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cmdIndex := &CmdsIndex{}
	return cmdIndex, json.NewDecoder(f).Decode(cmdIndex)
}