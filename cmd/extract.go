package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/neurosnap/sentences/english"
)

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("extract called")
		dirPath, _ = cmd.Flags().GetString("dir")
		performExtraction()
	},
}

var dirPath string

const maxLineWidth = 80

func init() {
	rootCmd.AddCommand(extractCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// extractCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	extractCmd.Flags().String("dir", "", "A help for dir")
	extractCmd.MarkFlagRequired("dir")
}

func performExtraction() {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".json" {
			outputFilename := path + ".org"

			if _, err := os.Stat(outputFilename); os.IsNotExist(err) {
				transcript, err := extractTranscript(path)
				if err != nil {
					fmt.Printf("Error extracting transcript from %s: %v\n", path, err)
					return nil
				}

				reformatted := reformatIntoParagraphs(transcript)

				err = writeFile(outputFilename, []byte(reformatted), 0o644)
				if err != nil {
					fmt.Printf("Error writing transcript to %s: %v\n", outputFilename, err)
				} else {
					fmt.Printf("Transcript written to %s\n", outputFilename)
				}
			}
		}

		return nil
	})
	if err != nil {
		fmt.Printf("Error walking through directory: %v\n", err)
	}
}

func extractTranscript(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var data struct {
		Results struct {
			Transcripts []struct {
				Transcript string `json:"transcript"`
			} `json:"transcripts"`
		} `json:"results"`
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return "", err
	}

	transcript := ""
	if len(data.Results.Transcripts) > 0 {
		transcript = data.Results.Transcripts[0].Transcript
	}

	return transcript, nil
}

func writeFile(filename string, data []byte, perm os.FileMode) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func wrapText(text string, lineWidth int) string {
	var wrappedText strings.Builder

	words := strings.Fields(text)
	currentLineLength := 0

	for _, word := range words {
		if currentLineLength+len(word)+1 > lineWidth { // +1 for the space
			wrappedText.WriteString("\n")
			currentLineLength = 0
		}

		if currentLineLength > 0 {
			wrappedText.WriteString(" ")
			currentLineLength++
		}

		wrappedText.WriteString(word)
		currentLineLength += len(word)
	}

	return wrappedText.String()
}

func reformatIntoParagraphs(text string) string {
	var builder strings.Builder

	sentenceTokenizer, err := english.NewSentenceTokenizer(nil)
	if err != nil {
		fmt.Println("Error creating sentence tokenizer:", err)
		return ""
	}

	sentences := sentenceTokenizer.Tokenize(text)

	for _, sentence := range sentences {
		wrappedText := wrapText(sentence.Text, maxLineWidth)
		builder.WriteString(wrappedText)
		builder.WriteString("\n\n")
	}

	return builder.String()
}
