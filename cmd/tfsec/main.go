package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/liamg/tml"

	"github.com/liamg/clinch/terminal"
	_ "github.com/liamg/tfsec/internal/app/tfsec/checks"
	"github.com/liamg/tfsec/internal/app/tfsec/parser"
	"github.com/liamg/tfsec/internal/app/tfsec/scanner"
	"github.com/liamg/tfsec/version"
	"github.com/spf13/cobra"
)

var showVersion = false
var disableColours = false

func init() {
	rootCmd.Flags().BoolVar(&disableColours, "no-colour", disableColours, "Disable coloured output")
	rootCmd.Flags().BoolVar(&disableColours, "no-color", disableColours, "Disable colored output (American style!)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", showVersion, "Show version information and exit")
	rootCmd.Flags().String("format", " ", "Specify format of the output. (Available value is json)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "tfsec [directory]",
	Short: "tfsec is a terraform security scanner",
	Long:  `tfsec is a simple tool to detect potential security vulnerabilities in your terraformed infrastructure.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		if disableColours {
			tml.DisableFormatting()
		}

		if showVersion {
			fmt.Println(version.Version)
			os.Exit(0)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		var dir string
		var err error
		format, _ := cmd.Flags().GetString("format")
		if len(args) == 1 {
			dir, err = filepath.Abs(args[0])
		} else {
			dir, err = os.Getwd()
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		blocks, err := parser.New().ParseDirectory(dir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		results := scanner.New().Scan(blocks)
		if format == "json" {
			// Even if there are no issues, encode results to json.
			if results == nil {
				// nil slice encodes as the `null` JSON value.
				// This happens when no terraform files are found. ( Their design. Duh!)
				// we want the value to be just []
				// so, we will explicitely set results to an empty slice of `scanner.Result`
				results = []scanner.Result{}
			}
			jsonResults, err := json.Marshal(results)
			if err != nil {
				log.Println(err)
				os.Exit(1)
			}
			// display the json results at stdout
			fmt.Println(string(jsonResults))
			// Exit 0 as there were no exceptions while running the analyzer.
			os.Exit(0)
		}
		if len(results) == 0 {
			terminal.PrintSuccessf("\nNo problems detected!\n")
			os.Exit(0)
		}

		terminal.PrintErrorf("\n%d potential problems detected:\n\n", len(results))
		for i, result := range results {
			terminal.PrintErrorf("<underline>Problem %d</underline>\n", i+1)
			_ = tml.Printf(`
  <blue>[</blue>%s<blue>]</blue> %s
  <blue>%s</blue>

`, result.Code, result.Description, result.Range.String())
			highlightCode(result)
		}

		os.Exit(1)
	},
}

// highlight the lines of code which caused a problem, if available
func highlightCode(result scanner.Result) {

	data, err := ioutil.ReadFile(result.Range.Filename)
	if err != nil {
		return
	}

	lines := append([]string{""}, strings.Split(string(data), "\n")...)

	start := result.Range.StartLine - 3
	if start <= 0 {
		start = 1
	}
	end := result.Range.EndLine + 3
	if end >= len(lines) {
		end = len(lines) - 1
	}

	for lineNo := start; lineNo <= end; lineNo++ {
		_ = tml.Printf("  <blue>% 6d</blue> | ", lineNo)
		if lineNo >= result.Range.StartLine && lineNo <= result.Range.EndLine {
			if lineNo == result.Range.StartLine && result.RangeAnnotation != "" {
				_ = tml.Printf("<bold><red>%s</red>    <blue>%s</blue></bold>\n", lines[lineNo], result.RangeAnnotation)
			} else {
				_ = tml.Printf("<bold><red>%s</red></bold>\n", lines[lineNo])
			}
		} else {
			_ = tml.Printf("<yellow>%s</yellow>\n", lines[lineNo])
		}
	}

	fmt.Println("")

}
