/*  gobib - convert TeX to BibTeX
    Copyright (C) 2018 nbena

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package gobib

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// BibtexEntry is an interface that defines what an entry should
// expose
// type BibtexEntry interface {
// 	Authors() []string
// 	Title() string
// 	URL() string
// 	Year() int

// 	String() string
// }

// BibItem is the constant that represents '\bibitem'
const BibItem = "\\bibitem{"

// EndBibliography is the constant: '\end{thebibliography}'
const EndBibliography = "\\end{thebibliography}"

// ErrBibUnclosed is an error that is returned when reading from the
// bib file and EOF is reached without seeing \end{thebibliography}
var ErrBibUnclosed = errors.New("Missing \\end{thebibliography}")

// ErrBibEmpty is an error that is returned when reading from
// an empty bibliography
var ErrBibEmpty = errors.New("Empty bibliography")

// ErrSyntax is an error that is returned when a generic
// syntax error is encountered
var ErrSyntax = errors.New("Syntax error")

// BibtexEntry is an interface that defines that basic behaviour
// of a BibtexEntry: returning a key to be used as key, and a String()
// for encoding itself into a Bibtex format.
type BibtexEntry interface {
	Key() string

	// String returns the BibTeX entry.
	String() string

	// unclosedToString returns the BibTeX entry without closing
	// the last bracket.
	unclosedToString() string
}

// BasicBibtexEntry is a struct that wraps the basic info
// about an entry.
type BasicBibtexEntry struct {
	Authors []string
	Title   string
	Year    int
}

// URLBibtexEntry is a struct that extends BasicBibtexEntry adding
// an URL.
type URLBibtexEntry struct {
	BasicBibtexEntry
	URL string
}

// OnlineEntry represents an entry with the 'urldate' field
type OnlineEntry struct {
	URLBibtexEntry
	Visited *time.Time
}

// Key returns a key to be used as the first argument of a Bibtex entry.
func (b *BasicBibtexEntry) Key() string {
	return fmt.Sprintf("%s-%d-%s", b.Title, b.Year, b.Authors[0])
}

// AuthorsToString returns a Bibtex-authors string, by joining the authors
// using 'and' keyword.
func (b *BasicBibtexEntry) AuthorsToString() string {
	return strings.Join(b.Authors, " and ")
}

// String returns a Bibtex-representation of the entry.
func (b *BasicBibtexEntry) String() string {
	return b.unclosedToString() + "}"
}

func (b *BasicBibtexEntry) unclosedToString() string {
	return fmt.Sprintf("@article{%s\n"+
		"\ttitle = \"%s\", \n"+
		"\tauthors = \"%s\",\n"+
		"\tyear = \"%d\"\n",
		b.Key(),
		b.AuthorsToString(),
		b.Title,
		b.Year,
	)
}

func (b *URLBibtexEntry) unclosedToString() string {
	returned := b.BasicBibtexEntry.unclosedToString()
	returned += "\"" + b.URL + "\"\n}"
	return returned
}

func (b *URLBibtexEntry) String() string {
	return b.unclosedToString()
}

func (b *OnlineEntry) String() string {
	returned := b.URLBibtexEntry.unclosedToString()
	year, month, day := b.Visited.Date()
	returned += fmt.Sprintf("%d-%d-%d", year, month, day)
	return returned
}

// Config is the configuration for the converter
type Config struct {
	// where to read from
	Input io.Reader
	// where to write to
	Output io.Writer
	// DefaultYear is the year to use if not present.
	DefaultYear int
	// DefaultVisited is the default 'urldate' value to use
	DefaultVisited *time.Time
}

// Tex2BibConverter is the converter from plain TeX to BibTeX.
type Tex2BibConverter struct {
	reader        *bufio.Reader
	config        *Config
	stage1Channel chan []byte
	stage2Channel chan BibtexEntry
	errorChannel  chan error
}

// NewConverter returns a new converter to convert a plain TeX
// bibliography into a BibTeX one.
func NewConverter(c *Config) *Tex2BibConverter {
	return &Tex2BibConverter{
		reader:        bufio.NewReader(c.Input),
		config:        c,
		stage1Channel: make(chan []byte, 10),
		stage2Channel: make(chan BibtexEntry, 10),
		errorChannel:  make(chan error),
	}
}

// what is returned from divider func
type dividerResult struct {
	// key is the bibitem key if any
	// value is the non-parsed TeX entry
	key, value string
}

func (d *dividerResult) String() string {
	return fmt.Sprintf("Bib key: %s,\nValue: %s", d.key, d.value)
}

func keyFromLine(line string) (string, error) {
	if !strings.Contains(line, BibItem) {
		return "", ErrSyntax
	}

	endIndex := strings.LastIndex(line, "}")
	if endIndex == -1 {
		return "", ErrSyntax
	}

	startIndex := strings.Index(line, "{")
	if startIndex == -1 {
		return "", ErrSyntax
	}

	return line[startIndex+1 : endIndex], nil
}

// divider take a reader that contains a bibliography and it divides
// it into different string, each string is a bibitem that still
// need to be parsed.
// reader is the reader which items will be read from
// output is a channel which items will be written to
// errChan is a channel which errors will be written to
// When an error occurs, output channel is closed
func divider(reader *bufio.Reader, output chan<- dividerResult, errChan chan<- error) {

	// entries := list.New()
	var line []byte

	var key string
	// var value string

	var readLine string
	var err error

	bibitemFindLoop := true
	innerLoop := true

	var currentEntry strings.Builder
	var currentResult dividerResult

	// FIRST LOOP: till the first \bibitem
	for bibitemFindLoop {

		line, _, err = reader.ReadLine()

		if err != nil {
			if err == io.EOF {
				err = ErrBibEmpty
			}
			innerLoop = false
			bibitemFindLoop = false
			errChan <- err
			close(output)
		}

		readLine = string(line)

		if strings.Contains(readLine, BibItem) {
			bibitemFindLoop = false
			key, _ = keyFromLine(readLine)
			currentResult.key = key
		}
	}

	// SECOND LOOP: till the end of the file
	for innerLoop {

		line, _, err = reader.ReadLine()
		readLine = string(line)

		if err != nil {
			// OLD VERSION: treat io.EOF as a non-error but it's wrong because

			// if there's an error we exit from the loop
			if err == io.EOF {
				err = ErrBibUnclosed
			}
			errChan <- err
			innerLoop = false
			// entries.PushBack(currentEntry.String())
			currentResult.value = currentEntry.String()
			// output <- currentEntry.String()
			output <- currentResult
			close(output)
		}

		if strings.Contains(readLine, BibItem) {
			// we're at the end of this bibitem
			// we push the current item to the list
			// and we reset the Builder for holding the next entry
			// entries.PushBack(currentEntry.String())
			currentResult.value = currentEntry.String()
			// output <- currentEntry.String()
			output <- currentResult
			currentEntry.Reset()

			// now reading the key
			key, _ = keyFromLine(readLine)
			currentResult.key = key
		} else if strings.Contains(readLine, EndBibliography) {
			// the bibliography is finished
			innerLoop = false
			// entries.PushBack(currentEntry.String())
			currentResult.value = currentEntry.String()
			// output <- currentEntry.String()
			output <- currentResult
		} else {
			// if here, it's just another line of our entry
			// we trim spaces and we write it to the Builder
			readLine = strings.TrimSpace(readLine)
			if len(readLine) > 0 {
				currentEntry.WriteString(readLine)
			}
		}
	}
}

// tokenizer takes an input chan in which \bibitem are
// and converts them to a BibTextEntry.
func tokenizer(input <-chan string, output chan<- BibtexEntry) {

}
