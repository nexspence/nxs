package output

import "io"

type Printer interface {
	Table(headers []string, rows [][]string)
	Success(msg string)
	Error(msg string)
	JSON(v any)
}

func NewPrinter(jsonMode, plainMode bool) Printer {
	if jsonMode {
		return &JSONPrinter{}
	}
	if plainMode {
		return &PlainPrinter{}
	}
	return &RichPrinter{}
}

func NewPlainPrinterTo(w io.Writer) *PlainPrinter {
	return &PlainPrinter{w: w}
}

func NewJSONPrinterTo(w io.Writer) *JSONPrinter {
	return &JSONPrinter{w: w}
}
