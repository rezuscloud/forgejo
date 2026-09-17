// specstrip renders templates/swagger/v1_json.tmpl into the committed
// client-go spec format, byte-stably: object key order, indentation,
// escaping and the info.version line all match the existing spec, so a
// re-strip of an unchanged template is a no-op and a changed template
// produces a purely additive diff.
//
// Conventions reproduced (they are the committed spec's, not inventions):
//   - object keys in document order (encoding/json maps would sort them)
//   - 2-space indent, no trailing newline
//   - <, >, & stay literal (Go's HTML escaping is NOT applied)
//   - non-ASCII runes escape as \uXXXX (lowercase, surrogate pairs for
//     astral runes) — Python ensure_ascii style
//   - {{AppVer | JSEscape}} becomes the previous spec's info.version
//     (fallback 16.0.0-dev); every other {{...}} action becomes ""
//
// Usage: go run ./.github/fork/specstrip -out <spec.json> <v1_json.tmpl>
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf16"
)

const defaultVersion = "16.0.0-dev"

// ordered is a JSON object that remembers its key order.
type ordered struct {
	keys []string
	vals map[string]interface{}
}

func main() {
	out := flag.String("out", "", "spec file to write (its existing info.version is reused)")
	flag.Parse()
	if flag.NArg() != 1 || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: specstrip -out <spec.json> <v1_json.tmpl>")
		os.Exit(2)
	}

	tmpl, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fail("read template: %v", err)
	}
	version := previousVersion(*out)
	s := regexp.MustCompile(`\{\{[^}]+\}\}`).ReplaceAllStringFunc(string(tmpl), func(m string) string {
		if strings.Contains(m, "AppVer") {
			return version
		}
		return ""
	})

	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	val, err := decodeValue(dec)
	if err != nil {
		fail("template is not valid JSON after action substitution: %v", err)
	}
	if dec.More() {
		fail("trailing content after JSON document")
	}

	var buf bytes.Buffer
	writeValue(&buf, val, 0)
	if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil {
		fail("write spec: %v", err)
	}
	fmt.Printf("stripped (order-stable, AppVer=%s)\n", version)
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "specstrip: "+format+"\n", args...)
	os.Exit(1)
}

// previousVersion keeps the info.version line stable across re-strips: the
// committed spec is the source of truth for the labelled API version.
func previousVersion(specPath string) string {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return defaultVersion
	}
	var probe struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if json.Unmarshal(raw, &probe) == nil && probe.Info.Version != "" {
		return probe.Info.Version
	}
	return defaultVersion
}

// --- ordered decode (json.Decoder token stream) ---

func decodeValue(dec *json.Decoder) (interface{}, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeFromToken(dec, tok)
}

func decodeFromToken(dec *json.Decoder, tok json.Token) (interface{}, error) {
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			o := &ordered{vals: make(map[string]interface{})}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("non-string object key %v", keyTok)
				}
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				if _, seen := o.vals[key]; !seen {
					o.keys = append(o.keys, key)
				}
				o.vals[key] = val
			}
			if _, err := dec.Token(); err != nil { // closing brace
				return nil, err
			}
			return o, nil
		case '[':
			arr := []interface{}{}
			for dec.More() {
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
			if _, err := dec.Token(); err != nil { // closing bracket
				return nil, err
			}
			return arr, nil
		default:
			return nil, fmt.Errorf("unexpected delim %v", t)
		}
	default:
		return tok, nil // string (already unescaped), json.Number, bool, nil
	}
}

// --- encode (committed spec conventions) ---

func writeValue(buf *bytes.Buffer, v interface{}, depth int) {
	switch t := v.(type) {
	case *ordered:
		if len(t.keys) == 0 {
			buf.WriteString("{}")
			return
		}
		buf.WriteString("{\n")
		for i, k := range t.keys {
			writeIndent(buf, depth+1)
			writeString(buf, k)
			buf.WriteString(": ")
			writeValue(buf, t.vals[k], depth+1)
			if i < len(t.keys)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		writeIndent(buf, depth)
		buf.WriteByte('}')
	case []interface{}:
		if len(t) == 0 {
			buf.WriteString("[]")
			return
		}
		buf.WriteString("[\n")
		for i, e := range t {
			writeIndent(buf, depth+1)
			writeValue(buf, e, depth+1)
			if i < len(t)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		writeIndent(buf, depth)
		buf.WriteByte(']')
	case json.Number:
		buf.WriteString(t.String())
	case string:
		writeString(buf, t)
	case bool:
		if t {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case nil:
		buf.WriteString("null")
	default:
		panic(fmt.Sprintf("specstrip: unencodable %T", v))
	}
}

func writeIndent(buf *bytes.Buffer, depth int) {
	for i := 0; i < depth; i++ {
		buf.WriteString("  ")
	}
}

// writeString escapes like the committed spec: control chars short-form,
// non-ASCII (and astral, via surrogate pairs) as lowercase \uXXXX, while
// <, >, & and all other printable ASCII stay literal.
func writeString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			buf.WriteString(`\"`)
		case r == '\\':
			buf.WriteString(`\\`)
		case r == '\n':
			buf.WriteString(`\n`)
		case r == '\r':
			buf.WriteString(`\r`)
		case r == '\t':
			buf.WriteString(`\t`)
		case r == '\b':
			buf.WriteString(`\b`)
		case r == '\f':
			buf.WriteString(`\f`)
		case r < 0x20:
			fmt.Fprintf(buf, `\u%04x`, r)
		case r < 0x7f:
			buf.WriteByte(byte(r))
		case r <= 0xffff:
			fmt.Fprintf(buf, `\u%04x`, r)
		default:
			hi, lo := utf16.EncodeRune(r)
			fmt.Fprintf(buf, `\u%04x\u%04x`, hi, lo)
		}
	}
	buf.WriteByte('"')
}
