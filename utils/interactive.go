package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Interactive(items []string) []string {
	var answers []string
	reader := bufio.NewReader(os.Stdin)
	for _, item := range items {
		fmt.Print(item + " ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			// On read error, append an empty answer and continue
			answers = append(answers, "")
			continue
		}
		answers = append(answers, strings.TrimRight(answer, "\r\n"))
	}
	fmt.Println("-------------------------------- Answer --------------------------------")
	return answers
}
