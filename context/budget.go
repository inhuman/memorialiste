package context

// ApproxTokens estimates the token count of s using the formula len(s)/4.
func ApproxTokens(s string) int {
	return len(s) / 4
}
