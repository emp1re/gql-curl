package cmd

func init() {
	generateCmd.Flags().StringVarP(&varsStr, "vars", "v", "", "JSON raw with variables (exam. '{\"id\": 1}')")
	generateCmd.Flags().StringVarP(&varsFile, "var-file", "f", "", "Path to a JSON file containing variables (exam. './vars.json')")
	generateCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactively fill in variable")
	// Expose the --run flag to allow users to execute the generated query directly against the endpoint
	generateCmd.Flags().BoolVarP(&run, "run", "r", false, "Connect to the endpoint and execute the generated query, printing the response")
	generateCmd.Flags().StringVarP(&filterStr, "filter", "q", "", "Path to filter the response using gjson syntax (e.g. 'data.user.name') - works only with --run flag")
	generateCmd.Flags().StringVarP(&genSchema, "schema", "s", "", "Schema name from config.schemas to use (default: all)")
	generateCmd.Flags().StringVarP(&genFormat, "format", "o", "curl", "Output format: curl, payload/json/postman, or playground")
	rootCmd.AddCommand(generateCmd)
}
