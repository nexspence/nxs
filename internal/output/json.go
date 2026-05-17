package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type JSONPrinter struct {
	w io.Writer
}

func (p *JSONPrinter) out() io.Writer {
	if p.w != nil {
		return p.w
	}
	return os.Stdout
}

func (p *JSONPrinter) Table(headers []string, rows [][]string) {
	result := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		m := map[string]string{}
		for i, h := range headers {
			if i < len(row) {
				m[h] = row[i]
			}
		}
		result = append(result, m)
	}
	p.JSON(result)
}

func (p *JSONPrinter) Success(msg string) { p.JSON(map[string]string{"message": msg}) }
func (p *JSONPrinter) Error(msg string)   { fmt.Fprintf(os.Stderr, `{"error":%q}`+"\n", msg) }
func (p *JSONPrinter) JSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Fprintln(p.out(), string(b))
}
