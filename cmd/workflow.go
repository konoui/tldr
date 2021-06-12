package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/konoui/go-alfred"
)

// decide next action for workflow filter
const (
	nextActionKey   = "nextAction"
	nextActionCopy  = "copy"
	nextActionShell = "shell"
	// Note the key is also defined in workflow environment variable
	updateDBRecommendationEnvKey       = "TLDR_DB_UPDATE_RECOMMENDATION"
	updateWorkflowRecommendationEnvKey = "TLDR_WORKFLOW_UPDATE_RECOMMENDATION"
	updateWorkflowIntervalDays         = "TLDR_WORKFLOW_UPDATE_INTERVAL_DAYS"
)

var awf *alfred.Workflow

func init() {
	interval, err := getUpdateWorkflowInterval()
	if err != nil {
		interval = twoWeeks
	}
	defer func() {
		if err != nil {
			awf.Logger().Warnln(err.Error())
		}
	}()
	awf = alfred.NewWorkflow(
		alfred.WithMaxResults(30),
		alfred.WithGitHubUpdater(
			"konoui",
			"alfred-tldr",
			version,
			interval,
		),
	)
	awf.SetOut(outStream)
	awf.SetLog(errStream)
	if err := awf.OnInitialize(); err != nil {
		awf.Fatal(err.Error(), err.Error())
	}
}

func isUpdateDBRecommendEnabled() bool {
	return parseBool(updateDBRecommendationEnvKey)
}

func isUpdateWorkflowRecommendEnabled() bool {
	return parseBool(updateWorkflowRecommendationEnvKey)
}

func getUpdateWorkflowInterval() (time.Duration, error) {
	v := os.Getenv(updateWorkflowIntervalDays)
	fv, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse update interval env: %w", err)
	}
	tv := time.Duration(fv) * 24 * time.Hour
	return tv, err
}

func parseBool(key string) bool {
	sv := os.Getenv(key)
	bv, err := strconv.ParseBool(sv)
	if err != nil {
		return false
	}
	return bv
}

func fatal(err error) {
	awf.Fatal("a fatal error occurred", fmt.Sprint(err))
}
