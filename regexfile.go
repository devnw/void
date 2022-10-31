package main

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.devnw.com/event"
)

func ReadRegex(
	ctx context.Context,
	pub *event.Publisher,
	path string,
) []*regexp.Regexp {
	var regex []*regexp.Regexp

	count := 0
	bodies := ReadFiles(ctx, pub, ReadDirectory(ctx, pub, path, ".regex"))
	for {
		select {
		case <-ctx.Done():
			return regex
		case body, ok := <-bodies:
			if !ok {
				return regex
			}

			regex = append(regex, parseRegexFile(ctx, pub, body)...)
			count++
			fmt.Printf("Processed %d files\n", count)
		}
	}
}

func parseRegexFile(
	ctx context.Context,
	pub *event.Publisher,
	body io.ReadCloser,
) []*regexp.Regexp {
	regex := []*regexp.Regexp{}
	defer func() {
		r := recover()
		if r != nil {
			pub.ErrorFunc(ctx, func() error {
				return &Error{
					Msg:   "error parsing regex file",
					Inner: fmt.Errorf("%s", r),
				}
			})
		}
	}()

	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		pub.ErrorFunc(ctx, func() error {
			return &Error{
				Msg:   "failed to read regex file",
				Inner: err,
			}
		})
		return nil
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		select {
		case <-ctx.Done():
			return regex
		default:
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
				pub.ErrorFunc(ctx, func() error {
					return &Error{
						Msg: fmt.Sprintf(
							"failed to compile regex %s", line,
						),
						Inner: err,
					}
				})
				continue
			}

			regex = append(regex, r)
		}
	}

	return regex
}
