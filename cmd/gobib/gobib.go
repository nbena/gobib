package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nbena/gobib/pkg/gobib"
)

var (
	input          string
	output         string
	year           int
	defaultVisited string
	visited        time.Time
	printFinished  bool
)

func setFlags() {
	flag.StringVar(&input, "in", os.Stdin.Name(), "the input file")
	flag.StringVar(&output, "out", os.Stdout.Name(), "the output file")
	flag.IntVar(&year, "default-year", gobib.NoDefaultYear, "the default year value to use when a year is not found")
	flag.StringVar(&defaultVisited, "default-urldate", "", "the default urldate value to use, the format is YYYY-MM-DD")
	flag.BoolVar(&printFinished, "print-finished", false, "print a message when conversion is finished")

	flag.Parse()
}

func main() {

	setFlags()
	var err error

	var finalDefaultVisited = gobib.NoDefaultURLDate

	if defaultVisited != "" {
		visited, err = time.Parse("2006-01-02", defaultVisited)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error in 'default-urldate' format")
			os.Exit(-1)
		}
		finalDefaultVisited = &visited
	} else {
		finalDefaultVisited = nil
	}

	var inputFile, outputFile *os.File
	if input != os.Stdin.Name() {
		inputFile, err = os.Open(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %s: %s", input, err.Error())
			os.Exit(-1)
		}
	} else {
		inputFile = os.Stdin
	}

	if output != os.Stdout.Name() {
		outputFile, err = os.OpenFile(output, os.O_WRONLY, 0755)
		if err != nil {
			if os.IsNotExist(err) {
				outputFile, err = os.Create(output)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Fail to create file: %s, %s", output, err.Error())
					inputFile.Close()
					os.Exit(-1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Error opening file %s: %s", output, err.Error())
				inputFile.Close()
				os.Exit(-1)
			}
		}
	} else {
		outputFile = os.Stdout
	}

	in := bufio.NewReader(inputFile)
	out := bufio.NewWriter(outputFile)

	config := &gobib.Config{
		Input:          in,
		Output:         out,
		DefaultYear:    year,
		DefaultVisited: finalDefaultVisited,
	}

	converter := gobib.NewConverter(config)
	converter.Convert()
	okChan, errChan := converter.OkChan(), converter.ErrChan()
	exit := 0
	select {
	case <-okChan:
		if printFinished {
			fmt.Fprintf(os.Stdout, "Conversion finished\n")
		}
	case err = <-errChan:
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		exit = 1
	}
	// closing files and goobye
	if err = out.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error in flushing: %s\n", err.Error())
	}
	inputFile.Close()
	outputFile.Close()
	os.Exit(exit)
}
