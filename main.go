package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func setCoverPage(book *Epub, root string) error {
	path := filepath.Join(root, "cover.html")
	data, e := ioutil.ReadFile(path)
	if e == nil {
		book.SetCoverPage("cover.html", data)
	}
	return e
}

func addFilesToBook(book *Epub, root string) error {
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		path, _ = filepath.Rel(root, path)
		path = strings.ToLower(filepath.ToSlash(path))
		if path == "book.ini" || path == "book.html" || path == "cover.html" {
			return nil
		}
		return book.AddFile(path, data)
	}

	return filepath.Walk(root, walk)
}

func addChaptersToBook(book *Epub, root string, depth int) error {
	f, e := os.Open(filepath.Join(root, "book.html"))
	if e != nil {
		return e
	}
	defer f.Close()
	br := bufio.NewReader(f)

	header := ""
	for {
		s, _, e := br.ReadLine()
		if e != nil {
			return e
		}
		l := string(s)
		header += l + "\n"
		if strings.ToLower(strings.TrimSpace(l)) == "<body>" {
			break
		}
	}

	buf := new(bytes.Buffer)
	title := ""
	d := 0
	for {
		s, _, e := br.ReadLine()
		if e == io.EOF {
			break
		}
		l := string(s)
		if l[0] == '<' && (l[1] == 'h' || l[1] == 'H') && l[3] == '>' {
			nd := int(l[2] - '0')
			if nd > 0 && nd <= depth {
				if buf.Len() > 0 {
					buf.WriteString("	</body>\n</html>")
					if e = book.AddChapter(title, buf.Bytes(), d); e != nil {
						return e
					}
				}
				d = nd
				title = l[4:strings.LastIndex(l, "<")]
				buf.Reset()
				buf.WriteString(header)
			}
		}

		buf.WriteString(l + "\n")
	}

	if buf.Len() > 0 {
		if e = book.AddChapter(title, buf.Bytes(), d); e != nil {
			return e
		}
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: makeepub folder [output]")
		os.Exit(1)
	}

	ini, e := Open(filepath.Join(os.Args[1], "book.ini"))
	if e != nil {
		fmt.Println("Error: failed to open 'book.ini'")
		os.Exit(1)
	}

	s := ini.GetString("book", "id", "")
	book, e := NewEpub(s)
	if e != nil {
		fmt.Println("Error: failed to create epub book.")
		os.Exit(1)
	}

	s = ini.GetString("book", "name", "")
	if len(s) == 0 {
		fmt.Println("Warning: book name is empty.")
	}
	book.SetName(s)

	s = ini.GetString("book", "author", "")
	if len(s) == 0 {
		fmt.Println("Warning: author name is empty.")
	}
	book.SetAuthor(s)

	if setCoverPage(book, os.Args[1]) != nil {
		fmt.Println("Error: failed to set cover page.")
		os.Exit(1)
	}

	if addFilesToBook(book, os.Args[1]) != nil {
		fmt.Println("Error: failed to add files to book.")
		os.Exit(1)
	}

	depth := ini.GetInt("book", "depth", 1)
	if depth < 1 || depth > book.MaxDepth() {
		fmt.Println("Warning: invalid 'depth' value, reset to '1'")
		depth = 1
	}
	if addChaptersToBook(book, os.Args[1], depth) != nil {
		fmt.Println("Error: failed to add chapters to book.")
		os.Exit(1)
	}

	s = ini.GetString("output", "path", "")
	if len(os.Args) >= 3 {
		s = os.Args[2]
	}
	if len(s) == 0 {
		fmt.Println("Warning: output path has not set.")
	} else if book.Save(s) != nil {
		fmt.Println("Error: failed to create output file: ", e.Error())
		os.Exit(1)
	}

	fmt.Println("Done.")
	os.Exit(0)
}