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
	"strings"
	"testing"
	"time"
)

const correctBibliography = `
\begin{thebibliography}
	\bibitem{wcf}
	Ross Anderson, Why Cryptosystems Fail

	\bibitem{wcdf}
	Ross Anderson, Why Cryptosystems Don't Fail
\end{thebibliography}
`

const bib = `
\begin{thebibliography}
	\bibitem{wcf}
	Ross Anderson, Why Cryptosystems Fail, 1909, \url{example.com/ra/wcf.pdf}

	\bibitem{wcdf}
	Ross Anderson, Why Cryptosystems Don't Fail

	\bibitem{aass}
	Asking Alexandria, Someone Somewhere, 2011

\end{thebibliography}
`

var bibResult = []Entry{
	{
		Title:   "Why Cryptosystems Fail",
		Authors: []string{"Ross Anderson"},
		Year:    1909,
		URL:     "example.com/ra/wcf.pdf",
		Key:     "wcf",
	},
	{
		Title:   "Why Cryptosystems Don't Fail",
		Authors: []string{"Ross Anderson"},
		Key:     "wcdf",
	},
	{
		Title:   "Someone Somewhere",
		Authors: []string{"Asking Alexandria"},
		Year:    2011,
		Key:     "aass",
	},
}

const expectedBib = `@online{wcf,
	author = "Ross Anderson",
	title = {{Why Cryptosystems Fail}},
	year = "1909",
	url = {example.com/ra/wcf.pdf},
}

@online{wcdf,
	author = "Ross Anderson",
	title = {{Why Cryptosystems Don't Fail}},
	year = "2010",
}

@online{aass,
	author = "Asking Alexandria",
	title = {{Someone Somewhere}},
	year = "2011",
}

`

const expectedBibWithVisited = `@online{wcf,
	author = "Ross Anderson",
	title = {{Why Cryptosystems Fail}},
	year = "1909",
	url = {example.com/ra/wcf.pdf},
	urldate = "2018-7-6",
}

@online{wcdf,
	author = "Ross Anderson",
	title = {{Why Cryptosystems Don't Fail}},
	year = "2010",
	urldate = "2018-7-6",
}

@online{aass,
	author = "Asking Alexandria",
	title = {{Someone Somewhere}},
	year = "2011",
	urldate = "2018-7-6",
}

`

const wrongBibliography = `
\begin{thebibliography}
	\bibitem{}
	Ross Anderson, Why Cryptosystems Fail

	\bibitem{}
	Ross Anderson, Why Cryptosystems Don't Fail
`

// var wrongBibliographyReader = strings.NewReader(wrongBibliography)
// var emptyBibliographyReader = strings.NewReader("")

type converterTest struct {
	Converter *Tex2BibConverter
}

// func BibtexEntryEqual(entry1, entry2 *Entry) bool {
//	return entry1.Key == entry2.Key
// }

func ExtendedBibtexEntryEqual(entry1, entry2 *Entry) bool {
	if entry1.Key != entry2.Key {
		return false
	}
	if entry1.AuthorsToString() != entry2.AuthorsToString() {
		return false
	}

	if entry1.Title != entry2.Title {
		return false
	}

	if entry1.URL != "" && entry2.URL != "" {
		if entry1.URL != entry2.URL {
			return false
		}
	}

	if entry1.Year != 0 && entry2.Year != 0 {
		if entry1.Year != entry2.Year {
			return false
		}
	}
	return true
}

func gotExpected(got, expected string, checkSimilar bool, t *testing.T) {
	ok := false
	if checkSimilar {
		if strings.Contains(got, expected) || strings.Contains(expected, got) {
			ok = true
		}
	}
	if expected == got {
		ok = true
	}
	if !ok {
		t.Errorf("Got: '%s',\nExp: '%s'", got, expected)
	}
}

func initConverter(c *Config) *converterTest {
	converter := NewConverter(c)
	return &converterTest{converter}
}

func (t *converterTest) runDivider() {
	go t.Converter.divider()
}

func runKeyFromLine(line string) (string, error) {
	return extractKey(line)
}

func runExtractURL(line string) string {
	return extractURL(line)
}

func TestDividerOk(t *testing.T) {
	var writer strings.Builder
	bibliographyReader := strings.NewReader(correctBibliography)
	converter := initConverter(&Config{
		Input:  bibliographyReader,
		Output: &writer,
	})

	expectedLen := 2
	converter.runDivider()
	var entriesLen int
	var err error
	loop := true
	for loop {
		select {
		case entry := <-converter.Converter.stage1OutChannel:
			entriesLen++
			if entriesLen == expectedLen {
				loop = false
			}
			t.Logf(entry.String())
		case err = <-converter.Converter.errorChannel:
			loop = false
		}
	}
	if err != nil {
		t.Fatal("Got error while dividing")
	}
	if entriesLen != 2 {
		t.Errorf("Error, mismatch length in list")
	}

	// for e := got.Front(); e != nil; e = e.Next() {
	// 	fmt.Printf("Entry " + e.Value.(string) + "\n")
	// }
}

