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

// URLToken is the constanr: '\url{'
const URLToken = "\\url{"

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
	GenKey() string

	// String returns the BibTeX entry.
	String() string

	// unclosedToString returns the BibTeX entry without closing
	// the last bracket.
	unclosedToString() string
}

// BasicOnlineBibtexEntry is a struct that wraps the basic info
// about an entry. It is a 'base struct'
type BasicOnlineBibtexEntry struct {
	Key     string
	Authors []string
	Title   string
	Year    int
	URL     string
}

// AdvancedOnlineBibtexEntry represents an entry with the 'urldate' field
type AdvancedOnlineBibtexEntry struct {
	BasicOnlineBibtexEntry
	Visited *time.Time
}

// NewBasicEntry returns a new BasicOnlineBibtexEntry.
func NewBasicEntry(key string, authors []string, title string, year int, URL string) *BasicOnlineBibtexEntry {
	entry := &BasicOnlineBibtexEntry{
		Key:     key,
		Authors: authors,
		Title:   title,
		Year:    year,
		URL:     URL,
	}

	if entry.Key == "" {
		entry.Key = entry.GenKey()
	}
	return entry
}

// GenKey generates, sets, returns a new key for this entry.
func (b *BasicOnlineBibtexEntry) GenKey() string {
	key := fmt.Sprintf("%s-%d-%s", b.Title, b.Year, b.Authors[0])
	b.Key = key
	return key
}

// AuthorsToString returns a Bibtex-authors string, by joining the authors
// using 'and' keyword.
func (b *BasicOnlineBibtexEntry) AuthorsToString() string {
	return strings.Join(b.Authors, " and ")
}

// String returns a Bibtex-representation of the entry.
func (b *BasicOnlineBibtexEntry) String() string {
	return b.unclosedToString() + "}"
}

func (b *BasicOnlineBibtexEntry) unclosedToString() string {
	result := fmt.Sprintf("@online{%s,\n"+
		"\tauthor = \"%s\",\n"+
		"\ttitle = \"%s\",\n"+
		"\tyear = \"%d\",\n",
		b.Key,
		b.AuthorsToString(),
		b.Title,
		b.Year,
	)
	if b.URL != "" {
		result += "\turl = \"" + b.URL + "\",\n"
	}
	return result
}

func (b *AdvancedOnlineBibtexEntry) unclosedToString() string {
	result := b.BasicOnlineBibtexEntry.unclosedToString()
	if b.Visited != nil {
		year, month, day := b.Visited.Date()
		result += fmt.Sprintf("\turldate = \"%d-%d-%d\",\n", year, month, day)
	}
	return result
}

func (b *AdvancedOnlineBibtexEntry) String() string {
	return b.unclosedToString() + "}"
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
	// if it's nil it won't be set
	DefaultVisited *time.Time
}

// Tex2BibConverter is the converter from plain TeX to BibTeX.
type Tex2BibConverter struct {
	reader           *bufio.Reader
	config           *Config
	stage1OutChannel chan dividerResult
	stage2OutChannel chan BibtexEntry
	errorChannel     chan error
	okChannel        chan struct{}
}

// NewConverter returns a new converter to convert a plain TeX
// bibliography into a BibTeX one.
func NewConverter(c *Config) *Tex2BibConverter {
	return &Tex2BibConverter{
		reader: bufio.NewReader(c.Input),
		config: c,
		//  stage1Channel: make(chan []byte, 10),
		// stage2Channel: make(chan BibtexEntry, 10),
		stage1OutChannel: make(chan dividerResult, 10),
		stage2OutChannel: make(chan BibtexEntry, 10),
		errorChannel:     make(chan error),
		okChannel:        make(chan struct{}, 1),
	}
}

// ErrChan returns the used error channel as a receive-only channel.
func (c *Tex2BibConverter) ErrChan() <-chan error {
	return c.errorChannel
}

