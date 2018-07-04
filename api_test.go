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
)

const correctBibliography = `
\begin{thebibliography}
	\bibitem{}
	Ross Anderson, Why Cryptosystems Fail

	\bibitem{}
	Ross Anderson, Why Cryptosystems Don't Fail
\end{thebibliography}
`

const wrongBibliography = `
\begin{thebibliography}
	\bibitem{}
	Ross Anderson, Why Cryptosystems Fail

	\bibitem{}
	Ross Anderson, Why Cryptosystems Don't Fail
`

var bibliographyReader = strings.NewReader(correctBibliography)
var wrongBibliographyReader = strings.NewReader(wrongBibliography)
var emptyBibliographyReader = strings.NewReader("")
var bibliographyWriter strings.Builder

type converterTest struct {
	Converter *Tex2BibConverter
}

func (c *converterTest) gotExpected(got, expected string, checkSimilar bool, t *testing.T) {
	ok := false
	if checkSimilar {
		if strings.Contains(got, expected) || strings.Contains(expected, got) {
			ok = true
		}
	}
	if expected != got {
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

func (c *converterTest) runDivider() (chan string, chan error) {
	output := make(chan string, 2)
	errChan := make(chan error, 2)
	go divider(c.Converter.reader, output, errChan)
	return output, errChan
}

func TestDividerOk(t *testing.T) {
	converter := initConverter(&Config{
		Input:  bibliographyReader,
		Output: &bibliographyWriter,
	})

	expectedLen := 2
	output, errChan := converter.runDivider()
	var entriesLen int
	var err error
	loop := true
	for loop {
		select {
		case <-output:
			entriesLen++
			if entriesLen == expectedLen {
				loop = false
			}
		case err = <-errChan:
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
	converter := initConverter(&Config{
		Input:  wrongBibliographyReader,
		Output: &bibliographyWriter,
	})

	_, errChan := converter.runDivider()
	err := <-errChan
	if err == nil {
		t.Fatalf("error is nil")
	} else if err != ErrBibUnclosed {
		t.Fatalf("err != ErrBibUnclosed" + err.Error())
	}
}

func TestEmptyDivider(t *testing.T) {
	converter := initConverter(&Config{
		Input:  wrongBibliographyReader,
		Output: &bibliographyWriter,
	})

	_, errChan := converter.runDivider()
	err := <-errChan
	if err == nil {
		t.Fatalf("error is nil")
	} else if err != ErrBibEmpty {
		t.Fatalf("err != ErrBibEmpty" + err.Error())
	}
}
