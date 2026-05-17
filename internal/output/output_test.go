package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nexspence/nxs/internal/output"
)

func TestPlainPrinter_Table(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPlainPrinterTo(&buf)
	p.Table([]string{"NAME", "FORMAT"}, [][]string{
		{"maven-releases", "maven"},
		{"npm-proxy", "npm"},
	})
	out := buf.String()
	if !strings.Contains(out, "maven-releases\tmaven") {
		t.Errorf("missing TSV row, got: %q", out)
	}
	if !strings.Contains(out, "NAME\tFORMAT") {
		t.Errorf("missing TSV header, got: %q", out)
	}
}

func TestJSONPrinter_JSON(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewJSONPrinterTo(&buf)
	type Repo struct {
		Name string `json:"name"`
	}
	p.JSON(Repo{Name: "maven-releases"})
	var got Repo
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "maven-releases" {
		t.Errorf("got name %q", got.Name)
	}
}

func TestNewPrinter_JSONMode(t *testing.T) {
	p := output.NewPrinter(true, false)
	if _, ok := p.(*output.JSONPrinter); !ok {
		t.Errorf("expected JSONPrinter, got %T", p)
	}
}

func TestNewPrinter_PlainMode(t *testing.T) {
	p := output.NewPrinter(false, true)
	if _, ok := p.(*output.PlainPrinter); !ok {
		t.Errorf("expected PlainPrinter, got %T", p)
	}
}
