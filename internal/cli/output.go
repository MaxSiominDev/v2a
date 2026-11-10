package cli

import (
	"fmt"
	"strings"
)

func header(text string) {
	fmt.Println(text)
	fmt.Println()
}

func report(input, output string, notes []string) {
	fmt.Printf("Input:  %s\n", input)
	fmt.Printf("Output: %s\n", output)
	for _, note := range notes {
		fmt.Printf("Note: %s\n", note)
	}
	fmt.Println()
	fmt.Println("Done.")
}

func indentBlock(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}
