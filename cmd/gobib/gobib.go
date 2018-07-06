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
	flag.StringVar(&input, "in", "", "the input file")
	flag.StringVar(&output, "out", "", "the output file")
	flag.IntVar(&year, "default-year", 0, "the default year value to use when a year is not found")
	flag.StringVar(&defaultVisited, "default-urldate", "", "the default urldate value to use, in a form YYYY-MM-DD")
	flag.BoolVar(&printFinished, "print-finished", false, "print a message when conversion is finished")

	flag.Parse()
}

func main() {

	setFlags()
	var err error

	var finalDefaultVisited *time.Time

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

	inputFile, err := os.Open(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file %s: %s", input, err.Error())
		os.Exit(-1)
	}
	var outputFile *os.File
	if output == "/dev/stout" || output == "" {
		outputFile = os.Stdout
	} else {
		outputFile, err = os.Open(output)
		if err != nil {
			if os.IsNotExist(err) {
				outputFile, err = os.Create(output)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Fail to create file: %s, %s", output, err.Error())
					os.Exit(-1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Error opening file %s: %s", output, err.Error())
				os.Exit(-1)
			}
		}
	}

	in := bufio.NewReader(inputFile)
	out := bufio.NewWriter(outputFile)
	// var out strings.Builder

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
			fmt.Fprintf(os.Stdout, "Conversion finished")
		}
	case err = <-errChan:
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		exit = 1
	}
	os.Exit(exit)
}
