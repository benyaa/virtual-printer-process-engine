package engine

import (
	"github.com/google/uuid"
	"io"
	"os"
	"path"
)

type DefaultEngineFileHandler struct {
	input  string
	output string
	reader *os.File
	writer *os.File
}

func (d *DefaultEngineFileHandler) Read() (io.Reader, error) {
	if d.reader != nil {
		return d.reader, nil
	}
	file, err := os.Open(d.input)
	if err != nil {
		return nil, err
	}
	d.reader = file
	return d.reader, nil
}

func (d *DefaultEngineFileHandler) Write() (io.Writer, error) {
	if d.writer != nil {
		return d.writer, nil
	}
	file, err := os.Create(d.output)
	if err != nil {
		return nil, err
	}
	d.writer = file
	return d.writer, nil
}

func (d *DefaultEngineFileHandler) Close() {
	if d.reader != nil {
		d.reader.Close()
		d.reader = nil
	}
	if d.writer != nil {
		d.writer.Close()
		d.writer = nil
	}
}

func (d *DefaultEngineFileHandler) getNewFileHandler() *DefaultEngineFileHandler {
	input := d.input
	if d.writer != nil {
		input = d.output
		defer os.Remove(d.input)
	}

	d.Close()

	return &DefaultEngineFileHandler{
		input:  input,
		output: generateNewOutputFilePath(input),
	}
}

func NewDefaultEngineFileHandler(input string) *DefaultEngineFileHandler {
	return &DefaultEngineFileHandler{
		input:  input,
		output: generateNewOutputFilePath(input),
	}
}

func generateNewOutputFilePath(input string) string {
	return path.Join(path.Dir(input), uuid.NewString())
}
