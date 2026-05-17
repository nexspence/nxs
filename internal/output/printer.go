package output

import "fmt"

type Printer interface {
	Table(headers []string, rows [][]string)
	Success(msg string)
	Error(msg string)
	JSON(v any)
}

type stubPrinter struct{}

func (s *stubPrinter) Table(h []string, r [][]string) {}
func (s *stubPrinter) Success(msg string)              { fmt.Println(msg) }
func (s *stubPrinter) Error(msg string)                { fmt.Println("Error: " + msg) }
func (s *stubPrinter) JSON(v any)                      { fmt.Printf("%+v\n", v) }

func NewPrinter(jsonMode, plainMode bool) Printer { return &stubPrinter{} }
