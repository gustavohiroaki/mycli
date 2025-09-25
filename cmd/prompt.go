/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/spf13/cobra"
)

var (
	contextFile string
)

func init() {
	rootCmd.AddCommand(promptCmd)
	promptCmd.Flags().StringVarP(&contextFile, "context", "c", "", "Path to a file containing context to be used in the prompt")
}

// promptCmd represents the prompt command
var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Command to generate prompts",
	Long: `This is command to generate prompts easily.
	You can use this command to create various types of prompts for different scenarios.
	For example:
	- Generating creative writing prompts
	- Generate prompts using techniques
	
	Usage examples:
	`,
	Run: func(cmd *cobra.Command, args []string) {
		openai_key := os.Getenv("OPENAI_API_KEY")
		if openai_key == "" {
			fmt.Println("Please set the OPENAI_API_KEY environment variable.")
			return
		}

		answers := Interactive([]string{
			"O que você quer fazer com o prompt?",
		})

		client := openai.NewClient(option.WithAPIKey(openai_key))
		orientation := "Você é um Refinador de Prompts profissional. Sua tarefa é analisar o prompt do usuário, identificar ambiguidades, generalidades ou falta de contexto, e reescrevê-lo para torná-lo significativamente mais claro, específico e completo. O prompt aprimorado deve maximizar a precisão da resposta da IA. Não adicione comentários, apenas o prompt final e aprimorado. NUNCA dê sugestões ou comentários, apenas o prompt final."
		prompt := answers[0]
		if contextFile != "" {
			content, err := os.ReadFile(contextFile)
			if err != nil {
				fmt.Printf("Error reading context file: %v\n", err)
				return
			}
			prompt = fmt.Sprintf("%s\n\nContexto adicional:\n%s", prompt, string(content))
		}

		stream := client.Chat.Completions.NewStreaming(context.TODO(), openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(orientation),
				openai.UserMessage(prompt),
			},
			Model: openai.ChatModelGPT5Nano,
		})

		acc := openai.ChatCompletionAccumulator{}

		for stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)
			print(chunk.Choices[0].Delta.Content)
			if content, ok := acc.JustFinishedContent(); ok {
				clipboard.WriteAll(content)
			}
		}

		if err := stream.Err(); err != nil {
			fmt.Printf("Stream error: %v\n", err)
			return
		}

		content := acc.Choices[0].Message.Content
		clipboard.WriteAll(content)
		fmt.Println("\nThe final prompt has been copied to your clipboard.")
	},
}