// OkChan returns the channel used to notify that the conversion
// is finished. You should wait for a single receive over this channel.
// It is a 1-buffered channel.
func (c *Tex2BibConverter) OkChan() <-chan struct{} {
	return c.okChannel
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

func extractKey(line string) (string, error) {
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

// extractURL extract the URL, if any, from a plain TeX
// bib entry, by lookig for the \url command.
// If this is not found, an empty URL is returned.
func extractURL(line string) string {
	url := ""
	startIndex := strings.LastIndex(line, "\\url{")
	if startIndex == -1 {
		return url
	}

	// now we walk the string from startIndex till the
	// first '}'
	// +5 because we jump to what is after '\url{'
	for i := startIndex + 5; i < len(line); i++ {
		if line[i] != '}' {
			url += string(line[i])
		} else {
			// exit
			i = len(line)
		}
	}
	return url
}

func extractYear(line string) int {
	year := 0
	fmt.Sscanf(line, "%4d", &year)
	if !(year != 0 && len(line) <= 6) {
		year = 0
	}
	return year
}

// divider take a reader that contains a bibliography and it divides
// it into different string, each string is a bibitem that still
// need to be parsed.
// reader is the reader which items will be read from
// output is a channel which items will be written to
// errChan is a channel which errors will be written to
// When an error occurs, output channel is closed
func (c *Tex2BibConverter) divider() {

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

		line, _, err = c.reader.ReadLine()

		if err != nil {
			if err == io.EOF {
				err = ErrBibEmpty
			}
			innerLoop = false
			bibitemFindLoop = false
			c.errorChannel <- err
			close(c.stage1OutChannel)
		}

		readLine = string(line)

		if strings.Contains(readLine, BibItem) {
			bibitemFindLoop = false
			key, _ = extractKey(readLine)
			currentResult.key = key
		}
	}

	// SECOND LOOP: till the end of the file
	for innerLoop {

		line, _, err = c.reader.ReadLine()
		readLine = string(line)

		if err != nil {
			// OLD VERSION: treat io.EOF as a non-error but it's wrong because

			// if there's an error we exit from the loop
			if err == io.EOF {
				err = ErrBibUnclosed
			}
			c.errorChannel <- err
			innerLoop = false
			currentResult.value = currentEntry.String()
			c.stage1OutChannel <- currentResult
			close(c.stage1OutChannel)
		}

		if strings.Contains(readLine, BibItem) {
			// we're at the end of this bibitem
			// we push the current item to the list
			// and we reset the Builder for holding the next entry
			currentResult.value = currentEntry.String()
			c.stage1OutChannel <- currentResult
			currentEntry.Reset()

			// now reading the key
			key, _ = extractKey(readLine)
			currentResult.key = key
		} else if strings.Contains(readLine, EndBibliography) {
			// the bibliography is finished
			innerLoop = false
			currentResult.value = currentEntry.String()
			c.stage1OutChannel <- currentResult
			close(c.stage1OutChannel)
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

// parser takes an input chan in which \bibitem are
// and converts them to a BibTextEntry.
func (c *Tex2BibConverter) parser() {
	for item := range c.stage1OutChannel {

		// entry := &BasicOnlineBibtexEntry{}

		var entryURL string
		var entryAuthors []string
		var entryTitle string
		var entryYear int
		var entryVisited *time.Time

		entryVisited = c.config.DefaultVisited

		tokens := strings.Split(item.value, ",")

		// trying to extract the URL and set it
		entryURL = extractURL(item.value)

		// determine how many splits we have
		tokenLen := len(tokens)
		switch tokenLen {
		case 1:
			entryTitle = tokens[0]
		case 2:
			// just one author
			entryAuthors = tokens[0:1]
			if entryURL == "" {
				entryTitle = tokens[1]
			}
		case 3:
			entryAuthors = tokens[0:1]
			// trying to find out if the year
			// is the last token
			entryYear = extractYear(tokens[tokenLen-1])
			if entryURL == "" && entryYear == 0 {
				// author, author, title
				entryAuthors = append(entryAuthors, tokens[1])
				entryTitle = tokens[2]
			} else {
				// author, title, year|URL
				entryTitle = tokens[1]
			}
		default:

			// default case, no URL, no year
			lastAuthorIndex := tokenLen - 2
			titleIndex := tokenLen - 1

			// if URL is not empty, go back of one position
			if entryURL != "" {
				lastAuthorIndex--
				titleIndex--
			}
			//  searching the year
			entryYear = extractYear(tokens[tokenLen-1])
			if entryYear == 0 {
				entryYear = extractYear(tokens[tokenLen-2])
			}

			if entryYear != 0 {
				// going back of one position
				lastAuthorIndex--
				titleIndex--
			}

			entryAuthors = tokens[:lastAuthorIndex+1]
			entryTitle = tokens[titleIndex]
		}

		// now applying defaults
		if c.config.DefaultVisited != nil {
			entryVisited = c.config.DefaultVisited
		}

		if entryYear == 0 {
			entryYear = c.config.DefaultYear
		}

		entry := &AdvancedOnlineBibtexEntry{}

		if entryVisited != nil {
			entry.Visited = entryVisited
		}

		entry.Title = strings.TrimSpace(entryTitle)
		for i, author := range entryAuthors {
			entryAuthors[i] = strings.TrimSpace(author)
		}
		entry.Authors = entryAuthors
		entry.URL = entryURL
		entry.Year = entryYear

		key := item.key
		if key == "" {
			key = entry.GenKey()
		}
		entry.Key = key

		c.stage2OutChannel <- entry
	}
	close(c.stage2OutChannel)
}

// Convert starts the conversion into different goroutines and
// prints result to c.config.Writer.
// When it's finished, it send an empty struct on c.OkChan().
// Any error will be sent to c.ErrChan() and will cause the
/// conversion to immediately finish.
func (c *Tex2BibConverter) Convert() {
	go c.writer()
	go c.parser()
	go c.divider()
}

// writer takes input from stage2OutChannel and writes
// to the internal writer. Errors are returned
// in c.ErrChan()
func (c *Tex2BibConverter) writer() {
	for bibEntry := range c.stage2OutChannel {
		_, err := c.config.Output.Write([]byte(bibEntry.String() + "\n\n"))
		if err != nil {
			c.errorChannel <- err
		}
	}
	// when finished, just sending the ok value.
	c.okChannel <- struct{}{}
}
