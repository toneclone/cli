package cmd

func wantsJSONFormat(commandFormat string) bool {
	return jsonOutput || commandFormat == "json"
}
