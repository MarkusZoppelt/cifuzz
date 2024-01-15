//go:build !windows

package execute

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/internal/bundler/archive"
	"code-intelligence.com/cifuzz/internal/cmdutils"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/internal/testutil"
	"code-intelligence.com/cifuzz/pkg/dependencies"
)

func Test_getFuzzer(t *testing.T) {
	sampleMetadata := &archive.Metadata{
		Fuzzers: []*archive.Fuzzer{
			{
				Name: "a-fuzzer",
			},
		},
	}
	fuzzer, err := findFuzzer("a-fuzzer", sampleMetadata)
	require.NoError(t, err)
	require.Equal(t, "a-fuzzer", fuzzer.Name)

	fuzzer, err = findFuzzer("b-fuzzer", sampleMetadata)
	require.EqualErrorf(t, err, "fuzzer 'b-fuzzer' not found in a bundle metadata file", "error message mismatch")
}

func Test_getFuzzerName(t *testing.T) {
	type args struct {
		fuzzer *archive.Fuzzer
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "use fuzzer name",
			args: args{
				fuzzer: &archive.Fuzzer{
					Name:   "fuzzer-name",
					Target: "fuzzer-target",
				},
			},
			want: "fuzzer-name",
		},
		{
			name: "use fuzzer target",
			args: args{
				fuzzer: &archive.Fuzzer{
					Name:   "",
					Target: "fuzzer-target",
				},
			},
			want: "fuzzer-target",
		},
		{
			name: "use fuzzer target",
			args: args{
				fuzzer: &archive.Fuzzer{
					Target: "fuzzer-target",
				},
			},
			want: "fuzzer-target",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFuzzerName(tt.args.fuzzer); got != tt.want {
				t.Errorf("getFuzzerName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_findFuzzer(t *testing.T) {
	type args struct {
		nameToFind     string
		bundleMetadata *archive.Metadata
	}
	tests := []struct {
		name    string
		args    args
		want    *archive.Fuzzer
		wantErr bool
	}{
		{
			name: "find fuzzer by name",
			args: args{
				nameToFind: "a-fuzzer",
				bundleMetadata: &archive.Metadata{
					Fuzzers: []*archive.Fuzzer{
						{
							Name: "0-fuzzer",
						},
						{
							Name: "a-fuzzer",
						},
					},
				},
			},
			want: &archive.Fuzzer{
				Name: "a-fuzzer",
			},
			wantErr: false,
		},
		{
			name: "find fuzzer by target",
			args: args{
				nameToFind: "a-fuzzer",
				bundleMetadata: &archive.Metadata{
					Fuzzers: []*archive.Fuzzer{
						{
							Name: "0-fuzzer",
						},
						{
							Target: "a-fuzzer",
						},
					},
				},
			},
			want: &archive.Fuzzer{
				Target: "a-fuzzer",
			},
			wantErr: false,
		},
		{
			name: "return single fuzzer if name is empty",
			args: args{
				nameToFind: "",
				bundleMetadata: &archive.Metadata{
					Fuzzers: []*archive.Fuzzer{
						{
							Name: "0-fuzzer",
						},
					},
				},
			},
			want: &archive.Fuzzer{
				Name: "0-fuzzer",
			},
		},
		{
			name: "error if name is empty and there are multiple fuzzers",
			args: args{
				nameToFind: "",
				bundleMetadata: &archive.Metadata{
					Fuzzers: []*archive.Fuzzer{
						{
							Name: "0-fuzzer",
						},
						{
							Name: "1-fuzzer",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error out if fuzzer not found",
			args: args{
				nameToFind: "a-fuzzer",
				bundleMetadata: &archive.Metadata{
					Fuzzers: []*archive.Fuzzer{
						{
							Name: "0-fuzzer",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findFuzzer(tt.args.nameToFind, tt.args.bundleMetadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("findFuzzer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findFuzzer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStopSignalFile(t *testing.T) {
	dir := testutil.BootstrapExampleProjectForTest(t, "execute-stop-signal-test", config.BuildSystemCMake)

	// We don't care if this command fails, it should create the file in any case
	// nolint
	cmdutils.ExecuteCommand(t, New(), os.Stdin, "my_fuzz_test", "--stop-signal-file=test")
	assert.FileExists(t, filepath.Join(dir, "test"), "--stop-signal-file flag did not create the file 'cifuzz-execution-finished'on exit")
}

func TestLLVMSymbolizerMissing(t *testing.T) {
	dependencies.TestMockAllDeps(t)
	// let the llvm-symbolizer dep fail
	dependencies.OverwriteUninstalled(dependencies.GetDep(dependencies.LLVMSymbolizer))

	// Create an empty project directory and change working directory to it
	testDir := testutil.ChdirToTempDir(t, "execute-cmd-test")

	fuzzers := []*archive.Fuzzer{
		{
			Name:       "my_fuzz_test",
			Engine:     "LIBFUZZER",
			ProjectDir: testDir,
		},
	}
	createBundleYamlFile(t, testDir, fuzzers)

	_, stdErr, err := cmdutils.ExecuteCommand(t, New(), os.Stdin, "my_fuzz_test")
	require.Error(t, err)
	assert.Contains(t, stdErr, fmt.Sprintf(dependencies.MessageMissing, "llvm-symbolizer"))
}

func TestJavaMissing(t *testing.T) {
	dependencies.TestMockAllDeps(t)
	// let the java dep fail
	dependencies.OverwriteUninstalled(dependencies.GetDep(dependencies.Java))

	// Create an empty project directory and change working directory to it
	testDir := testutil.ChdirToTempDir(t, "execute-cmd-test")

	fuzzers := []*archive.Fuzzer{
		{
			Name:       "com.example.FuzzTestCase::myFuzzTest",
			Engine:     "JAVA_LIBFUZZER",
			ProjectDir: testDir,
		},
	}
	createBundleYamlFile(t, testDir, fuzzers)

	_, stdErr, err := cmdutils.ExecuteCommand(t, New(), os.Stdin, "com.example.FuzzTestCase::myFuzzTest")
	require.Error(t, err)
	assert.Contains(t, stdErr, fmt.Sprintf(dependencies.MessageMissing, "java"))
}

func createBundleYamlFile(t *testing.T, outputPath string, fuzzers []*archive.Fuzzer) {
	metadata := &archive.Metadata{
		Fuzzers: fuzzers,
	}
	metadataYamlContent, err := metadata.ToYaml()
	require.NoError(t, err)
	metadataYamlPath := filepath.Join(outputPath, archive.MetadataFileName)
	err = os.WriteFile(metadataYamlPath, metadataYamlContent, 0644)
	require.NoError(t, err)
}
