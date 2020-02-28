package main

import (
	"archive/zip"
	"github.com/DATA-DOG/godog/gherkin"
	"io"
	"os"
	"path/filepath"
)

func tagsContains(tags []*gherkin.Tag, tagName string) bool {
	for _, t := range tags {
		if t.Name == tagName {
			return true
		}
	}
	return false
}

func zipDirectory(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Right now, just zip files from directory
		if info.IsDir() {
			return nil
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Method = zip.Deflate
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}
