package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/konoui/alfred-tldr/pkg/tldr"
	"github.com/konoui/go-alfred"
)

func (cfg *config) printPage(cmds []string) error {
	// insert update recommendation first
	ctx := context.Background()
	if isUpdateWorkflowRecommendEnabled() && awf.Updater().NewerVersionAvailable(ctx) {
		awf.Append(
			alfred.NewItem().
				Title("Newer tldr wrokflow is available!").
				Subtitle("Please Enter!").
				Arg(fmt.Sprintf("--%s --%s", updateWorkflowFlag, confirmFlag)).
				Variable(nextActionKey, nextActionShell).
				Icon(awf.Assets().IconAlertNote()),
		)
	}

	if isUpdateDBRecommendEnabled() && cfg.tldrClient.Expired(twoWeeks) {
		awf.Append(
			alfred.NewItem().
				Title("Tldr database is older than 2 weeks").
				Subtitle("Please Enter!").
				Arg(fmt.Sprintf("--%s --%s", longUpdateFlag, confirmFlag)).
				Icon(awf.Assets().IconAlertNote()).
				Variable(nextActionKey, nextActionShell),
		)
	}

	// no input case
	if len(cmds) == 0 {
		awf.Append(
			alfred.NewItem().
				Title("Please input a command").
				Subtitle("e.g.) tldr tar e.g.) tldr --help").
				Valid(false),
		).Output()
		return nil
	}

	awf.SetEmptyWarning("No matching query", "Try a different query")
	p, err := cfg.tldrClient.FindPage(cmds)
	if err != nil {
		if errors.Is(err, tldr.ErrNotFoundPage) {
			if cfg.language != "" {
				awf.Clear().SetEmptyWarning(
					"Not found the command in selected language",
					"Try not to specify language option",
				).Output()
				return nil
			}
			if cfg.fuzzy {
				// list suggestions
				return cfg.printFuzzyPages(cmds)
			}
			awf.Output()
			return nil
		}
		return err
	}

	// Note descriptions has one line at least
	// see: https://github.com/tldr-pages/tldr/blob/master/contributing-guides/style-guide.md
	title := p.CmdDescriptions[0]
	subtitle := ""
	if len(p.CmdDescriptions) >= 2 {
		subtitle = p.CmdDescriptions[1]
	}
	awf.Append(
		alfred.NewItem().
			Title(title).
			Subtitle(subtitle).
			Valid(false).
			Icon(
				alfred.NewIcon().
					Path("description.png"),
			),
	)
	for _, cmd := range p.CmdExamples {
		awf.Append(
			alfred.NewItem().
				Title(cmd.Cmd).
				Subtitle(cmd.Description).
				Arg(cmd.Cmd),
		).Variable(nextActionKey, nextActionCopy)
	}

	awf.Output()
	return nil
}

func (cfg *config) printFuzzyPages(cmds []string) error {
	index, err := cfg.tldrClient.LoadIndexFile()
	if err != nil {
		return err
	}

	suggestions := index.Commands.Search(cmds)
	for _, cmd := range suggestions {
		complete := cmd.Name
		pt := choicePlatform(cmd.Platforms, cfg.platform)
		if pt != tldr.PlatformCommon && pt != defaultPlatform {
			complete = fmt.Sprintf("-%s %s %s",
				platformFlag,
				pt,
				cmd.Name,
			)
		}
		awf.Append(
			alfred.NewItem().
				Title(cmd.Name).
				Subtitle(fmt.Sprintf("Platforms: %s", fmt.Sprintf("%s", cmd.Platforms))).
				Valid(false).
				Autocomplete(complete).
				Icon(
					alfred.NewIcon().
						Path("candidate.png"),
				),
		)
	}

	awf.Output()
	return nil
}

func (cfg *config) printVersion(v, r string) (_ error) {
	title := fmt.Sprintf("alfred-tldr %v(%s)", v, r)
	awf.Append(
		alfred.NewItem().Title(title),
	).Output()
	return
}

func choicePlatform(pts []tldr.Platform, selected tldr.Platform) tldr.Platform {
	if len(pts) >= 2 {
		// if there are more than two platforms,
		// priority are follow
		// selected pt, common, others
		for _, pt := range pts {
			if pt == selected {
				return selected
			}
		}

		for _, pt := range pts {
			if pt == tldr.PlatformCommon {
				return tldr.PlatformCommon
			}
		}
	}

	return pts[0]
}
