package cmd

// orDash renders empty strings as "-" so table cells never look truncated.
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
