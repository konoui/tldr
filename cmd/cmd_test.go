package cmd

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/konoui/alfred-tldr/pkg/tldr"
	"github.com/konoui/go-alfred"
	"github.com/konoui/go-alfred/update"
	mock "github.com/konoui/go-alfred/update/mock_update"
	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
)

func testdataPath(file string) string {
	return filepath.Join("testdata", file)
}

func setup(t *testing.T, awf *alfred.Workflow, command string) (outBuf, errBuf *bytes.Buffer, cmd *cobra.Command) {
	t.Helper()

	outBuf, errBuf = new(bytes.Buffer), new(bytes.Buffer)
	outStream = outBuf
	errStream = outBuf
	awf.SetOut(outBuf)
	awf.SetLog(errBuf)
	cmd = NewRootCmd()
	cmdArgs, err := shellwords.Parse(command)
	if err != nil {
		t.Fatalf("args parse error: %+v", err)
	}
	cmd.SetOutput(outBuf)
	cmd.SetArgs(cmdArgs)

	return
}

func execute(t *testing.T, rootCmd *cobra.Command) {
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error got: %+v", err)
	}
}

func TestExecute(t *testing.T) {
	type args struct {
		filepath string
		command  string
	}
	tests := []struct {
		name   string
		args   args
		update bool
	}{
		{
			name: "lsof",
			args: args{
				command:  "lsof",
				filepath: testdataPath("test_output_lsof.json"),
			},
		},
		{
			name: "sub command git checkout",
			args: args{
				command:  "git checkout",
				filepath: testdataPath("test_output_git-checkout.json"),
			},
		},
		{
			name: "fuzzy search returns git checkout",
			args: args{
				command:  "gitchec --fuzzy",
				filepath: testdataPath("test_output_git-checkout_with_fuzzy.json"),
			},
		},
		{
			name: "fuzzy search returns non-common platform",
			args: args{
				command:  "pstree --fuzzy",
				filepath: testdataPath("test_output_pstree_with_fuzzy.json"),
			},
		},
		{
			name: "multiple platform command yank without -p flag returns computer platform OSX",
			args: args{
				command:  "yan --fuzzy",
				filepath: testdataPath("test_output_yank_without_p-flag_fuzzy.json"),
			},
		},
		{
			name: "multiple platform command yank with -p flag returns specified platform",
			args: args{
				command:  "yan -p linux --fuzzy",
				filepath: testdataPath("test_output_yank_with_p-flag_with_fuzzy.json"),
			},
		},
		{
			name: "outputs no error when cache expired",
			args: args{
				command:  "lsof",
				filepath: testdataPath("test_output_lsof.json"),
			},
		},
		{
			name: "version flag is the highest priority",
			args: args{
				command:  "-v -u tldr -L -a",
				filepath: testdataPath("test_output_version.json"),
			},
		},
		{
			name: "prints update confirmation when specified --update flag and ignore argument",
			args: args{
				command:  "--update tldr",
				filepath: testdataPath("test_output_update-confirmation.json"),
			},
		},
		{
			name: "specify language flag but commands not found",
			args: args{
				command:  "-L invalid-lang tar",
				filepath: testdataPath("test_output_language-empty-result.json"),
			},
		},
		{
			name: "input something but commands not found",
			args: args{
				command:  "dummy-empty-result",
				filepath: testdataPath("test_output_empty-result.json"),
			},
		},
		{
			name: "fuzzy search but commands not found",
			args: args{
				command:  "--fuzzy dummy-empty-result",
				filepath: testdataPath("test_output_empty-result.json"),
			},
		},
		{
			name: "no input returns no-input message",
			args: args{
				command:  "",
				filepath: testdataPath("test_output_no-input.json"),
			},
		},
		{
			name: "specify help flag returns usage",
			args: args{
				command:  "--help",
				filepath: testdataPath("test_output_usage.json"),
			},
		},
		{
			name: "string flag without any value and invalid flag return no-input message",
			args: args{
				command:  "-L -a",
				filepath: testdataPath("test_output_no-input.json"),
			},
		},
		{
			name: "string flag without any value returns usage",
			args: args{
				command:  "--fuzzy -L",
				filepath: testdataPath("test_output_usage.json"),
			},
		},
		{
			name: "invalid bool flag returns usage",
			args: args{
				command:  "lsof -a",
				filepath: testdataPath("test_output_usage.json"),
			},
		},
		{
			name: " bool invalid flag and valid flag return usage",
			args: args{
				command:  "-a -u",
				filepath: testdataPath("test_output_usage.json"),
			},
		},
		{
			name: "invalid platform flag returns platform error message",
			args: args{
				command:  "-p a",
				filepath: testdataPath("test_output_platform-error.json"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantData, err := ioutil.ReadFile(tt.args.filepath)
			if err != nil {
				t.Fatal(err)
			}

			// overwrite global awf
			awf = alfred.NewWorkflow()
			outBuf, _, cmd := setup(t, awf, tt.args.command)
			execute(t, cmd)
			outGotData := outBuf.Bytes()

			// automatically update test data
			if tt.update {
				if err := writeFile(tt.args.filepath, outGotData); err != nil {
					t.Fatal(err)
				}
			}

			if diff := alfred.DiffOutput(wantData, outGotData); diff != "" {
				t.Errorf("-want +got\n%+v", diff)
			}
		})
	}
}

func TestUpdateConfirmation(t *testing.T) {
	type args struct {
		filepath              string
		command               string
		dbTTL                 time.Duration
		newerVersionAvailable bool
	}
	tests := []struct {
		name   string
		args   args
		update bool
	}{
		{
			name: "no input and update recommendations",
			args: args{
				dbTTL:                 0,
				newerVersionAvailable: true,
				command:               "",
				filepath:              testdataPath("output_update-recommendations.json"),
			},
		},
		{
			name: "lsof with workflow update recommendation",
			args: args{
				dbTTL:                 1000 * time.Hour,
				newerVersionAvailable: true,
				command:               "lsof",
				filepath:              testdataPath("output_lsof-with-update-workflow-recommendation.json"),
			},
		},
		{
			name: "lsof with db recommendation",
			args: args{
				dbTTL:                 0,
				newerVersionAvailable: false,
				command:               "lsof",
				filepath:              testdataPath("output_lsof-with-update-db-recommendation.json"),
			},
		},
		{
			name: "update db confirmation",
			args: args{
				dbTTL:                 0,
				newerVersionAvailable: false,
				command:               "--update",
				filepath:              testdataPath("output_update-db-confirmation.json"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.Setenv(updateDBRecommendationEnvKey, "true"); err != nil {
				t.Fatal(err)
			}
			if err := os.Setenv(updateWorkflowRecommendationEnvKey, "true"); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Unsetenv(updateDBRecommendationEnvKey); err != nil {
					t.Fatal(err)
				}
				if err := os.Unsetenv(updateWorkflowRecommendationEnvKey); err != nil {
					t.Fatal(err)
				}
			}()

			wantData, err := ioutil.ReadFile(tt.args.filepath)
			if err != nil {
				t.Fatal(err)
			}

			mockSource, teardown := newMockUpdaterSource(t, tt.args.newerVersionAvailable)
			defer teardown()

			// disable ttl
			twoWeeks = tt.args.dbTTL
			awf = alfred.NewWorkflow(
				alfred.WithUpdater(mockSource, ""),
			)

			outBuf, errBuf, cmd := setup(t, awf, tt.args.command)
			execute(t, cmd)
			outGotData := outBuf.Bytes()

			// automatically update test data
			if tt.update {
				if err := writeFile(tt.args.filepath, outGotData); err != nil {
					t.Fatal(err)
				}
			}

			if diff := alfred.DiffOutput(wantData, outGotData); diff != "" {
				t.Errorf("-want +got\n%+v\n", diff)
				t.Errorf("error output: %s", errBuf.String())
			}
		})
	}
}

func newMockUpdaterSource(t *testing.T, newerVersionAvailable bool) (_ update.UpdaterSource, _ func()) {
	ctrl := gomock.NewController(t)
	mockSource := mock.NewMockUpdaterSource(ctrl)
	mockUpdater := mock.NewMockUpdater(ctrl)
	mockUpdater.EXPECT().Update(gomock.Any()).Return(nil).AnyTimes()
	mockSource.EXPECT().NewerVersionAvailable(gomock.Any()).Return(newerVersionAvailable, nil).AnyTimes()
	mockSource.EXPECT().IfNewerVersionAvailable().Return(mockUpdater).AnyTimes()
	return mockSource, ctrl.Finish
}

func TestUpdateExecution(t *testing.T) {
	type args struct {
		command string
	}
	tests := []struct {
		name        string
		args        args
		expectedErr bool
		errMsg      string
		wantMsg     string
	}{
		{
			name: "update db returns succeeded message",
			args: args{
				command: "--update --confirm",
			},
			wantMsg:     "update succeeded",
			expectedErr: false,
		},
		{
			name: "update-workflow confirmation does not support",
			args: args{
				command: "--update-workflow",
			},
			expectedErr: true,
			errMsg:      "direct update via flag is not supported",
		},
		{
			name: "when update-workflow without updater, nil updater returns error. update execution outputs message to stdout",
			args: args{
				command: "--update-workflow --confirm",
			},
			expectedErr: false,
			wantMsg:     "update failed due to no implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			awf = alfred.NewWorkflow()
			outBuf, _, cmd := setup(t, awf, tt.args.command)
			err := cmd.Execute()
			if tt.expectedErr && err == nil {
				t.Errorf("unexpected results")
			}
			if tt.expectedErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("want: %v\n got: %v", tt.errMsg, err.Error())
				}
			}
			got := outBuf.String()
			if got != tt.wantMsg {
				t.Errorf("want: %v\n got: %v", tt.wantMsg, got)
			}
		})
	}
}

