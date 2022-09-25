package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	stream "go.atomizer.io/stream"
	"go.structs.dev/gen"
)

// ReadFiles reads the files at the path provided
// and returns a channel of io.ReadCloser where it
// deposits the open file.
func ReadFiles(
	ctx context.Context,
	files <-chan string,
) <-chan io.ReadCloser {
	s := stream.Scaler[string, io.ReadCloser]{
		Wait: time.Nanosecond,
		Life: time.Millisecond,
		Fn: func(
			ctx context.Context,
			path string,
		) (io.ReadCloser, bool) {
			data, err := os.Open(path)
			if err != nil {
				return nil, false
			}

			return data, true
		},
	}

	out, err := s.Exec(ctx, files)
	if err != nil {
		panic(err)
	}

	return out
}

// ReadDirectory recursively reads through the directory structure
// providing a channel of file paths.
func ReadDirectory(
	ctx context.Context,
	dir string,
	exts ...string,
) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Println(err)
			return
		}

		wg := sync.WaitGroup{}
		for _, file := range files {
			if !file.IsDir() {
				i, err := file.Info()
				if err != nil {
					continue
				}

				if len(exts) > 0 {
					if !gen.Has(exts, filepath.Ext(i.Name())) {
						continue
					}
				}

				select {
				case <-ctx.Done():
					return
				case out <- path.Join(dir, i.Name()):
					fmt.Println(i.Name())
				}

				continue
			}

			i, err := file.Info()
			if err != nil {
				return
			}

			wg.Add(1)
			go func(d os.FileInfo) {
				defer wg.Done()

				stream.Pipe(
					ctx,
					ReadDirectory(
						ctx,
						path.Join(
							dir,
							d.Name(),
						),
					),
					out,
				)
			}(i)
		}

		wg.Wait()
	}()

	return out
}
