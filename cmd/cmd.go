package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/konoui/go-alfred"
	"github.com/konoui/tldr/pkg/tldr"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var (
	outStream io.Writer = os.Stdout
	errStream io.Writer = os.Stderr
)

var (
	op tldr.Options
)

func init() {
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "osx"
	}
	op = tldr.Options{
		Platform: platform,
		Language: "",
		Update:   false,
	}
}

// NewRootCmd create a new cmd for root
func NewRootCmd() *cobra.Command {
	var isWorkflow bool
	var enableFuzzy bool
	rootCmd := &cobra.Command{
		Use:   "tldr <cmd>",
		Short: "show cmd examples",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args, op, isWorkflow, enableFuzzy)
		},
		SilenceUsage: true,
	}
	rootCmd.PersistentFlags().StringVarP(&op.Platform, "platform", "p", op.Platform, "platform")
	//rootCmd.PersistentFlags().StringVarP(&op.Language, "language", "l", op.Language, "language")
	rootCmd.PersistentFlags().BoolVarP(&op.Update, "update", "u", op.Update, "update")
	rootCmd.PersistentFlags().BoolVarP(&isWorkflow, "workflow", "w", false, "rendering for alfred workflow")
	rootCmd.PersistentFlags().BoolVarP(&enableFuzzy, "fuzzy", "f", false, "enable fuzzy search for cmds")

	return rootCmd
}

// Execute Execute root cmd
func Execute(rootCmd *cobra.Command) {
	rootCmd.SetOutput(outStream)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(errStream, err)
		os.Exit(1)
	}
}

func run(cmds []string, op tldr.Options, isWorkflow, enableFuzzy bool) error {
	const tldrDir = ".tldr"
	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, tldrDir)

	t := tldr.NewTldr(path, op)

	err = t.OnInitialize()
	if !tldr.IsCacheExpired(err) && err != nil {
		return err
	}

	// workflow will not show cache expired message
	if isWorkflow {
		renderToWorkflow(t, cmds, enableFuzzy)
		return nil
	}

	// cli will show cache expired message
	if err != nil {
		cacheExpiredMsg := err.Error()
		fmt.Fprintf(errStream, "%s\n", cacheExpiredMsg)
	}
	renderToOut(t, cmds)
	return nil
}

const (
	bold  = "\x1b[1m"
	blue  = "\x1b[34m"
	green = "\x1b[32m"
	red   = "\x1b[31m"
	reset = "\x1b[33;0m"
)

func renderToOut(t *tldr.Tldr, cmds []string) {
	p, err := t.FindPage(cmds)
	if err != nil {
		fmt.Fprintln(errStream, "This page doesn't exist yet!\nSubmit new pages here: https://github.com/tldr-pages/tldr")
		return
	}

	coloredCmdName := bold + p.CmdName + reset
	fmt.Fprintln(outStream, coloredCmdName)
	fmt.Fprintln(outStream)
	fmt.Fprintln(outStream, p.CmdDescription)
	for _, cmd := range p.CmdExamples {
		coloredDescription := "- " + green + cmd.Description + reset
		fmt.Fprintln(outStream, coloredDescription)
		line := strings.Replace(cmd.Cmd, "{{", blue, -1)
		line = strings.Replace(line, "}}", red, -1)
		coloredCmd := red + line + reset
		fmt.Fprintln(outStream, coloredCmd)
		fmt.Fprintln(outStream)
	}
}

func renderToWorkflow(t *tldr.Tldr, cmds []string, enableFuzzy bool) {
	awf := alfred.NewWorkflow()
	awf.SetStreams(outStream, outStream)
	awf.EmptyWarning("No matching query", "Try a different query")

	p, _ := t.FindPage(cmds)
	for _, cmd := range p.CmdExamples {
		awf.Append(alfred.Item{
			Title:        cmd.Cmd,
			Subtitle:     cmd.Description,
			Autocomplete: cmd.Cmd,
			Arg:          cmd.Cmd,
		})
	}

	if enableFuzzy && len(p.CmdExamples) == 0 {
		index, err := t.LoadIndexFile()
		if err != nil {
			awf.Fatal(fmt.Sprintf("an error occurs: %s", err), "")
			return
		}

		suggestions := index.Commands.Search(cmds)
		for _, cmd := range suggestions {
			awf.Append(alfred.Item{
				Title:        cmd.Name,
				Subtitle:     fmt.Sprintf("Platforms: %s", strings.Join(cmd.Platform, ",")),
				Autocomplete: cmd.Name,
			})
		}
	}

	awf.Output()
}
