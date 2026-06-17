package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emp1re/gql-curl/internal/config"
	"github.com/emp1re/gql-curl/internal/parser"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate a shell completion script for gqc.

After installing the script in your shell, gqc can complete commands, flags,
configured schema names, and GraphQL query/mutation operation names from
graphql.curl.yaml.`,
	Example: `  source <(gqc completion bash)
  gqc completion zsh > "${fpath[1]}/_gqc"
  gqc completion fish > ~/.config/fish/completions/gqc.fish
  gqc completion powershell | Out-String | Invoke-Expression`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return newCommandHelpError(cmd, "expected exactly one shell, got %d", len(args))
		}

		switch args[0] {
		case "bash", "zsh", "fish", "powershell":
			return nil
		default:
			return newCommandHelpError(cmd, "unsupported shell %q", args[0])
		}
	},
	ValidArgsFunction: cobra.FixedCompletions([]string{
		"bash\tgenerate bash completion",
		"zsh\tgenerate zsh completion",
		"fish\tgenerate fish completion",
		"powershell\tgenerate PowerShell completion",
	}, cobra.ShellCompDirectiveNoFileComp),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateCompletionScript(args[0], cmd.OutOrStdout())
	},
}

var completionInstallCmd = &cobra.Command{
	Use:   "install [bash|zsh|fish|powershell]",
	Short: "Install shell completion for the current user",
	Long: `Install a shell completion script for gqc in the current user's standard
completion directory.

