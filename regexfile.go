package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

func ReadRegex(ctx context.Context, path string) []*regexp.Regexp {
	var regex []*regexp.Regexp

	count := 0
	bodies := ReadFiles(ctx, ReadDirectory(ctx, path, ".regex"))
	for {
		select {
		case <-ctx.Done():
			return regex
		case body, ok := <-bodies:
			if !ok {
				return regex
			}

			regex = append(regex, parseRegexFile(ctx, body)...)
			count++
			fmt.Printf("Processed %d files\n", count)
		}
	}
}

func parseRegexFile(ctx context.Context, body io.ReadCloser) []*regexp.Regexp {
	regex := []*regexp.Regexp{}
	defer func() {
		r := recover()
		if r != nil {
			fmt.Printf("error parsing regex file; %s\n", r)
		}
	}()

	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return nil
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove the comment from the line
		commentIndex := strings.Index(line, "#")
		if commentIndex != -1 {
			line = strings.TrimSpace(
				line[:commentIndex],
			)
		}

		var r *regexp.Regexp
		r, err = regexp.Compile(line)
		if err != nil {
			continue
		}

		regex = append(regex, r)
	}

	return regex
}