func TestDividerNoEnd(t *testing.T) {
	var writer strings.Builder
	wrongBibliographyReader := strings.NewReader(wrongBibliography)
	converter := initConverter(&Config{
		Input:  wrongBibliographyReader,
		Output: &writer,
	})

	converter.runDivider()
	err := <-converter.Converter.errorChannel
	if err == nil {
		t.Fatalf("error is nil")
	} else if err != ErrBibUnclosed {
		t.Fatalf("err != ErrBibUnclosed" + err.Error())
	}
}

func TestEmptyDivider(t *testing.T) {
	var writer strings.Builder
	wrongBibliographyReader := strings.NewReader("")
	converter := initConverter(&Config{
		Input:  wrongBibliographyReader,
		Output: &writer,
	})

	converter.runDivider()
	err := <-converter.Converter.errorChannel
	if err == nil {
		t.Fatalf("error is nil")
	} else if err != ErrBibEmpty {
		t.Fatalf("err != ErrBibEmpty: " + err.Error())
	}
}

func TestKeyFromLine(t *testing.T) {

	line := "\\bibitem{item}"
	expected := "item"
	key, err := runKeyFromLine(line)
	if err != nil {
		t.Fatalf("Err find key: %s", err.Error())
	}

	gotExpected(key, expected, false, t)
}

func TestExtractURL(t *testing.T) {
	line := "hello, \\url{example.com/golang}, hey"
	expected := "example.com/golang"
	got := runExtractURL(line)

	gotExpected(got, expected, false, t)
}

func TestExtractEmptyURL(t *testing.T) {
	line := "\\bibitem{item}\n hello world"
	expected := ""
	got := runExtractURL(line)
	gotExpected(got, expected, false, t)
}

func TestParser(t *testing.T) {
	config := &Config{
		Input:       strings.NewReader(bib),
		DefaultYear: 1900,
	}
	converter := initConverter(config)
	// converter.convert()

	go converter.Converter.parser()
	go converter.Converter.divider()

	i := 0

	loop := true
	for loop {
		select {
		case err := <-converter.Converter.errorChannel:
			t.Errorf(err.Error())
			loop = false
		case bibEntry, ok := <-converter.Converter.stage2OutChannel:
			if !ok {
				loop = false
			} else {
				t.Logf(bibEntry.String())
				if !ExtendedBibtexEntryEqual(bibEntry.(*Entry), &bibResult[i]) {
					t.Errorf("Fail to check: %s %s", bibEntry.String(), bibResult[i].String())
				}
				i++
			}
		}
	}
}

func runTestComplete(c *Config, expectedOutput string, t *testing.T) {
	// var writer strings.Builder
	converter := initConverter(c)
	writer := c.Output.(*strings.Builder)
	converter.Converter.Convert()

	var result string
	ok, err := converter.Converter.OkChan(), converter.Converter.ErrChan()
	select {
	case <-err:
		t.Errorf("Fail to convert")
	case <-ok:
		result = writer.String()
		t.Logf("Result of conversion:\n%s", result)
	}

	if result != expectedOutput {
		t.Errorf("Difference!, expected:\n%s", expectedOutput)
	}
}

func TestCompleteEmptyVisit(t *testing.T) {
	var writer strings.Builder
	config := &Config{
		Output:      &writer,
		Input:       strings.NewReader(bib),
		DefaultYear: 2010,
	}

	runTestComplete(config, expectedBib, t)
}

func TestCompleteWithVisit(t *testing.T) {
	var writer strings.Builder
	defaultTime, _ := time.Parse("2006-01-02", "2018-07-06")
	config := &Config{
		Output:         &writer,
		Input:          strings.NewReader(bib),
		DefaultYear:    2010,
		DefaultVisited: &defaultTime,
	}
	runTestComplete(config, expectedBibWithVisited, t)
}

func TestNewBasic(t *testing.T) {
	NewEntry("key", []string{"author0"}, "title", 2018, "", nil)
}

func TestNewAdvancedWithKey(t *testing.T) {
	entry := NewEntry("", []string{"foo"}, "bar", 2018, "", nil)
	expected := "bar-2018-foo"
	got := entry.Key
	if expected != got {
		t.Errorf("Fail to test GenKey(), expected: %s, got: %s", expected, got)
	}
}
