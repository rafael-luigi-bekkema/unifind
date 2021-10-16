package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const unicodeIndex = "https://www.unicode.org/Public/UCD/latest/ucd/Index.txt"
const unicodeNamesList = "https://www.unicode.org/Public/UCD/latest/ucd/NamesList.txt"
const appName = "unifind"

func fetchUnicodeURL(url string) (io.ReadCloser, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("could not find user cache dir: %w", err)
	}
	cacheDir = filepath.Join(cacheDir, appName, "ucd")
	fileName := path.Base(url)
	cachePath := filepath.Join(cacheDir, fileName)
	f, err := os.Open(cachePath)
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("could not open file %q: %w", cachePath, err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("could not make cache path %s: %w", cachePath, err)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not fetch %q: %w", unicodeIndex, err)
	}
	defer resp.Body.Close()
	f, err = os.Create(cachePath)
	if err != nil {
		return nil, fmt.Errorf("could not create cache path %q: %w", cachePath, err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		return nil, fmt.Errorf("could not download %q to %q: %w", unicodeIndex, cachePath, err)
	}
	f.Seek(0, 0)
	return f, nil
}

type CodePoint struct {
	Chr  rune
	Desc string
}

func errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func searchIndex(search string) (cp []CodePoint, err error) {
	search = strings.ToLower(search)
	f, err := fetchUnicodeURL(unicodeIndex)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := bufio.NewScanner(f)
	for buf.Scan() {
		parts := strings.Split(buf.Text(), "\t")
		if len(parts) != 2 {
			errorf("invalid format, expected 2 fields, got %d\n", len(parts))
		}
		if strings.Contains(strings.ToLower(parts[0]), search) {
			chr, err := strconv.ParseInt(parts[1], 16, 32)
			if err != nil {
				errorf("invalid rune %q: %s", parts[1], err)
			}
			cp = append(cp, CodePoint{rune(chr), parts[0]})
		}
	}
	return cp, nil
}

func matchAll(target []string, query string) bool {
	q := strings.Fields(query)
	for _, part := range q {
		var match bool
		for _, t := range target {
			if strings.Contains(t, part) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

func searchNamesList(search string) (cp []CodePoint, err error) {
	search = strings.ToLower(search)
	var schr string
	matcher := func(desc []string) {
		if matchAll(desc, search) {
			i, err := strconv.ParseInt(schr, 16, 32)
			if err != nil {
				errorf("invalid rune %q: %s", schr, err)
				return
			}
			cp = append(cp, CodePoint{rune(i), strings.Join(desc, " ")})
		}
	}
	f, err := fetchUnicodeURL(unicodeNamesList)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rdr := bufio.NewScanner(f)
	desc := make([]string, 0, 5)
	for rdr.Scan() {
		line := rdr.Text()
		if strings.HasPrefix(line, ";") || strings.HasPrefix(line, "@") || strings.HasPrefix(line, "\t\t") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			errorf("invalid format, expected 2 fields, got %d: %s\n", len(parts), line)
			continue
		}
		if parts[0] != "" {
			matcher(desc)
			schr = parts[0]
			desc = desc[0:0]
		}
		desc = append(desc, parts[1])
	}
	matcher(desc)
	return cp, nil
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("provide a search term")
	}
	cp, err := searchNamesList(strings.Join(os.Args[1:], " "))
	if err != nil {
		return err
	}
	for _, c := range cp {
		fmt.Printf("%c %s\n", c.Chr, c.Desc)
	}
	if len(cp) == 0 {
		return fmt.Errorf("Not found")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
