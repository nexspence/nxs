package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type PlainPrinter struct {
	w io.Writer
}

func (p *PlainPrinter) out() io.Writer {
	if p.w != nil {
		return p.w
	}
	return os.Stdout
}

func (p *PlainPrinter) Table(headers []string, rows [][]string) {
	fmt.Fprintln(p.out(), strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(p.out(), strings.Join(row, "\t"))
	}
}

func (p *PlainPrinter) Success(msg string) { fmt.Fprintln(p.out(), msg) }
func (p *PlainPrinter) Error(msg string)   { fmt.Fprintln(os.Stderr, "Error: "+msg) }
func (p *PlainPrinter) JSON(v any)         { fmt.Fprintf(p.out(), "%+v\n", v) }
