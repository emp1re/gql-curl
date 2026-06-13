package cmd

func init() {
	generateCmd.Flags().StringVarP(&varsStr, "vars", "v", "", "Inline GraphQL variables JSON, for example '{\"id\":\"123\"}'")
	generateCmd.Flags().StringVarP(&varsFile, "var-file", "f", "", "Read GraphQL variables from a JSON file")
	generateCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Prompt for operation variables in the terminal")
	// Expose the --run flag to allow users to execute the generated query directly against the endpoint
	generateCmd.Flags().BoolVarP(&run, "run", "r", false, "Execute the generated request against the configured endpoint")
	generateCmd.Flags().StringVarP(&filterStr, "filter", "q", "", "Filter --run JSON response using gjson syntax, for example 'data.user.name'")
	generateCmd.Flags().StringVarP(&genSchema, "schema", "s", "", "Use one schema from config.schemas instead of all schemas")
	generateCmd.Flags().StringVarP(&genFormat, "format", "o", "curl", "Output format: curl, postman/json/payload, or playground")
	registerGenerateCompletions(generateCmd)
	rootCmd.AddCommand(generateCmd)
}
