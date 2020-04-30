package logger

import (
	"fmt"
	"io"
)

type CapturingWriter struct {
	Writer       io.Writer
	BytesWritten []byte
}

func NewLogger(writer io.Writer) CapturingWriter {
	return CapturingWriter{
		Writer: writer,
	}
}

func (k CapturingWriter) Write(p []byte) (n int, err error) {
	k.BytesWritten = append(k.BytesWritten, p...)
	return k.Writer.Write(p)
}

func (k CapturingWriter) Println(v ...interface{}) (n int, err error) {
	return k.Write([]byte(fmt.Sprintln(v...)))
}