func writeFile(filename string, data []byte) error {
	pretty := new(bytes.Buffer)
	if err := json.Indent(pretty, data, "", "  "); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, pretty.Bytes(), 0600); err != nil {
		return err
	}
	return nil
}

func Test_choicePlatform(t *testing.T) {
	type args struct {
		pts      []tldr.Platform
		selected tldr.Platform
	}
	tests := []struct {
		name string
		args args
		want tldr.Platform
	}{
		{
			name: "return common when platforms do not contain selected",
			args: args{
				pts: []tldr.Platform{
					tldr.PlatformCommon,
				},
				selected: tldr.PlatformLinux,
			},
			want: tldr.PlatformCommon,
		},
		{
			name: "return selected insted of common when platforms contain selected",
			args: args{
				pts: []tldr.Platform{
					tldr.PlatformCommon,
					tldr.PlatformLinux,
				},
				selected: tldr.PlatformLinux,
			},
			want: tldr.PlatformLinux,
		},
		{
			name: "returns common when platforms do not contain selected",
			args: args{
				pts: []tldr.Platform{
					tldr.PlatformCommon,
					tldr.PlatformLinux,
				},
				selected: tldr.PlatformOSX,
			},
			want: tldr.PlatformCommon,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := choicePlatform(tt.args.pts, tt.args.selected); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("choicePlatform() = %v, want %v", got, tt.want)
			}
		})
	}
}
