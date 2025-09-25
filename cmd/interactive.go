package cmd

import (
	"bufio"
	"fmt"
	"os"
)

func Interactive(items []string) []string {
	var answers []string
	reader := bufio.NewReader(os.Stdin)
	for _, item := range items {
		fmt.Print(item + " ")
		answer, _ := reader.ReadString('\n')
		answers = append(answers, answer)
	}
	fmt.Println("-------------------------------- Answer --------------------------------")
	return answers
}
