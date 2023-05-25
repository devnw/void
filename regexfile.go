package main

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"runtime/debug"
	"strings"

	"golang.org/x/exp/slog"
)

func ReadRegex(
	ctx context.Context,
	logger *slog.Logger,
	path string,
) []*regexp.Regexp {
	var regex []*regexp.Regexp

	count := 0
	bodies := ReadFiles(ctx, logger, ReadDirectory(ctx, logger, path, ".regex"))
	for {
		select {
		case <-ctx.Done():
			return regex
		case body, ok := <-bodies:
			if !ok {
				return regex
			}

			regex = append(regex, parseRegexFile(ctx, logger, body)...)
			count++
		}
	}
}

func parseRegexFile(
	ctx context.Context,
	logger *slog.Logger,
	body io.ReadCloser,
) []*regexp.Regexp {
	regex := []*regexp.Regexp{}
	defer func() {
		r := recover()
		if r != nil {
			logger.ErrorCtx(ctx,
				"panic",
				slog.String("error", fmt.Sprintf("%v", r)),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()

	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		logger.ErrorCtx(ctx,
			"failed to read regex file body",
			slog.String("error", err.Error()),
		)
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
				logger.ErrorCtx(ctx,
					"failed to compile regex",
					slog.String("regex", line),
					slog.String("error", err.Error()),
				)
				continue
			}

			regex = append(regex, r)
		}
	}

	return regex
}
