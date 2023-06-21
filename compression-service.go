package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const BufferSize = 104857600 // You can adjust this value to fit your needs
//const BufferSize = 1024 // You can adjust this value to fit your needs

func compress(src string, outFileName string) error {
	// tar > gzip > outFile

	outFile, err := os.Create(outFileName)
	if err != nil {
		return err
	}
	zr := gzip.NewWriter(outFile)
	tw := tar.NewWriter(zr)

	buf := make([]byte, BufferSize)
	// walk through every file in the folder
	filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		// generate tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		// must provide real name
		// (see https://golang.org/src/archive/tar/common.go?#L626)
		header.Name = filepath.ToSlash(file)

		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			// use buffered reader for large files
			//if _, err := io.Copy(tw, data); err != nil {
			//	return err
			//}

			// Create a buffered reader
			reader := bufio.NewReader(data)

			// Write the file data in chunks
			for {
				// Read data to buffer
				n, err := reader.Read(buf)
				if err != nil && err != io.EOF {
					panic(err)
				}

				if n == 0 {
					break
				}

				// Write buffer to file
				if _, err := tw.Write(buf[:n]); err != nil {
					panic(err)
				}
			}
		}
		return nil
	})

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return err
	}
	//
	return nil
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
