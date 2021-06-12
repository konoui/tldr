package tldr

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sahilm/fuzzy"
)

// CmdInfo contains name, platform, language
type CmdInfo struct {
	Name      string     `json:"name"`
	Platforms []Platform `json:"platform"`
	Languages []string   `json:"language"`
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

// Search fuzzy search commands by query.
func (c Cmds) Search(args []string) Cmds {
	// Note: We should replace space with highfun before search as a index file format is joined with highfun
	// e.g.) git checkout -> git-checkout.md
	query := strings.Join(args, "-")
	results := fuzzy.FindFrom(query, c)
	cmds := make(Cmds, results.Len())
	for i, r := range results {
		// Note: replace highfun with space after search
		// e.g.) git-checkout -> git checkout
		cmdName := strings.Replace(c[r.Index].Name, "-", " ", -1)
		c[r.Index].Name = cmdName
		cmds[i] = c[r.Index]
	}

	return cmds
}

// LoadIndexFile load command index file
func (t *Tldr) LoadIndexFile() (*CmdsIndex, error) {
	f, err := os.Open(t.indexFilePath())
	if err != nil {
		return nil, fmt.Errorf("failed to open a index file: %w", err)
	}
	defer f.Close()

	cmdIndex := &CmdsIndex{}
	return cmdIndex, json.NewDecoder(f).Decode(cmdIndex)
}