If the shell argument is omitted, gqc tries to detect it from SHELL.`,
	Example: `  gqc completion install
  gqc completion install bash
  gqc completion install zsh
  gqc completion install fish
  gqc completion install powershell`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return newCommandHelpError(cmd, "expected at most one shell, got %d", len(args))
		}
		if len(args) == 0 {
			return nil
		}
		if isSupportedCompletionShell(args[0]) {
			return nil
		}

		return newCommandHelpError(cmd, "unsupported shell %q", args[0])
	},
	ValidArgsFunction: cobra.FixedCompletions([]string{
		"bash\tinstall bash completion",
		"zsh\tinstall zsh completion",
		"fish\tinstall fish completion",
		"powershell\tinstall PowerShell completion",
	}, cobra.ShellCompDirectiveNoFileComp),
	RunE: func(cmd *cobra.Command, args []string) error {
		shellName := ""
		if len(args) > 0 {
			shellName = args[0]
		} else {
			detected, err := detectCompletionShell()
			if err != nil {
				return commandError(cmd, "%v", err)
			}
			shellName = detected
		}

		path, err := completionInstallPath(shellName)
		if err != nil {
			return commandError(cmd, "%v", err)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return commandError(cmd, "create completion directory failed: %v", err)
		}

		file, err := os.Create(path)
		if err != nil {
			return commandError(cmd, "create completion file failed: %v", err)
		}
		defer file.Close()

		if err := generateCompletionScript(shellName, file); err != nil {
			return commandError(cmd, "%v", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Installed %s completion to: %s\n%s\n", shellName, path, completionInstallNote(shellName, path))
		return nil
	},
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	completionCmd.AddCommand(completionInstallCmd)
	rootCmd.AddCommand(completionCmd)
}

func generateCompletionScript(shellName string, out io.Writer) error {
	switch shellName {
	case "bash":
		return rootCmd.GenBashCompletionV2(out, true)
	case "zsh":
		return rootCmd.GenZshCompletion(out)
	case "fish":
		return rootCmd.GenFishCompletion(out, true)
	case "powershell":
		return rootCmd.GenPowerShellCompletionWithDesc(out)
	default:
		return fmt.Errorf("unsupported shell %q", shellName)
	}
}

func isSupportedCompletionShell(shellName string) bool {
	switch shellName {
	case "bash", "zsh", "fish", "powershell":
		return true
	default:
		return false
	}
}

func detectCompletionShell() (string, error) {
	if shellPath := os.Getenv("SHELL"); shellPath != "" {
		shellName := filepath.Base(shellPath)
		if isSupportedCompletionShell(shellName) {
			return shellName, nil
		}
	}
	if os.Getenv("PSModulePath") != "" {
		return "powershell", nil
	}

	return "", fmt.Errorf("could not detect shell; pass bash, zsh, fish, or powershell")
}

func completionInstallPath(shellName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect home directory failed: %w", err)
	}

	switch shellName {
	case "bash":
		base := os.Getenv("BASH_COMPLETION_USER_DIR")
		if base == "" {
			base = filepath.Join(xdgDataHome(home), "bash-completion")
		}
		return filepath.Join(base, "completions", "gqc"), nil
	case "zsh":
		base := os.Getenv("ZSH_COMPLETION_DIR")
		if base == "" {
			base = filepath.Join(xdgDataHome(home), "zsh", "site-functions")
		}
		return filepath.Join(base, "_gqc"), nil
	case "fish":
		return filepath.Join(xdgConfigHome(home), "fish", "completions", "gqc.fish"), nil
	case "powershell":
		return filepath.Join(xdgConfigHome(home), "powershell", "Completions", "gqc.ps1"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shellName)
	}
}

func xdgDataHome(home string) string {
	if value := os.Getenv("XDG_DATA_HOME"); value != "" {
		return value
	}

	return filepath.Join(home, ".local", "share")
}

func xdgConfigHome(home string) string {
	if value := os.Getenv("XDG_CONFIG_HOME"); value != "" {
		return value
	}

	return filepath.Join(home, ".config")
}

func completionInstallNote(shellName, path string) string {
	switch shellName {
	case "zsh":
		return fmt.Sprintf("Restart your shell. If completions are not loaded, add %q to fpath before compinit.", filepath.Dir(path))
	case "powershell":
		return fmt.Sprintf("Load this file from your PowerShell profile, then restart PowerShell: %s", path)
	default:
		return "Restart your shell to use it."
	}
}

func registerGenerateCompletions(cmd *cobra.Command) {
	cmd.ValidArgsFunction = completeGenerateOperationNames
	registerSchemaFlagCompletion(cmd, "schema")

	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions([]string{
		"curl\tprint a ready-to-run curl command",
		"postman\tprint a raw GraphQL JSON payload",
		"payload\tprint a raw GraphQL JSON payload",
		"json\talias for payload",
		"playground\tprint GraphQL Playground query and variables blocks",
		"pg\talias for playground",
	}, cobra.ShellCompDirectiveNoFileComp))
	_ = cmd.RegisterFlagCompletionFunc("vars", cobra.NoFileCompletions)
	_ = cmd.RegisterFlagCompletionFunc("filter", cobra.NoFileCompletions)
}

func registerSchemaFlagCompletion(cmd *cobra.Command, flagName string) {
	_ = cmd.RegisterFlagCompletionFunc(flagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeSchemaNames(cmd, flagName, toComplete)
	})
}

func completeSchemaNames(cmd *cobra.Command, flagName, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.LoadConfig("graphql.curl.yaml")
	if err != nil {
		cobra.CompErrorln(fmt.Sprintf("load config error: %v", err))
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prefix, current := schemaCompletionPrefix(toComplete)
	selected, _ := cmd.Flags().GetStringSlice(flagName)
	selectedValues := append([]string(nil), selected...)
	if prefix != "" {
		selectedValues = append(selectedValues, strings.TrimSuffix(prefix, ","))
	}
	selectedNames := schemaNameSet(config.NormalizeSchemaNames(selectedValues))

	names := cfg.SchemaNames()
	completions := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := selectedNames[name]; ok {
			continue
		}

		completions = appendCompletion(completions, prefix+name, "schema from graphql.curl.yaml", prefix+current)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeGenerateOperationNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	schemaNames, err := cmd.Flags().GetStringSlice("schema")
	if err != nil {
		cobra.CompErrorln(fmt.Sprintf("read --schema flag failed: %v", err))
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	completions, err := generateOperationNameCompletions(schemaNames, toComplete)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func generateOperationNameCompletions(schemaNames []string, toComplete string) ([]string, error) {
	cfg, err := config.LoadConfig("graphql.curl.yaml")
	if err != nil {
		return nil, fmt.Errorf("load config error: %w", err)
	}

	schemas, err := cfg.SelectedSchemas(schemaNames...)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	operations := make(map[string]map[string]struct{})
	parseErrors := make([]string, 0)
	for _, schemaCfg := range schemas {
		gql, err := parser.NewParserFromPaths([]string(schemaCfg.Config.Path))
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("parse schema %q error: %v", schemaCfg.Name, err))
			continue
		}

		for _, op := range rootOperationDefinitions(gql.Schema) {
			if op.Def == nil {
				continue
			}

			for _, field := range op.Def.Fields {
				if operations[field.Name] == nil {
					operations[field.Name] = make(map[string]struct{})
				}
				operations[field.Name][fmt.Sprintf("%s in %s", op.OpType, schemaCfg.Name)] = struct{}{}
			}
		}
	}

	if len(operations) == 0 && len(parseErrors) > 0 {
		return nil, errors.New(strings.Join(parseErrors, "; "))
	}

	names := make([]string, 0, len(operations))
	for name := range operations {
		names = append(names, name)
	}
	sort.Strings(names)

	completions := make([]string, 0, len(names))
	for _, name := range names {
		descriptions := make([]string, 0, len(operations[name]))
		for description := range operations[name] {
			descriptions = append(descriptions, description)
		}
		sort.Strings(descriptions)

		completions = appendCompletion(completions, name, strings.Join(descriptions, ", "), toComplete)
	}

	return completions, nil
}

func schemaCompletionPrefix(toComplete string) (string, string) {
	if commaIndex := strings.LastIndex(toComplete, ","); commaIndex >= 0 {
		return toComplete[:commaIndex+1], strings.TrimSpace(toComplete[commaIndex+1:])
	}

	return "", toComplete
}

func schemaNameSet(names []string) map[string]struct{} {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		result[name] = struct{}{}
	}

	return result
}

func appendCompletion(completions []string, value, description, toComplete string) []string {
	if !strings.HasPrefix(value, toComplete) {
		return completions
	}
	if description == "" {
		return append(completions, value)
	}

	return append(completions, value+"\t"+description)
}
