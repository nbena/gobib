# gobib

This package is a **yet-to-be-fully-tested** stupid program to convert plain TeX bibliography file into structured BibTeX, and I'm primarly writing it for my personal use hoping it'll be helpful for someone else too.

## Installing

```bash
go get https://github.com/nbena/gobib/cmd/gobib
```

## Usage

Assuming the your plain TeX file is `bib.tex`, and you want to write to `bib.bib`, invoking the program is very easy:

```bash
gobib -input=bib.tex -output=bib.bib
```

The help message:

```txt
Usage of ./gobib:
  -default-urldate string
        the default urldate value to use, the format is YYYY-MM-DD
  -default-year int
        the default year value to use when a year is not found
  -in string
        the input file
  -out string
        the output file
  -print-finished
        print a message when conversion is finished
```

## How it works

The program applies very simple heuristic that works fine for my use cases:

- the base case is:

  ```latex
    \bibitem{}
    author1, author2, authorn, title, \url, year
  ```

  or

  ```latex
  \bibitem{}
  author1, author2, authorn, title, year, \url
  ```

- when an URL is not found (the program will search for `\url`) it simply won't be added,
  and the last item will be considered the title.

- the program will search for a valid year in the last, or last - 1 items of each bib item, in that case, the title will just be the item before the year.

- the program can add a default `year` and `urldate`, but only if you want to. Don't invoke this options (`default-year` and `default-urldate`) to not add default values.

- any other element inside an item will be *probably* considered an author.

- the **default** generated Bibitem element is `@online`, this will maybe change in future but I don't think so.

The program by default reads from `stdin` and writes to `stdout` using a **3-stage pipeline** running 3 goroutines:

1. one for extract raw items from the input
2. one for parsing raw items into structured Bibitems
3. one for writing

Reading stops at `EOF` or better, at `\end{thebibliography}`. The first error that occurs causes the program to exit.

## Example

Given the following input:

```txt
\begin{thebibliography}{10}

    \bibitem{how-to-be}
    Foo Bar, How to be
  
    \bibitem{adv}
    F. Bar, Advanced Topics in Advanced Topics, 2018
  
    \bibitem{you-me}
    You, Me, How is it possible that You is not Me,
    \url{https://example.com/youvsme}
  
    \bibitem{yabe}
    One Author, Another One, YABE -- Yet Another Bib Entry,
    \url{https://example.com/yabe}, 2018
  
    \bibitem{yabe2}
    One Author, YABE2 -- A revision of YABE, 2018
    \url{https://example.com/yabe2}
  
\end{thebibliography}
```

Running:

```bash
gobib -in=input.tex -out=output.bib -default-year=2018
```

You get:

```txt
@online{how-to-be,
    author = "Foo Bar",
    title = "How to be",
    year = "2018",
}

@online{adv,
    author = "F. Bar",
    title = "Advanced Topics in Advanced Topics",
    year = "2018",
}

@online{you-me,
    author = "You and Me",
    title = "How is it possible that You is not Me",
    year = "2018",
    url = "https://example.com/youvsme",
}

@online{yabe,
    author = "One Author and Another One",
    title = "YABE -- Yet Another Bib Entry",
    year = "2018",
    url = "https://example.com/yabe",
}

@online{yabe2,
    author = "One Author",
    title = "YABE2 -- A revision of YABE",
    year = "2018",
    url = "https://example.com/yabe2",
}

```

## License

GPL3