package toolbox

import "testing"

const validTB = `
version: 1
tools:
  search:
    description: Search the web
    input:
      query:
        type: string
    output:
      results:
        type: array
    runtime:
      type: mcp
      server: "web-search-server"
`

func TestParseAndValidate_Valid(t *testing.T) {
	tb, err := Parse([]byte(validTB))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if errs := Validate(tb); len(errs) != 0 {
		t.Fatalf("Validate() errors = %v, want none", errs)
	}
	if tb.Tools["search"].Runtime.Server != "web-search-server" {
		t.Errorf("Runtime.Server = %q, want %q", tb.Tools["search"].Runtime.Server, "web-search-server")
	}
}

func TestValidate_WrongVersion(t *testing.T) {
	tb, _ := Parse([]byte("version: 2\ntools:\n  a:\n    runtime:\n      type: mcp\n      server: x\n"))
	if errs := Validate(tb); len(errs) == 0 {
		t.Fatal("expected an error for unsupported version")
	}
}

func TestValidate_NoTools(t *testing.T) {
	tb, _ := Parse([]byte("version: 1\ntools: {}\n"))
	if errs := Validate(tb); len(errs) == 0 {
		t.Fatal("expected an error for a toolbox with no tools")
	}
}

func TestValidate_MissingServer(t *testing.T) {
	tb, _ := Parse([]byte("version: 1\ntools:\n  a:\n    runtime:\n      type: mcp\n      server: \"\"\n"))
	if errs := Validate(tb); len(errs) == 0 {
		t.Fatal("expected an error for a missing runtime.server")
	}
}

func TestValidate_UnsupportedRuntime(t *testing.T) {
	tb, _ := Parse([]byte("version: 1\ntools:\n  a:\n    runtime:\n      type: wasm\n      server: x\n"))
	if errs := Validate(tb); len(errs) == 0 {
		t.Fatal("expected an error for an unsupported runtime type")
	}
}
