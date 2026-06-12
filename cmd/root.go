package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "gqc",
	Short:         "Generate and run GraphQL requests from a local schema",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `gqc reads the GraphQL schema configured in graphql.curl.yaml and generates
ready-to-copy requests for curl, Postman, and GraphQL Playground.

It can also execute generated operations directly against the configured endpoint.`,
	Example: `  gqc generate
  gqc generate getUser --schema main
  gqc generate createUser --format playground
  gqc generate getUser --format postman --vars '{"id":"123"}'
  gqc generate getUser --run --filter 'data.getUser.name'
  gqc fetch --schema main`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return newCommandHelpError(cmd, "%s", err.Error())
	})
}

type commandHelpError struct {
	cmd *cobra.Command
	err error
}

func (e *commandHelpError) Error() string {
	return e.err.Error()
}

func (e *commandHelpError) Unwrap() error {
	return e.err
}

func newCommandHelpError(cmd *cobra.Command, format string, args ...interface{}) error {
	return &commandHelpError{
		cmd: cmd,
		err: fmt.Errorf(format, args...),
	}
}

func noArgsWithHelp(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return newCommandHelpError(cmd, "unknown argument %q", args[0])
	}

	return nil
}

func maximumNArgsWithHelp(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			return newCommandHelpError(cmd, "expected at most %d argument(s), got %d", n, len(args))
		}

		return nil
	}
}

func commandError(cmd *cobra.Command, format string, args ...interface{}) error {
	return newCommandHelpError(cmd, format, args...)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		printErrorWithHelp(err)
		os.Exit(1)
	}
}

func printErrorWithHelp(err error) {
	var helpErr *commandHelpError
	if errors.As(err, &helpErr) && helpErr.cmd != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", helpErr.err)
		printCommandHelp(os.Stderr, helpErr.cmd)
		return
	}

	fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
	cmd, _, findErr := rootCmd.Find(os.Args[1:])
	if findErr != nil || cmd == nil {
		cmd = rootCmd
	}
	printCommandHelp(os.Stderr, cmd)
}

func printCommandHelp(out *os.File, cmd *cobra.Command) {
	cmd.SetOut(out)
	cmd.SetErr(out)
	_ = cmd.Help()
}
