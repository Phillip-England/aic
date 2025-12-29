package llmactions

const CmdEnd = "AIC: END;"

// IsEnd checks if a line strictly matches the end command
func IsEnd(line string) bool {
	return line == CmdEnd
}