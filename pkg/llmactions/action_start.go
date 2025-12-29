package llmactions

const CmdStart = "AIC: START;"

// IsStart checks if a line strictly matches the start command
func IsStart(line string) bool {
	return line == CmdStart
}