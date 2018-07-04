package gobib

import (
	"container/list"
	"fmt"
	"strings"
	"testing"
)

const bibliography = `
	\bibitem{}
	Ross Anderson, Why Cryptosystem Fails
`

var bibliographyReader = strings.NewReader(bibliography)
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

func (c *converterTest) runDivider() (*list.List, error) {
	return divider(c.Converter.reader)
}

func TestDivider1(t *testing.T) {
	converter := initConverter(&Config{
		Input:  bibliographyReader,
		Output: &bibliographyWriter,
	})

	// expected := strings.Replace(bibliography, "\\bibitem{}\n", "", -1)
	// expected = strings.TrimSpace(expected)
	got, err := converter.runDivider()
	if err != nil {
		t.Fatal("Got error while dividing")
	}
	for e := got.Front(); e != nil; e = e.Next() {
		fmt.Printf(e.Value.(string))
	}
}
