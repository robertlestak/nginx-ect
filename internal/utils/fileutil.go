package utils

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

func ReadFile(filePath string) (string, error) {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "ReadFile",
		"file": filePath,
	})
	l.Debug("reading file")
	if filePath == "-" {
		l.Debug("reading from stdin")
		return ReadStdin()
	}
	l.Debug("reading from file")
	return ReadFileFromPath(filePath)
}

func WriteFile(filePath string, content string) error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "WriteFile",
		"file": filePath,
	})
	l.Debug("writing file")
	if filePath == "-" {
		l.Debug("writing to stdout")
		return WriteStdout(content)
	}
	l.Debug("writing to file")
	return WriteFileToPath(filePath, content)
}

// WriteStdout writes the specified content to stdout.
func WriteStdout(content string) error {
	_, err := os.Stdout.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}

// WriteFileToPath writes the specified content to the specified file path.
func WriteFileToPath(filePath string, content string) error {
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return err
	}
	return nil
}

func ReadStdin() (string, error) {
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ReadFileFromPath reads from the specified file path and returns the content as a string.
func ReadFileFromPath(filePath string) (string, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
