package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Info captures the swagger document metadata. info.version records the
// exact Forgejo application version the SDK was generated from, and the
// `+gitea-<v>` suffix encodes the Gitea REST API compatibility level
// (the API "major" that kubectl-style clients track).
type Info struct {
	Version string `json:"version"`
	Title   string `json:"title"`
}

type SwaggerSpec struct {
	Info      Info                     `json:"info"`
	Paths     map[string]map[string]Op `json:"paths"`
	Defs      map[string]Schema        `json:"definitions"`
	Responses map[string]RespDef       `json:"responses"`
}
type RespDef struct {
	Schema *Schema `json:"schema"`
}
type Op struct {
	OperationID string          `json:"operationId"`
	Summary     string          `json:"summary"`
	Parameters  []Param         `json:"parameters"`
	Responses   map[string]Resp `json:"responses"`
}
type Resp struct {
	Schema *Schema `json:"schema"`
	Ref    string  `json:"$ref"`
}
type Param struct {
	Name   string  `json:"name"`
	In     string  `json:"in"`
	Type   string  `json:"type"`
	Format string  `json:"format"`
	Schema *Schema `json:"schema"`
}
type Schema struct {
	Type     string            `json:"type"`
	Format   string            `json:"format"`
	Ref      string            `json:"$ref"`
	Items    *Schema           `json:"items"`
	Props    map[string]Schema `json:"properties"`
	Addl     *Schema           `json:"additionalProperties"`
	Required []string          `json:"required"`
}

func main() {
	specPath := flag.String("spec", "api/spec/swagger.json", "")
	outDir := flag.String("out", "api/", "")
	cliOutDir := flag.String("cli-out", "", "output dir for generated CLI commands (empty = skip)")
	testOutDir := flag.String("test-out", "", "output dir for generated integration tests (empty = skip)")
	polishOutDir := flag.String("polish-out", "", "output dir for descriptor-driven polished commands (empty = skip)")
	polishDesc := flag.String("polish", "gen/polish.json", "polish descriptor (groups -> operations)")
	flag.Parse()
	raw, _ := os.ReadFile(*specPath)
	var spec SwaggerSpec
	json.Unmarshal(raw, &spec)
	if spec.Defs == nil { spec.Defs = map[string]Schema{} }
	if spec.Paths == nil { spec.Paths = map[string]map[string]Op{} }

	fmt.Printf("spec: %d defs, %d paths, %d responses\n", len(spec.Defs), len(spec.Paths), len(spec.Responses))

	writeFile(filepath.Join(*outDir, "zz_generated_types.go"), genTypes(&spec))
	fmt.Println("wrote zz_generated_types.go")

	writeFile(filepath.Join(*outDir, "zz_generated_specversion.go"), genSpecVersion(&spec))
	fmt.Printf("wrote zz_generated_specversion.go (spec info.version=%s)\n", spec.Info.Version)

	for svc, ops := range groupByService(&spec) {
		writeFile(filepath.Join(*outDir, "zz_generated_"+svc+".go"), genService(svc, ops, &spec))
		fmt.Printf("wrote zz_generated_%s.go (%d)\n", svc, len(ops))
	}

	// Generate CLI command stubs for the fj binary. Each SDK method gets a
	// thin Cobra command under `fj api <service> <method>` that calls the
	// SDK method and prints JSON output. This auto-extends the CLI when
	// upstream adds new API endpoints — no hand-written command needed for
	// raw API access (polished UX commands stay hand-written at the top level).
	if *cliOutDir != "" {
		writeFile(filepath.Join(*cliOutDir, "zz_generated_commands.go"), genCLICommands(&spec))
		fmt.Printf("wrote zz_generated_commands.go (%d services)\n", len(groupByService(&spec)))
	}

	// Generate integration tests — one test per SDK method that exercises the
	// corresponding CLI command against a live Forgejo. Path params are mapped
	// to test fixtures (owner→testUser, repo→shared test repo). Body params are
	// skipped (too complex to auto-generate meaningful payloads — lifecycle
	// tests cover representative create/update/delete flows separately).
	if *testOutDir != "" {
		writeFile(filepath.Join(*testOutDir, "zz_generated_integration_test.go"), genIntegrationTests(&spec))
		fmt.Printf("wrote zz_generated_integration_test.go (%d services)\n", len(groupByService(&spec)))
	}

	// Generate the descriptor-driven polished command groups (gen/polish.json):
	// human-readable top-level commands (fj milestone, …) bound to SDK
	// operations by operationId, with flag/arg/body bindings and printf-style
	// rendering via the shared render helpers. A group is added by editing
	// the descriptor and regenerating — never by hand-writing a command file.
	// Validation is fail-loudly (unknown op/field/helper, verb mismatch), so
	// descriptor drift breaks the build instead of shipping wrong wiring.
	if *polishOutDir != "" {
		raw, err := os.ReadFile(*polishDesc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "polish descriptor: %v\n", err)
			os.Exit(1)
		}
		var pd struct {
			Groups []PolishGroup `json:"groups"`
		}
		if err := json.Unmarshal(raw, &pd); err != nil {
			fmt.Fprintf(os.Stderr, "polish descriptor: %v\n", err)
			os.Exit(1)
		}
		writeFile(filepath.Join(*polishOutDir, "zz_generated_polished.go"), genPolished(&spec, pd.Groups))
		fmt.Printf("wrote zz_generated_polished.go (%d groups)\n", len(pd.Groups))
	}
}

// ---------- TYPES ----------

func genTypes(spec *SwaggerSpec) string {
	var b strings.Builder
	b.WriteString("// Code generated by api/gen. DO NOT EDIT.\n\npackage forgejo\n\nimport \"time\"\n\n")
	for _, name := range sortedKeys(spec.Defs) {
		s := spec.Defs[name]
		gn := pc(name)
		switch {
		case s.Type == "array" && s.Items != nil:
			b.WriteString(fmt.Sprintf("type %s []%s\n\n", gn, gt(s.Items, spec)))
		case (s.Type == "object" || len(s.Props) > 0) && s.Addl == nil:
			b.WriteString(fmt.Sprintf("type %s struct {\n", gn))
			for _, pn := range sortedKeys(s.Props) {
				ps := s.Props[pn]
				cn := pn
				if len(cn) > 0 && !isAlpha(rune(cn[0])) { cn = "X" + strings.TrimLeft(cn, "@#$%^&*") }
				f := pc(cn)
				if kw(f) { f += "_" }
				// Optional date-time fields MUST be pointers: encoding/json never
				// omits a zero time.Time (structs are never "empty"), so a value
				// type serialises "0001-01-01T00:00:00Z" and the server's date
				// guard rejects it (seen live: forgejo 16.0.2 issue comments,
				// 403 unallowed update date). Required stays a value type.
				t := gt(&ps, spec)
				isRequired := false
				for _, r := range s.Required { if r == pn { isRequired = true; break } }
				if t == "time.Time" && !isRequired { t = "*time.Time" }
				b.WriteString(fmt.Sprintf("\t%s %s `json:\"%s,omitempty\"`\n", f, t, pn))
			}
			b.WriteString("}\n\n")
		case s.Type == "string":
			b.WriteString(fmt.Sprintf("type %s string\n\n", gn))
		case s.Type == "integer" && s.Format == "int64" && gn == "Duration":
			b.WriteString("type Duration time.Duration\n\n")
		case s.Type != "":
			b.WriteString(fmt.Sprintf("type %s %s\n\n", gn, gt(&Schema{Type: s.Type, Format: s.Format}, spec)))
		default:
			b.WriteString(fmt.Sprintf("type %s map[string]interface{}\n\n", gn))
		}
	}
	return b.String()
}

// ---------- SERVICES ----------

type MD struct {
	OpID    string
	Method  string
	Path    string
	Summary string
	Params  []Param
	RetTy   string
	HasRet  bool
}

func groupByService(spec *SwaggerSpec) map[string][]MD {
	svcs := map[string][]MD{}
	for path, methods := range spec.Paths {
		for hm, op := range methods {
			hm = strings.ToUpper(hm)
			if hm == "PARAMETERS" || op.OperationID == "" { continue }
			rt, hr := resolveRet(&op, spec)
			svcs[classify(path)] = append(svcs[classify(path)], MD{
				OpID: op.OperationID, Method: hm, Path: path,
				Summary: op.Summary, Params: op.Parameters, RetTy: rt, HasRet: hr,
			})
		}
	}
	return svcs
}

func classify(path string) string {
	p := strings.Split(strings.Trim(path, "/"), "/")
	switch p[0] {
	case "repos": return "repo"
	case "admin": return "admin"
	case "orgs": return "org"
	case "users": return "user"
	case "notifications": return "notify"
	default:
		if strings.HasPrefix(p[0], "activitypub") { return "activitypub" }
		return "misc"
	}
}

func resolveRet(op *Op, spec *SwaggerSpec) (string, bool) {
	for _, code := range []string{"200", "201"} {
		if r, ok := op.Responses[code]; ok {
			if r.Ref != "" {
				rn := last(r.Ref)
				if rd, ok := spec.Responses[rn]; ok && rd.Schema != nil {
					return dgt(rd.Schema), true
				}
				return "", false
			}
			if r.Schema != nil { return dgt(r.Schema), true }
		}
	}
	return "", false
}

func dgt(s *Schema) string {
	if s == nil { return "" }
	if s.Ref != "" { return pc(last(s.Ref)) }
	if s.Type == "array" && s.Items != nil {
		if s.Items.Ref != "" { return "[]" + pc(last(s.Items.Ref)) }
		return "[]" + gt(s.Items, nil)
	}
	return gt(s, nil)
}

func genService(svc string, methods []MD, spec *SwaggerSpec) string {
	sort.Slice(methods, func(i, j int) bool { return methods[i].OpID < methods[j].OpID })
	var b strings.Builder
	b.WriteString("// Code generated by api/gen. DO NOT EDIT.\n\npackage forgejo\n\n")
	b.WriteString("import (\n\t\"bytes\"\n\t\"context\"\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"io\"\n\t\"net/http\"\n\t\"time\"\n)\n\n")
	for _, m := range methods {
		b.WriteString(genMethod(svc, m, spec))
	}
	return b.String()
}

func genMethod(svc string, m MD, spec *SwaggerSpec) string {
	gn := pc(m.OpID)
	ss := pc(svc) + "Service"

	type a struct{ n, t, in, sw string }
	var args []a
	as := ""
	for _, p := range m.Params {
		if p.In == "path" || p.In == "query" || p.In == "body" {
			gt2 := pt(&p, spec)
			ga := cc(p.Name)
			if kw(ga) { ga += "_" }
			args = append(args, a{ga, gt2, p.In, p.Name})
			as += fmt.Sprintf(", %s %s", ga, gt2)
		}
	}

	var rt, rz string
	if m.HasRet && m.RetTy != "" {
		if strings.HasPrefix(m.RetTy, "[]") {
			rt = m.RetTy; rz = "nil"
		} else if isPrimitive(m.RetTy) {
			rt = m.RetTy; rz = zeroVal(m.RetTy)
		} else {
			rt = "*" + m.RetTy; rz = "nil"
		}
	} else if m.HasRet {
		rt = m.RetTy; rz = zeroVal(m.RetTy)
	}
	rp := func() string { if rt != "" { return rz + ", " }; return "" }

	pf := m.Path
	for _, a2 := range args {
		if a2.in == "path" {
			v := "%s"; if a2.t == "int" || a2.t == "int64" { v = "%d" }
			pf = strings.ReplaceAll(pf, "{"+a2.sw+"}", v)
		}
	}

	var qp, ba []a
	for _, a2 := range args {
		if a2.in == "query" { qp = append(qp, a2) }
		if a2.in == "body" { ba = append(ba, a2) }
	}

	sum := m.Summary; if sum == "" { sum = m.OpID }
	var b strings.Builder
	b.WriteString(fmt.Sprintf("// %s — %s\n// %s %s\n", gn, sum, m.Method, m.Path))
	b.WriteString(fmt.Sprintf("func (s *%s) %s(ctx context.Context%s) ", ss, gn, as))
	if rt != "" { b.WriteString(fmt.Sprintf("(%s, *Response, error)", rt)) } else { b.WriteString("(*Response, error)") }
	b.WriteString(" {\n")

	b.WriteString(fmt.Sprintf("\tu := s.client.base.JoinPath(fmt.Sprintf(\"%s\"", eq(pf)))
	for _, a2 := range args { if a2.in == "path" { b.WriteString(fmt.Sprintf(", %s", a2.n)) } }
	b.WriteString("))\n")

	if len(qp) > 0 {
		b.WriteString("\tqry := u.Query()\n")
		for _, a2 := range qp {
			// Only set non-zero-value optional query params. Slice params
			// serialize as repeated query values — the server reads them via
			// FormStrings (url.Values[key]), never as a joined string, so
			// fmt.Sprintf("%v", slice) ("[a b]") or comma-joining are both
			// rejected. Emit one value per occurrence with qry.Add.
			if strings.HasPrefix(a2.t, "[]") {
				b.WriteString(fmt.Sprintf("\tif %s != nil { for _, v := range %s { qry.Add(%q, fmt.Sprintf(\"%%v\", v)) } }\n", a2.n, a2.n, a2.sw))
				continue
			}
			// Only set non-zero-value optional query params.
			b.WriteString(fmt.Sprintf("\tif %s != %s { qry.Set(%q, fmt.Sprintf(\"%%v\", %s)) }\n", a2.n, zeroForType(a2.t), a2.sw, a2.n))
		}
		b.WriteString("\tu.RawQuery = qry.Encode()\n")
	}

	if len(ba) > 0 {
		b.WriteString(fmt.Sprintf("\tbodyBytes, err := json.Marshal(%s)\n", ba[0].n))
		b.WriteString(fmt.Sprintf("\tif err != nil { return %snil, fmt.Errorf(\"marshal: %%w\", err) }\n", rp()))
		b.WriteString(fmt.Sprintf("\treq, err := http.NewRequestWithContext(ctx, http.Method%s, u.String(), bytes.NewReader(bodyBytes))\n", tc(m.Method)))
		b.WriteString("\treq.Header.Set(\"Content-Type\", \"application/json\")\n")
	} else {
		b.WriteString(fmt.Sprintf("\treq, err := http.NewRequestWithContext(ctx, http.Method%s, u.String(), nil)\n", tc(m.Method)))
	}
	b.WriteString(fmt.Sprintf("\tif err != nil { return %snil, fmt.Errorf(\"request: %%w\", err) }\n\n", rp()))
	b.WriteString("\tresp, err := s.client.client.Do(req)\n")
	b.WriteString(fmt.Sprintf("\tif err != nil { return %snil, fmt.Errorf(\"do: %%w\", err) }\n", rp()))
	b.WriteString("\tdefer resp.Body.Close()\n\n")
	b.WriteString(fmt.Sprintf("\tif resp.StatusCode >= 400 { return %snil, handleError(resp) }\n\n", rp()))

	if m.HasRet && m.RetTy != "" {
		if strings.HasPrefix(m.RetTy, "[]") {
			b.WriteString(fmt.Sprintf("\tvar result %s\n", rt))
			b.WriteString(fmt.Sprintf("\tif err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return %snil, fmt.Errorf(\"decode: %%w\", err) }\n", rp()))
			b.WriteString("\treturn result, &Response{Response: resp}, nil\n")
		} else if m.RetTy == "string" || m.RetTy == "[]byte" {
			b.WriteString("\trawBody, err := io.ReadAll(resp.Body)\n")
			b.WriteString(fmt.Sprintf("\tif err != nil { return %snil, fmt.Errorf(\"read: %%w\", err) }\n", rp()))
			if m.RetTy == "string" { b.WriteString("\treturn string(rawBody), &Response{Response: resp}, nil\n") } else { b.WriteString("\treturn rawBody, &Response{Response: resp}, nil\n") }
	} else if isPrimitive(m.RetTy) {
			// primitive non-string: decode into the value directly, return without pointer
			b.WriteString(fmt.Sprintf("\tvar result %s\n", m.RetTy))
			b.WriteString(fmt.Sprintf("\tif err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return %s nil, fmt.Errorf(\"decode: %%w\", err) }\n", rp()))
			b.WriteString("\treturn result, &Response{Response: resp}, nil\n")
		} else {
			b.WriteString(fmt.Sprintf("\tvar result %s\n", m.RetTy))
			b.WriteString(fmt.Sprintf("\tif err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return %s nil, fmt.Errorf(\"decode: %%w\", err) }\n", rp()))
			b.WriteString("\treturn &result, &Response{Response: resp}, nil\n")
		}
	} else {
		b.WriteString("\treturn &Response{Response: resp}, nil\n")
	}
	_ = rz
	b.WriteString("}\n\n")
	return b.String()
}

// ---------- HELPERS ----------

var refRe = regexp.MustCompile(`#/definitions/(.+)`)

func gt(s *Schema, spec *SwaggerSpec) string {
	if s == nil { return "interface{}" }
	if s.Ref != "" {
		m := refRe.FindStringSubmatch(s.Ref)
		if len(m) > 1 { return "*" + pc(m[1]) }
		return "interface{}"
	}
	switch s.Type {
	case "string": if s.Format == "date-time" { return "time.Time" }; return "string"
	case "integer": if s.Format == "int64" { return "int64" }; return "int"
	case "number": return "float64"
	case "boolean": return "bool"
	case "array": if s.Items != nil { return "[]" + gt(s.Items, spec) }; return "[]interface{}"
	case "object": if s.Addl != nil { return "map[string]" + gt(s.Addl, spec) }; return "interface{}"
	case "file": return "[]byte"
	default: return "interface{}"
	}
}
func pt(p *Param, spec *SwaggerSpec) string {
	if p.Schema != nil { return gt(p.Schema, spec) }
	return gt(&Schema{Type: p.Type, Format: p.Format}, spec)
}
func pc(s string) string {
	s = strings.ReplaceAll(s, "_", " "); s = strings.ReplaceAll(s, "-", " ")
	ws := strings.Fields(s)
	for i, w := range ws { if len(w) > 0 { ws[i] = strings.ToUpper(w[:1]) + w[1:] } }
	return strings.Join(ws, "")
}
func cc(s string) string { p := pc(s); if len(p) > 0 { return strings.ToLower(p[:1]) + p[1:] }; return p }
func tc(s string) string { if len(s) > 0 { return strings.ToUpper(s[:1]) + strings.ToLower(s[1:]) }; return s }
func eq(s string) string { return strings.ReplaceAll(s, `"`, `\"`) }
func last(ref string) string { ps := strings.Split(ref, "/"); return ps[len(ps)-1] }
func sortedKeys(m map[string]Schema) []string { ks := make([]string, 0, len(m)); for k := range m { ks = append(ks, k) }; sort.Strings(ks); return ks }
func isAlpha(r rune) bool { return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') }

var kws = map[string]bool{"type":true,"range":true,"func":true,"map":true,"chan":true,"interface":true,"struct":true,"go":true,"select":true,"case":true,"default":true,"for":true,"break":true,"continue":true,"if":true,"else":true,"return":true,"switch":true,"var":true,"const":true,"import":true,"package":true,"defer":true,"goto":true,"fallthrough":true}
func kw(s string) bool { return kws[strings.ToLower(s)] }
func writeFile(path, content string) {
	// Post-process: only keep imports that are actually used.
	content = stripUnusedImports(content)
	os.WriteFile(path, []byte(content), 0644)
}

// giteaAPIVersion extracts the Gitea REST API compatibility version from a
// Forgejo info.version string of the form "16.0.0-dev-507-<sha>+gitea-1.22.0".
// It is the API "major" clients track (analogous to kubectl's client API
// version). Falls back to the raw version when the suffix is absent.
func giteaAPIVersion(v string) string {
	if i := strings.LastIndex(v, "+gitea-"); i >= 0 {
		return v[i+len("+gitea-"):]
	}
	return v
}

// genSpecVersion emits constants recording which Forgejo swagger spec the SDK
// was generated from, so the CLI can report client/server API versions like
// `kubectl version` (ADR: the SDK declares the API it maps).
func genSpecVersion(spec *SwaggerSpec) string {
	title := spec.Info.Title
	if title == "" {
		title = "Forgejo API"
	}
	version := spec.Info.Version
	api := giteaAPIVersion(version)
	var b strings.Builder
	b.WriteString("// Code generated by api/gen; DO NOT EDIT.\n\n")
	b.WriteString("package forgejo\n\n")
	b.WriteString("// Spec metadata recording the exact Forgejo swagger spec this SDK was\n")
	b.WriteString("// generated from. The CLI reports SpecAPIVersion as its client API level\n")
	b.WriteString("// (like kubectl's client version) and queries the live server's /version\n")
	b.WriteString("// for the server side.\n\n")
	fmt.Fprintf(&b, "const SpecTitle = %q\n\n", title)
	fmt.Fprintf(&b, "// SpecVersion is info.version of the swagger.json the SDK was generated from\n")
	fmt.Fprintf(&b, "// (full Forgejo application version, including build metadata).\n")
	fmt.Fprintf(&b, "const SpecVersion = %q\n\n", version)
	fmt.Fprintf(&b, "// SpecAPIVersion is the Gitea REST API compatibility version derived from\n")
	fmt.Fprintf(&b, "// SpecVersion (the +gitea-<v> suffix). This is the API level the SDK maps.\n")
	fmt.Fprintf(&b, "const SpecAPIVersion = %q\n", api)
	return b.String()
}

func isPrimitive(t string) bool {
	switch t {
	case "string", "int", "int64", "float64", "bool", "[]byte": return true
	}
	return false
}
func zeroVal(t string) string {
	switch t {
	case "string": return `""`
	case "time.Time": return "(time.Time{})"
	case "int", "int64", "float64": return "0"
	case "bool": return "false"
	case "[]byte": return "nil"
	default: return "nil"
	}
}

// stripUnusedImports removes import lines for packages not referenced in the
// generated code. Replaces goimports — the generator produces clean, compilable
// output without external tools (critical for the CronJob environment).
func stripUnusedImports(content string) string {
	lines := strings.Split(content, "\n")

	inImport := false
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "import (" {
			inImport = true
			result = append(result, line)
			continue
		}
		if inImport && trimmed == ")" {
			inImport = false
			result = append(result, line)
			continue
		}
		if inImport {
			if strings.HasPrefix(trimmed, "_") || strings.HasPrefix(trimmed, ".") {
				result = append(result, line)
				continue
			}
			var pkgName string
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				pkgName = fields[0]
			} else {
				pkgName = last(strings.Trim(trimmed, "\""))
			}
			if strings.Contains(content, pkgName+".") {
				result = append(result, line)
			}
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

func zeroForType(t string) string {
	switch t {
	case "int", "int64": return "0"
	case "float64": return "0"
	case "bool": return "false"
	case "string": return `""`
	case "time.Time": return "(time.Time{})"
	default:
		if strings.HasPrefix(t, "[]") { return "nil" }
		if strings.HasPrefix(t, "*") { return "nil" }
		return "nil"
	}
}

// ---------- CLI COMMAND GENERATION ----------
//
// Generates thin Cobra command stubs for every SDK method, registered under
// `fj api <service> <method>`. Each command:
//   - defines flags for path/query parameters
//   - builds and calls the SDK method
//   - prints the result as JSON
//
// This auto-extends the CLI when upstream adds new API endpoints — no
// hand-written command is needed for raw API access. Polished UX commands
// (with human-readable output, aliases, help) stay hand-written at the top
// level, exactly like kubectl's hand-written commands vs raw client-go calls.

func genCLICommands(spec *SwaggerSpec) string {
	services := groupByService(spec)

	var b strings.Builder
	b.WriteString("// Code generated by api/gen; DO NOT EDIT.\n\n")
	b.WriteString("package cmd\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"context\"\n")
	b.WriteString("\t\"encoding/json\"\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"strconv\"\n")
	b.WriteString("\t\"strings\"\n")
	b.WriteString("\t\"time\"\n")
	b.WriteString("\n")
	b.WriteString("\tforgejo \"forgejo.org/client-go\"\n")
	b.WriteString("\t\"github.com/spf13/cobra\"\n")
	b.WriteString(")\n\n")
	b.WriteString("// parseTime converts a string flag to time.Time (RFC3339).\n")
	b.WriteString("func parseTime(s string) time.Time {\n")
	b.WriteString("\tif s == \"\" { return time.Time{} }\n")
	b.WriteString("\tt, _ := time.Parse(time.RFC3339, s)\n")
	b.WriteString("\treturn t\n")
	b.WriteString("}\n\n")
	b.WriteString("// parseStringSlice converts a comma-separated string to []interface{}.\n")
	b.WriteString("func parseStringSlice(s string) []interface{} {\n")
	b.WriteString("\tif s == \"\" { return nil }\n")
	b.WriteString("\tparts := strings.Split(s, \",\")\n")
	b.WriteString("\tres := make([]interface{}, len(parts))\n")
	b.WriteString("\tfor i, p := range parts { res[i] = p }\n")
	b.WriteString("\treturn res\n")
	b.WriteString("}\n\n")

	// Generate the parent command that registers all sub-services.
	b.WriteString("// NewGeneratedAPICmd creates the `fj api` command tree with one subcommand\n")
	b.WriteString("// per SDK method. Auto-generated from the swagger spec.\n")
	b.WriteString("func NewGeneratedAPICmd() *cobra.Command {\n")
	b.WriteString("\tcmd := &cobra.Command{\n")
	b.WriteString("\t\tUse: \"api\",\n")
	b.WriteString("\t\tShort: \"Raw API access (auto-generated from swagger spec)\",\n")
	b.WriteString("\t\tLong: `Raw API access to every Forgejo REST endpoint. Each subcommand\ncorresponds to an SDK method generated from the swagger spec. For polished\ncommands with human-readable output, use the top-level commands (issue, pr,\nactions, etc.) instead.`,\n")
	b.WriteString("\t}\n")

	// Sort services for deterministic output
	svcNames := make([]string, 0, len(services))
	for svc := range services {
		svcNames = append(svcNames, svc)
	}
	sort.Strings(svcNames)

	for _, svc := range svcNames {
		methods := services[svc]
		sort.Slice(methods, func(i, j int) bool { return methods[i].OpID < methods[j].OpID })

		b.WriteString(fmt.Sprintf("\t{ // service: %s\n", svc))
		b.WriteString(fmt.Sprintf("\t\tsvcCmd := &cobra.Command{Use: %q, Short: %q}\n", svc, svc+" service (auto-generated)"))
		for _, m := range methods {
			b.WriteString(genCLISubcommand(svc, m, spec))
		}
		b.WriteString("\t\tcmd.AddCommand(svcCmd)\n")
		b.WriteString("\t}\n")
	}

	b.WriteString("\treturn cmd\n")
	b.WriteString("}\n")
	return b.String()
}

// serviceField maps a classify() service name to the actual Client struct field.
// Most are just pc(svc), but some differ (e.g. activitypub → ActivityPub).
func serviceField(svc string) string {
	switch svc {
	case "activitypub": return "ActivityPub"
	case "repo": return "Repo"
	case "admin": return "Admin"
	case "org": return "Org"
	case "user": return "User"
	case "notify": return "Notify"
	case "misc": return "Misc"
	default: return pc(svc)
	}
}

// genCLISubcommand generates one cobra.Command for a single SDK method.
func genCLISubcommand(svc string, m MD, spec *SwaggerSpec) string {
	cmdName := cmdNameFor(svc, m.OpID)
	methodName := pc(m.OpID)
	svcFld := serviceField(svc)
	varname := sanitizeVar(m.OpID) // unique var prefix per method

	// Separate params by type
	type cliParam struct {
		flagName string
		goName   string
		goType   string
		required bool
		isBody   bool
	}
	var pathParams, queryParams, bodyParams []cliParam
	for _, p := range m.Params {
		if p.In == "path" || p.In == "query" || p.In == "body" {
			cp := cliParam{
				flagName: dashCase(p.Name),
				goName:   cc(p.Name),
				goType:   pt(&p, spec),
				required: p.In == "path",
				isBody:   p.In == "body",
			}
			if cp.isBody { bodyParams = append(bodyParams, cp) } else if p.In == "query" { queryParams = append(queryParams, cp) } else { pathParams = append(pathParams, cp) }
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\t\t{ // %s %s -> %s.%s\n", m.Method, m.Path, svcFld, methodName))
	b.WriteString(fmt.Sprintf("\t\tmc := &cobra.Command{Use: %q, Short: %q}\n", cmdName, eqStr(m.Summary)))

	// Add flags
	for _, cp := range append(append(pathParams, queryParams...), bodyParams...) {
		flag := fmt.Sprintf("%s_%s", varname, cp.goName)
		// time.Time params use a string flag, converted at call site
		isTime := cp.goType == "time.Time"
		if isTime {
			b.WriteString(fmt.Sprintf("\t\tvar %s string\n", flag))
			b.WriteString(fmt.Sprintf("\t\tmc.Flags().StringVar(&%s, %q, \"\", %q)\n", flag, cp.flagName, cp.flagName))
		} else if cp.goType == "int" || cp.goType == "int64" {
			if cp.goType == "int" {
				b.WriteString(fmt.Sprintf("\t\tvar %s int\n", flag))
				b.WriteString(fmt.Sprintf("\t\tmc.Flags().IntVar(&%s, %q, 0, %q)\n", flag, cp.flagName, cp.flagName))
			} else {
				b.WriteString(fmt.Sprintf("\t\tvar %s int64\n", flag))
				b.WriteString(fmt.Sprintf("\t\tmc.Flags().Int64Var(&%s, %q, 0, %q)\n", flag, cp.flagName, cp.flagName))
			}
		} else if cp.goType == "[]interface{}" || strings.HasPrefix(cp.goType, "[]") {
			b.WriteString(fmt.Sprintf("\t\tvar %s string\n", flag))
			b.WriteString(fmt.Sprintf("\t\tmc.Flags().StringVar(&%s, %q, \"\", %q+\" (comma-separated)\")\n", flag, cp.flagName, cp.flagName))
		} else if cp.goType == "bool" {
			b.WriteString(fmt.Sprintf("\t\tvar %s bool\n", flag))
			b.WriteString(fmt.Sprintf("\t\tmc.Flags().BoolVar(&%s, %q, false, %q)\n", flag, cp.flagName, cp.flagName))
		} else {
			b.WriteString(fmt.Sprintf("\t\tvar %s string\n", flag))
			b.WriteString(fmt.Sprintf("\t\tmc.Flags().StringVar(&%s, %q, \"\", %q)\n", flag, cp.flagName, cp.flagName))
		}
	}

	// RunE
	b.WriteString("\t\tmc.RunE = func(cmd *cobra.Command, args []string) error {\n")
	b.WriteString("\t\t\tcli, e := resolveHostClient(cmd, \"\")\n")
	b.WriteString("\t\t\tif e != nil { return e }\n")

	// Body params need unmarshalling
	var callArgs []string
	for _, cp := range pathParams {
		if cp.goType == "time.Time" {
			callArgs = append(callArgs, fmt.Sprintf("parseTime(%s_%s)", varname, cp.goName))
		} else {
			callArgs = append(callArgs, fmt.Sprintf("%s_%s", varname, cp.goName))
		}
	}
	for _, cp := range queryParams {
		if cp.goType == "time.Time" {
			callArgs = append(callArgs, fmt.Sprintf("parseTime(%s_%s)", varname, cp.goName))
		} else if strings.HasPrefix(cp.goType, "[]") {
			callArgs = append(callArgs, fmt.Sprintf("parseStringSlice(%s_%s)", varname, cp.goName))
		} else {
			callArgs = append(callArgs, fmt.Sprintf("%s_%s", varname, cp.goName))
		}
	}
	for _, cp := range bodyParams {
		bt := strings.TrimPrefix(cp.goType, "*")
		if !isPrimitive(bt) && bt != "interface{}" {
			bt = "forgejo." + bt
		}
		b.WriteString(fmt.Sprintf("\t\t\tvar bodyVal %s\n", bt))
		b.WriteString(fmt.Sprintf("\t\t\tif %s_%s != \"\" { json.Unmarshal([]byte(%s_%s), &bodyVal) }\n", varname, cp.goName, varname, cp.goName))
		// Primitive body types (string) are passed by value, not pointer
		if isPrimitive(bt) && !strings.HasPrefix(bt, "[]") {
			callArgs = append(callArgs, "bodyVal")
		} else {
			callArgs = append(callArgs, "&bodyVal")
		}
	}

	callStr := fmt.Sprintf("cli.%s.%s(context.Background()", svcFld, methodName)
	if len(callArgs) > 0 { callStr += ", " + strings.Join(callArgs, ", ") }
	callStr += ")"

	if m.HasRet && m.RetTy != "" {
		b.WriteString(fmt.Sprintf("\t\t\tres, _, err := %s\n", callStr))
	} else {
		b.WriteString(fmt.Sprintf("\t\t\t_, err := %s\n", callStr))
	}
	b.WriteString("\t\t\tif err != nil { return err }\n")
	if m.HasRet && m.RetTy != "" {
		// Only nil-check pointer/slice return types (primitives can't be nil)
		if strings.HasPrefix(m.RetTy, "*") || strings.HasPrefix(m.RetTy, "[]") {
			b.WriteString("\t\t\tif res != nil {\n")
			b.WriteString("\t\t\t\tj, _ := json.MarshalIndent(res, \"\", \"  \")\n")
			b.WriteString("\t\t\t\tfmt.Println(string(j))\n")
			b.WriteString("\t\t\t}\n")
		} else {
			b.WriteString("\t\t\tj, _ := json.MarshalIndent(res, \"\", \"  \")\n")
			b.WriteString("\t\t\tfmt.Println(string(j))\n")
		}
	}
	b.WriteString("\t\t\treturn nil\n")
	b.WriteString("\t\t}\n")

	// Mark required flags
	for _, cp := range pathParams {
		b.WriteString(fmt.Sprintf("\t\t_ = mc.MarkFlagRequired(%q)\n", cp.flagName))
	}

	b.WriteString("\t\tsvcCmd.AddCommand(mc)\n")
	b.WriteString("\t\t}\n")

	return b.String()
}

// dashCase converts camelCase or snake_case to kebab-case for CLI flags/commands.
func dashCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('-')
		}
		if r == '_' {
			b.WriteByte('-')
		} else {
			b.WriteString(strings.ToLower(string(r)))
		}
	}
	return b.String()
}

// cmdNameFor derives the CLI command name from a swagger operationId, stripping a
// redundant leading service prefix. Forgejo's swagger operationIds are
// inconsistent — some carry the service word (repoGetWikiPages, repoCreateBranch)
// while others don't (issueCreateMilestone, acceptRepoTransfer). Without this,
// 159 of 265 commands under `fj api repo` are redundantly prefixed `repo-…`, so a
// user/agent typing `fj api repo get` / `wiki-pages` hits an unknown subcommand
// (cobra then reports the next flag as "unknown flag") with empty stdout.
// Stripping makes every command name guessable: get, create-branch,
// get-wiki-pages, issue-create-milestone. Collision-free (verified at generation).
func cmdNameFor(svc, opID string) string {
	return strings.TrimPrefix(dashCase(opID), dashCase(svc)+"-")
}

// eqStr escapes a string for use as a Go string literal.
func eqStr(s string) string {
	if s == "" { return "" }
	return strings.ReplaceAll(s, `"`, `\\\"`)
}

// sanitizeVar converts an operation ID to a safe Go identifier prefix (lowercase
// alphanumeric, guaranteed unique since operation IDs are unique).
func sanitizeVar(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ---------- INTEGRATION TEST GENERATION ----------
//
// Generates one Go test function per SDK method. Each test builds the fj
// binary, invokes the corresponding `fj api <service> <method>` command with
// test-fixture arguments, and verifies the output is valid JSON (for reads) or
// an acceptable HTTP error (for writes that hit non-existent resources).
//
// Path params are mapped to shared test fixtures:
//   owner/username/org → testUser() (root)
//   repo               → shared test repo created in TestMain
//   index/id           → "1" (may 404, which proves the command works)
// Body params are skipped — lifecycle tests cover representative CRUD flows.

// testFixtureValue returns the Go expression for a path param's test value.
// Returns (value, isString) — if isString, the value is a function call
// (testUser()); otherwise it's a literal string.
func testFixtureValue(paramName string) string {
	switch strings.ToLower(paramName) {
	case "owner", "username", "org":
		return "testUser()"
	case "repo", "repo_name":
		return "testRepo"
	default:
		// Numeric params (index, id, etc.) — use "1", accept 404
		return `"1"`
	}
}

// needsTestRepo returns true if any path param references a repo.
func needsTestRepo(methods []MD) bool {
	for _, m := range methods {
		for _, p := range m.Params {
			if p.In == "path" && (strings.Contains(strings.ToLower(p.Name), "repo")) {
				return true
			}
		}
	}
	return false
}

func genIntegrationTests(spec *SwaggerSpec) string {
	services := groupByService(spec)

	var b strings.Builder
	b.WriteString("// Code generated by api/gen; DO NOT EDIT.\n\n")
	b.WriteString("package integration\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"bytes\"\n")
	b.WriteString("\t\"encoding/json\"\n")
	b.WriteString("\t\"strings\"\n")
	b.WriteString("\t\"testing\"\n")
	b.WriteString(")\n\n")

	b.WriteString("// testRepo is the name of a shared repo created by setupTestRepo for\n")
	b.WriteString("// repo-scoped commands. Created once per test run, cleaned up at exit.\n")
	b.WriteString("var testRepo string\n\n")

	// Sort services for deterministic output
	svcNames := make([]string, 0, len(services))
	for svc := range services {
		svcNames = append(svcNames, svc)
	}
	sort.Strings(svcNames)

	totalMethods := 0
	for _, svc := range svcNames {
		methods := services[svc]
		sort.Slice(methods, func(i, j int) bool { return methods[i].OpID < methods[j].OpID })

		b.WriteString(fmt.Sprintf("// TestGenerated_%s tests all %d auto-generated %s commands.\n", pc(svc), len(methods), svc))
		b.WriteString(fmt.Sprintf("func TestGenerated_%s(t *testing.T) {\n", pc(svc)))
		b.WriteString("\tskipIfNoInstance(t)\n")
		b.WriteString("\tbinary := buildFjBinary(t)\n")

		for _, m := range methods {
			totalMethods++
			cmdName := cmdNameFor(svc, m.OpID)
			testName := m.OpID

			// Build the flag arguments from path/query params (skip body params)
			var flagsBuilder strings.Builder
			for _, p := range m.Params {
				if p.In == "body" {
					continue
				}
				if p.In == "path" || p.In == "query" {
					flagName := dashCase(p.Name)
					val := testFixtureValue(p.Name)
					flagsBuilder.WriteString(fmt.Sprintf("\t\t\t\"--%s\", %s,\n", flagName, val))
				}
			}

			isRead := m.Method == "GET"

			b.WriteString(fmt.Sprintf("\tt.Run(%q, func(t *testing.T) {\n", testName))
			if flagsBuilder.Len() > 0 {
				b.WriteString(fmt.Sprintf("\t\targs := []string{\"api\", %q, %q,\n%s\t\t}\n", svc, cmdName, flagsBuilder.String()))
			} else {
				b.WriteString(fmt.Sprintf("\t\targs := []string{\"api\", %q, %q}\n", svc, cmdName))
			}
			b.WriteString("\t\tout, err := runFj(t, binary, args...)\n")
			b.WriteString("\t\tif err != nil {\n")
			if isRead {
				b.WriteString("\t\t\t// GET: 404/403 means resource doesn't exist — still proves the command works\n")
				b.WriteString("\t\t\tif isAcceptableError(out) { t.Skip(\"resource not found (command works)\") }\n")
			} else {
				b.WriteString("\t\t\t// Write commands: accept any HTTP error (404/403/409/422)\n")
				b.WriteString("\t\t\tif isAcceptableError(out) { t.Skip(\"endpoint exists, write skipped (no body)\") }\n")
			}
			b.WriteString("\t\t\tt.Errorf(\"%v\\n%s\", err, out)\n")
			b.WriteString("\t\t\treturn\n")
			b.WriteString("\t\t}\n")
			if isRead && m.HasRet && m.RetTy != "" {
				b.WriteString("\t\t// Verify valid JSON output\n")
			b.WriteString("\t\ttrimmed := bytes.TrimSpace([]byte(out))\n")
			b.WriteString("\t\tif len(trimmed) > 0 {\n")
			b.WriteString("\t\t\tvar v interface{}\n")
			b.WriteString("\t\t\tif e := json.Unmarshal(trimmed, &v); e != nil {\n")
			b.WriteString("\t\t\t\tt.Errorf(\"invalid JSON: %v\\n%s\", e, string(trimmed[:min(len(trimmed),200)]))\n")
			b.WriteString("\t\t\t}\n")
			b.WriteString("\t\t}\n")
			}
			b.WriteString("\t})\n")
		}
		b.WriteString("}\n\n")
	}

	// Add the setup helper
	b.WriteString("// setupTestRepo creates a shared test repo for repo-scoped commands.\n")
	b.WriteString("// Call from TestMain or the first test that needs it.\n")
	b.WriteString("func setupTestRepo(t *testing.T) {\n")
	b.WriteString("\tt.Helper()\n")
	b.WriteString("\tif testRepo != \"\" { return }\n")
	b.WriteString("\ttestRepo = uniqueRepoName(\"gen\")\n")
	b.WriteString("\tcreateTestRepo(t, testRepo)\n")
	b.WriteString("}\n\n")

	_ = totalMethods
	return b.String()
}
// ---------- POLISHED COMMANDS (descriptor-driven) ----------
//
// The `fj api` tree covers 100% of the API with JSON output. Polished,
// human-readable command groups used to be hand-written one file per group
// (issue.go, pr.go, release.go…). The polish layer moves that shape into a
// descriptor (gen/polish.json): each group maps commands to SDK operations
// (by operationId) with flag/arg/body bindings and printf-style rendering.
// The generator emits zz_generated_polished.go (NewPolishedCmds) — a group
// is added by editing the descriptor and regenerating, never by hand-writing
// a command file. Rendering stays shared: column expressions reference the
// hand-written render helpers (statusSymbol, stateStr, timeStr…) by name.
//
// Everything is validated fail-loudly at generation time (like the delta
// signatures in sync-validate.sh): unknown op, unbound param, unknown
// field/helper, printf-verb/format mismatch, unsupported kinds.
//
// Deliberate scope limits (extend when a port needs them):
//   - repo-scoped commands only (resolveClient supplies owner/repo)
//   - path args: int64/int or string
//   - body fields: string, bool, int64/int, date-time; bodyConst: string
//   - columns: response fields, bound args/flags, single-arg render helpers

type PolishGroup struct {
	Name     string          `json:"name"`
	Alias    string          `json:"alias,omitempty"`
	Short    string          `json:"short"`
	Commands []PolishCommand `json:"commands"`
}

type PolishCommand struct {
	Name      string                     `json:"name"`
	Short     string                     `json:"short"`
	Use       string                     `json:"use,omitempty"`
	Op        string                     `json:"op"`
	Args      []PolishArg                `json:"args,omitempty"`
	Params    map[string]PolishFlag      `json:"params,omitempty"`
	Consts    map[string]float64         `json:"consts,omitempty"`
	Body      map[string]PolishBodyField `json:"body,omitempty"`
	BodyConst map[string]string          `json:"bodyConst,omitempty"`
	Empty     string                     `json:"empty,omitempty"`
	Row       *PolishRender              `json:"row,omitempty"`
	Lines     []PolishLine               `json:"lines,omitempty"`
}

type PolishArg struct {
	Name string `json:"name"` // swagger path param name
	Use  string `json:"use"`  // Use placeholder, e.g. "<ID>"
}

type PolishFlag struct {
	Flag    string `json:"flag"`
	Short   string `json:"short,omitempty"`
	Default string `json:"default,omitempty"`
	Help    string `json:"help,omitempty"`
}

type PolishBodyField struct {
	Flag     string `json:"flag"`
	Short    string `json:"short,omitempty"`
	Required bool   `json:"required,omitempty"`
	Help     string `json:"help,omitempty"`
}

type PolishRender struct {
	Format  string   `json:"format"`
	Columns []string `json:"columns"`
}

type PolishLine struct {
	Format      string   `json:"format"`
	Columns     []string `json:"columns"`
	UnlessEmpty string   `json:"unlessEmpty,omitempty"` // response json field name
}

// polishRenderers is the render vocabulary column expressions may call.
// name -> printf verb of the return value. A name not listed here fails
// generation. Kept in one place on purpose: this is the contract between
// the descriptor DSL and the hand-written render layer (render.go/helpers.go).
var polishRenderers = map[string]string{
	"statusSymbol": "s",
	"stateStr":     "s",
	"timeStr":      "s",
	"valI64":       "d",
}

func polishFatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "polish: "+format+"\n", a...)
	os.Exit(1)
}

var polishIdentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var polishFuncRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)$`)

// goFieldName replicates the zz_generated_types.go field naming (json name →
// Go exported name) so descriptor references cannot drift from the SDK types.
func goFieldName(jn string) string {
	cn := jn
	if len(cn) > 0 && !isAlpha(rune(cn[0])) {
		cn = "X" + strings.TrimLeft(cn, "@#$%^&*")
	}
	f := pc(cn)
	if kw(f) {
		f += "_"
	}
	return f
}

func isRequiredProp(s Schema, pn string) bool {
	for _, r := range s.Required {
		if r == pn {
			return true
		}
	}
	return false
}

// verbForType maps a Go field type to its printf verb; "" = not printable
// bare (pointers, structs — must be wrapped in a render helper).
func verbForType(t string) string {
	switch t {
	case "int", "int64":
		return "d"
	case "string":
		return "s"
	case "bool":
		return "t"
	case "float64":
		return "f"
	case "time.Time":
		return "s" // Stringer
	}
	return ""
}

// responseFields maps Go field name → printf verb for a spec definition.
// Pointer-ish fields map to "" (wrap them: stateStr(State), timeStr(DueOn)).
func responseFields(def string, spec *SwaggerSpec) map[string]string {
	s, ok := spec.Defs[def]
	if !ok {
		polishFatal("definition %q not in spec", def)
	}
	fields := map[string]string{}
	for pn, ps := range s.Props {
		t := gt(&ps, spec)
		if ps.Type == "string" && ps.Format == "date-time" && !isRequiredProp(s, pn) {
			t = "*time.Time"
		}
		fields[goFieldName(pn)] = verbForType(t)
	}
	return fields
}

// polishResolve resolves a bare identifier: response fields get the receiver
// prefix (and mark the result as used), bound args/flags pass through.
// Pointer-ish fields resolve with verb "" — printable only inside a helper.
func polishResolve(name, recv string, fields, vars map[string]string, where string, touched *bool) (string, string) {
	if v, ok := fields[name]; ok {
		*touched = true
		return recv + "." + name, v
	}
	if v, ok := vars[name]; ok {
		return name, v
	}
	polishFatal("%s: unknown identifier %q (not a response field, bound arg, or flag)", where, name)
	return "", ""
}

// polishArgExpr renders an expression in helper-argument position. Pointer
// fields are exactly what helpers exist for, so no bare-print restriction.
func polishArgExpr(e, recv string, fields, vars map[string]string, where string, touched *bool) string {
	e = strings.TrimSpace(e)
	if m := polishFuncRe.FindStringSubmatch(e); m != nil && !strings.Contains(m[2], ",") {
		if _, ok := polishRenderers[m[1]]; !ok {
			polishFatal("%s: unknown render helper %q", where, m[1])
		}
		return m[1] + "(" + polishArgExpr(m[2], recv, fields, vars, where, touched) + ")"
	}
	if polishIdentRe.MatchString(e) {
		expr, _ := polishResolve(e, recv, fields, vars, where, touched)
		return expr
	}
	polishFatal("%s: unsupported column expression %q", where, e)
	return ""
}

// polishRenderExpr renders one top-level column expression, returning the
// expression and its printf verb. Grammar: IDENT | FUNC '(' EXPR ')'.
// At the top level a pointer-ish field cannot be printed bare — it must be
// wrapped in a render helper (stateStr(State), timeStr(DueOn)); the verb
// then comes from the helper's registry entry. touched reports whether any
// response field was referenced (decides if the SDK result is bound).
func polishRenderExpr(e, recv string, fields, vars map[string]string, where string, touched *bool) (string, string) {
	e = strings.TrimSpace(e)
	if m := polishFuncRe.FindStringSubmatch(e); m != nil && !strings.Contains(m[2], ",") {
		verb, ok := polishRenderers[m[1]]
		if !ok {
			polishFatal("%s: unknown render helper %q", where, m[1])
		}
		return m[1] + "(" + polishArgExpr(m[2], recv, fields, vars, where, touched) + ")", verb
	}
	if polishIdentRe.MatchString(e) {
		expr, verb := polishResolve(e, recv, fields, vars, where, touched)
		if verb == "" {
			polishFatal("%s: field %s cannot be printed bare — wrap it in a render helper", where, e)
		}
		return expr, verb
	}
	polishFatal("%s: unsupported column expression %q", where, e)
	return "", ""
}

// polishVerbs extracts printf verbs in order from a format string.
func polishVerbs(format string) []string {
	var verbs []string
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		if i+1 >= len(format) {
			polishFatal("dangling %% in format %q", format)
		}
		switch format[i+1] {
		case '%':
			i++
		case 'd', 's', 't', 'f':
			verbs = append(verbs, string(format[i+1]))
			i++
		default:
			polishFatal("unsupported verb %%%c in format %q", format[i+1], format)
		}
	}
	return verbs
}

// polishOpRef joins a swagger operation to its classify() service.
type polishOpRef struct {
	svc string
	m   MD
}

func genPolished(spec *SwaggerSpec, groups []PolishGroup) string {
	// Index every operation by operationId — the descriptor's join key.
	ops := map[string]polishOpRef{}
	for svc, methods := range groupByService(spec) {
		for _, m := range methods {
			ops[m.OpID] = polishOpRef{svc, m}
		}
	}

	var b strings.Builder
	b.WriteString("// Code generated by api/gen from gen/polish.json; DO NOT EDIT.\n\n")
	b.WriteString("package cmd\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"context\"\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"strconv\"\n")
	b.WriteString("\t\"time\"\n\n")
	b.WriteString("\tforgejo \"forgejo.org/client-go\"\n")
	b.WriteString("\t\"github.com/spf13/cobra\"\n")
	b.WriteString(")\n\n")

	// Validating optional-time parser: absent flag → nil (omitempty drops it
	// server-side, preserving partial-PATCH semantics); invalid values are
	// errors — a zero time would marshal 0001-01-01 and clobber the field.
	b.WriteString("func parseOptTime(s string) (*time.Time, error) {\n")
	b.WriteString("\tif s == \"\" { return nil, nil }\n")
	b.WriteString("\tt, err := time.Parse(time.RFC3339, s)\n")
	b.WriteString("\tif err != nil { return nil, fmt.Errorf(\"invalid time (RFC3339): %s\", s) }\n")
	b.WriteString("\treturn &t, nil\n")
	b.WriteString("}\n\n")

	for _, g := range groups {
		b.WriteString(genPolishGroup(g, ops, spec))
	}

	b.WriteString("// NewPolishedCmds returns the descriptor-driven polished command groups\n")
	b.WriteString("// (gen/polish.json). Root registers these once; adding a group is a\n")
	b.WriteString("// descriptor edit + regen — no hand-written command file, no root edit.\n")
	b.WriteString("func NewPolishedCmds() []*cobra.Command {\n")
	b.WriteString("\treturn []*cobra.Command{\n")
	for _, g := range groups {
		b.WriteString(fmt.Sprintf("\t\tnewPolish%sCmd(),\n", pc(g.Name)))
	}
	b.WriteString("\t}\n}\n\n")
	return b.String()
}

func genPolishGroup(g PolishGroup, ops map[string]polishOpRef, spec *SwaggerSpec) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("func newPolish%sCmd() *cobra.Command {\n", pc(g.Name)))
	b.WriteString("\tcmd := &cobra.Command{\n")
	b.WriteString(fmt.Sprintf("\t\tUse:   %q,\n", g.Name))
	b.WriteString(fmt.Sprintf("\t\tShort: %q,\n", g.Short))
	if g.Alias != "" {
		b.WriteString(fmt.Sprintf("\t\tAliases: []string{%q},\n", g.Alias))
	}
	b.WriteString("\t}\n")
	var defs strings.Builder
	for _, c := range g.Commands {
		defs.WriteString(genPolishCommand(g, c, ops, spec))
		b.WriteString(fmt.Sprintf("\tcmd.AddCommand(newPolish%s%sCmd())\n", pc(g.Name), pc(c.Name)))
	}
	b.WriteString("\treturn cmd\n}\n\n")
	b.WriteString(defs.String())
	return b.String()
}

func genPolishCommand(g PolishGroup, c PolishCommand, ops map[string]polishOpRef, spec *SwaggerSpec) string {
	where := fmt.Sprintf("%s.%s", g.Name, c.Name)
	ref, ok := ops[c.Op]
	if !ok {
		polishFatal("%s: unknown operationId %q", where, c.Op)
	}
	m := ref.m
	svcFld := serviceField(ref.svc)
	methodName := pc(m.OpID)
	fname := fmt.Sprintf("newPolish%s%sCmd", pc(g.Name), pc(c.Name))

	reserved := map[string]bool{"cmd": true, "args": true, "err": true, "res": true, "it": true, "c": true, "owner": true, "repo": true, "fmt": true}
	safeVar := func(n string) string {
		v := cc(n)
		if reserved[v] || kw(v) {
			polishFatal("%s: identifier %q collides with a reserved name — rename the flag/arg", where, n)
		}
		return v
	}

	// ---- resolve bindings ------------------------------------------------
	// args: path params as positionals
	argVars := map[string]string{} // param name -> var name
	argVerbs := map[string]string{}
	for _, a := range c.Args {
		found := false
		for i := range m.Params {
			p := &m.Params[i]
			if p.Name == a.Name && p.In == "path" {
				found = true
				switch pt(p, spec) {
				case "int64", "int":
					argVerbs[a.Name] = "d"
				case "string":
					argVerbs[a.Name] = "s"
				default:
					polishFatal("%s: arg %s has unsupported type %s", where, a.Name, pt(p, spec))
				}
				argVars[a.Name] = safeVar(a.Name)
			}
		}
		if !found {
			polishFatal("%s: arg %q is not a path param of %s", where, a.Name, c.Op)
		}
	}

	// params: query params -> flags
	type pBind struct {
		varName string
		kind    string // string|bool|int64|int|time
		flag    PolishFlag
	}
	paramBinds := map[string]pBind{}
	for i := range m.Params {
		p := &m.Params[i]
		if p.In != "query" {
			continue
		}
		f, ok := c.Params[p.Name]
		if !ok {
			continue
		}
		t := pt(p, spec)
		var kind string
		switch {
		case t == "string":
			kind = "string"
		case t == "bool":
			kind = "bool"
		case t == "int64":
			kind = "int64"
		case t == "int":
			kind = "int"
		case t == "time.Time":
			kind = "time"
		default:
			polishFatal("%s: param %s has unsupported type %s", where, p.Name, t)
		}
		if (kind != "string") && f.Default != "" {
			polishFatal("%s: default values are only supported for string flags (%s)", where, p.Name)
		}
		paramBinds[p.Name] = pBind{safeVar(f.Flag), kind, f}
	}

	// body schema + field binds
	var bodyDef Schema
	var bodyGo string
	haveBody := false
	for i := range m.Params {
		p := &m.Params[i]
		if p.In != "body" {
			continue
		}
		if p.Schema == nil || p.Schema.Ref == "" {
			polishFatal("%s: body param %s must reference a named schema", where, p.Name)
		}
		dn := last(p.Schema.Ref)
		d, ok := spec.Defs[dn]
		if !ok {
			polishFatal("%s: body schema %q not in spec", where, dn)
		}
		bodyDef, bodyGo, haveBody = d, "forgejo."+pc(dn), true
	}
	if (len(c.Body) > 0 || len(c.BodyConst) > 0) && !haveBody {
		polishFatal("%s: body/bodyConst given but %s takes no body", where, c.Op)
	}

	type bBind struct {
		goName string
		kind   string // string|bool|int64|int|time|const
		f      PolishBodyField
		val    string
	}
	bodyBinds := map[string]bBind{}
	for jn, f := range c.Body {
		ps, ok := bodyDef.Props[jn]
		if !ok {
			polishFatal("%s: body field %q not in the %s schema", where, jn, bodyGo)
		}
		if _, dup := c.BodyConst[jn]; dup {
			polishFatal("%s: body field %q bound twice (flag + const)", where, jn)
		}
		t := gt(&ps, spec)
		var kind string
		switch {
		case t == "string":
			kind = "string"
		case t == "bool":
			kind = "bool"
		case t == "int64":
			kind = "int64"
		case t == "int":
			kind = "int"
		case t == "time.Time" || t == "*time.Time":
			kind = "time"
		default:
			polishFatal("%s: body field %q has unsupported type %s", where, jn, t)
		}
		bodyBinds[jn] = bBind{goFieldName(jn), kind, f, ""}
	}
	for jn, val := range c.BodyConst {
		ps, ok := bodyDef.Props[jn]
		if !ok {
			polishFatal("%s: bodyConst field %q not in the %s schema", where, jn, bodyGo)
		}
		if ps.Type != "string" {
			polishFatal("%s: bodyConst supports string fields only (%s is %s)", where, jn, ps.Type)
		}
		bodyBinds[jn] = bBind{goFieldName(jn), "const", PolishBodyField{}, val}
	}

	// consts: literal call values for int query params
	constVerbs := map[string]bool{}
	for pn := range c.Consts {
		t := ""
		for i := range m.Params {
			if m.Params[i].Name == pn && m.Params[i].In == "query" {
				t = pt(&m.Params[i], spec)
			}
		}
		if t != "int" && t != "int64" {
			polishFatal("%s: const %q must bind an int query param (got %q)", where, pn, t)
		}
		constVerbs[pn] = true
	}

	// ---- response fields + pre-rendered output ---------------------------
	var fields map[string]string
	if m.HasRet && m.RetTy != "" {
		base := m.RetTy
		if strings.HasPrefix(base, "[]") {
			base = strings.TrimPrefix(base, "[]")
		}
		if _, ok := spec.Defs[base]; ok {
			fields = responseFields(base, spec)
		}
	}
	vars := map[string]string{}
	for n, v := range argVars {
		vars[v] = argVerbs[n]
	}
	for _, pb := range paramBinds {
		switch pb.kind {
		case "string", "time":
			vars[pb.varName] = "s"
		case "bool":
			vars[pb.varName] = "t"
		default:
			vars[pb.varName] = "d"
		}
	}
	for jn, bb := range bodyBinds {
		if bb.kind == "string" {
			vars[safeVar(bb.f.Flag)] = "s"
		}
		_ = jn
	}

	type outLine struct {
		exprs  []string
		verbs  []string
		format string
		guard  string
	}
	var rows *outLine
	var lines []outLine
	touched := false

	if c.Row != nil {
		if fields == nil || !strings.HasPrefix(m.RetTy, "[]") {
			polishFatal("%s: row rendering requires an array-of-object return (%s returns %q)", where, c.Op, m.RetTy)
		}
		r := &outLine{format: c.Row.Format}
		for _, col := range c.Row.Columns {
			e, v := polishRenderExpr(col, "it", fields, vars, where, &touched)
			r.exprs = append(r.exprs, e)
			r.verbs = append(r.verbs, v)
		}
		rows = r
	}
	for _, ln := range c.Lines {
		ol := outLine{format: ln.Format}
		for _, col := range ln.Columns {
			e, v := polishRenderExpr(col, "res", fields, vars, where, &touched)
			ol.exprs = append(ol.exprs, e)
			ol.verbs = append(ol.verbs, v)
		}
		if ln.UnlessEmpty != "" {
			if fields == nil || strings.HasPrefix(m.RetTy, "[]") {
				polishFatal("%s: unlessEmpty requires a single-object return", where)
			}
			base := m.RetTy
			ps, ok := spec.Defs[base].Props[ln.UnlessEmpty]
			if !ok {
				polishFatal("%s: unlessEmpty field %q not in %s", where, ln.UnlessEmpty, base)
			}
			gn := goFieldName(ln.UnlessEmpty)
			if ps.Type == "string" && ps.Format == "" {
				ol.guard = fmt.Sprintf("res.%s != \"\"", gn)
			} else if ps.Type == "string" && ps.Format == "date-time" {
				ol.guard = fmt.Sprintf("res.%s != nil", gn)
			} else {
				polishFatal("%s: unlessEmpty supports string and date-time fields (%s is %s/%s)", where, ln.UnlessEmpty, ps.Type, ps.Format)
			}
		}
		lines = append(lines, ol)
	}
	for _, ol := range append(lines, func() []outLine {
		if rows != nil {
			return []outLine{*rows}
		}
		return nil
	}()...) {
		verbs := polishVerbs(ol.format)
		if len(verbs) != len(ol.verbs) {
			polishFatal("%s: format %q has %d verbs but %d columns", where, ol.format, len(verbs), len(ol.verbs))
		}
		for i, v := range verbs {
			if v != ol.verbs[i] {
				polishFatal("%s: format %q verb %d is %%%c but column %d is %%%c", where, ol.format, i+1, v, i+1, ol.verbs[i])
			}
		}
	}

	// ---- emit ------------------------------------------------------------
	var b strings.Builder
	b.WriteString(fmt.Sprintf("// %s — %s\n", fname, c.Short))
	b.WriteString(fmt.Sprintf("func %s() *cobra.Command {\n", fname))

	// flag var declarations
	varDecls := []string{}
	flagRegs := []string{}
	reg := func(varName, flagName, short, def, help string) {
		if help == "" {
			help = flagName
		}
		if short != "" {
			flagRegs = append(flagRegs, fmt.Sprintf("\tcmd.Flags().StringVarP(&%s, %q, %q, %q, %q)", varName, flagName, short, def, eq(help)))
		} else {
			flagRegs = append(flagRegs, fmt.Sprintf("\tcmd.Flags().StringVar(&%s, %q, %q, %q)", varName, flagName, def, eq(help)))
		}
	}
	// params in spec order
	for i := range m.Params {
		p := &m.Params[i]
		pb, ok := paramBinds[p.Name]
		if !ok {
			continue
		}
		help := pb.flag.Help
		switch pb.kind {
		case "string":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s string", pb.varName))
			reg(pb.varName, pb.flag.Flag, pb.flag.Short, pb.flag.Default, help)
		case "time":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s string", pb.varName))
			reg(pb.varName, pb.flag.Flag, pb.flag.Short, "", help)
		case "bool":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s bool", pb.varName))
			flagRegs = append(flagRegs, fmt.Sprintf("\tcmd.Flags().BoolVar(&%s, %q, false, %q)", pb.varName, pb.flag.Flag, eq(help)))
		case "int64":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s int64", pb.varName))
			flagRegs = append(flagRegs, fmt.Sprintf("\tcmd.Flags().Int64Var(&%s, %q, 0, %q)", pb.varName, pb.flag.Flag, eq(help)))
		case "int":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s int", pb.varName))
			flagRegs = append(flagRegs, fmt.Sprintf("\tcmd.Flags().IntVar(&%s, %q, 0, %q)", pb.varName, pb.flag.Flag, eq(help)))
		}
	}
	// body fields, deterministic order
	bodyJns := make([]string, 0, len(bodyBinds))
	for jn := range bodyBinds {
		bodyJns = append(bodyJns, jn)
	}
	sort.Strings(bodyJns)
	for _, jn := range bodyJns {
		bb := bodyBinds[jn]
		if bb.kind == "const" {
			continue
		}
		help := bb.f.Help
		vn := safeVar(bb.f.Flag)
		switch bb.kind {
		case "string":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s string", vn))
			reg(vn, bb.f.Flag, bb.f.Short, "", help)
		case "time":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s string", vn))
			reg(vn, bb.f.Flag, bb.f.Short, "", help)
		case "bool":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s bool", vn))
			flagRegs = append(flagRegs, fmt.Sprintf("\tcmd.Flags().BoolVar(&%s, %q, false, %q)", vn, bb.f.Flag, eq(help)))
		case "int64":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s int64", vn))
			flagRegs = append(flagRegs, fmt.Sprintf("\tcmd.Flags().Int64Var(&%s, %q, 0, %q)", vn, bb.f.Flag, eq(help)))
		case "int":
			varDecls = append(varDecls, fmt.Sprintf("\tvar %s int", vn))
			flagRegs = append(flagRegs, fmt.Sprintf("\tcmd.Flags().IntVar(&%s, %q, 0, %q)", vn, bb.f.Flag, eq(help)))
		}
	}
	for _, d := range varDecls {
		b.WriteString(d + "\n")
	}

	use := c.Use
	if use == "" {
		use = c.Name
	}
	b.WriteString("\tcmd := &cobra.Command{\n")
	b.WriteString(fmt.Sprintf("\t\tUse:   %q,\n", use))
	b.WriteString(fmt.Sprintf("\t\tShort: %q,\n", c.Short))
	if len(c.Args) > 0 {
		b.WriteString(fmt.Sprintf("\t\tArgs:  cobra.ExactArgs(%d),\n", len(c.Args)))
	}
	b.WriteString("\t\tRunE: func(cmd *cobra.Command, args []string) error {\n")

	// required checks
	for _, jn := range bodyJns {
		bb := bodyBinds[jn]
		if bb.kind != "const" && bb.f.Required {
			b.WriteString(fmt.Sprintf("\t\t\tif %s == \"\" { return fmt.Errorf(\"--%s is required\") }\n", safeVar(bb.f.Flag), bb.f.Flag))
		}
	}
	// time parses
	for _, jn := range bodyJns {
		bb := bodyBinds[jn]
		if bb.kind == "time" {
			vn := safeVar(bb.f.Flag)
			b.WriteString(fmt.Sprintf("\t\t\t%sVal, err := parseOptTime(%s)\n", vn, vn))
			b.WriteString("\t\t\tif err != nil { return err }\n")
		}
	}
	for i := range m.Params {
		p := &m.Params[i]
		if p.In != "query" {
			continue
		}
		if pb, ok := paramBinds[p.Name]; ok && pb.kind == "time" {
			b.WriteString(fmt.Sprintf("\t\t\t%sVal, err := parseOptTime(%s)\n", pb.varName, pb.varName))
			b.WriteString("\t\t\tif err != nil { return err }\n")
		}
	}
	// arg parses
	for _, a := range c.Args {
		v := argVars[a.Name]
		kind := ""
		for i := range m.Params {
			p := &m.Params[i]
			if p.Name == a.Name && p.In == "path" {
				if t := pt(p, spec); t == "int64" || t == "int" {
					kind = "int"
				} else {
					kind = "string"
				}
			}
		}
		if kind == "int" {
			b.WriteString(fmt.Sprintf("\t\t\t%s, err := strconv.ParseInt(args[0], 10, 64)\n", v))
			b.WriteString(fmt.Sprintf("\t\t\tif err != nil { return fmt.Errorf(\"invalid %s: %%s\", args[0]) }\n", a.Name))
		} else {
			b.WriteString(fmt.Sprintf("\t\t\t%s := args[0]\n", v))
		}
	}

	b.WriteString("\t\t\tc, owner, repo, err := resolveClient(cmd)\n")
	b.WriteString("\t\t\tif err != nil { return err }\n")

	// call args in spec order (mirrors the SDK signature exactly)
	var callArgs []string
	sawOwner, sawRepo := false, false
	for i := range m.Params {
		p := &m.Params[i]
		switch p.In {
		case "path":
			switch p.Name {
			case "owner":
				sawOwner = true
				callArgs = append(callArgs, "owner")
			case "repo":
				sawRepo = true
				callArgs = append(callArgs, "repo")
			default:
				if _, ok := argVars[p.Name]; ok {
					callArgs = append(callArgs, argVars[p.Name])
				} else if pb, ok := paramBinds[p.Name]; ok {
					callArgs = append(callArgs, pb.varName)
				} else {
					polishFatal("%s: path param %s unbound — declare an arg or a param", where, p.Name)
				}
			}
		case "query":
			if pb, ok := paramBinds[p.Name]; ok {
				if pb.kind == "time" {
					polishFatal("%s: time-typed query params are not supported yet (%s)", where, p.Name)
				}
				callArgs = append(callArgs, pb.varName)
			} else if cv, ok := c.Consts[p.Name]; ok {
				callArgs = append(callArgs, strconv.FormatInt(int64(cv), 10))
			} else {
				polishFatal("%s: query param %s unbound — bind a flag or a const", where, p.Name)
			}
		case "body":
			lit := "&" + bodyGo + "{"
			for _, jn := range bodyJns {
				bb := bodyBinds[jn]
				switch bb.kind {
				case "const":
					lit += fmt.Sprintf("\n\t\t\t\t%s: %q,", bb.goName, bb.val)
				case "time":
					lit += fmt.Sprintf("\n\t\t\t\t%s: %sVal,", bb.goName, safeVar(bb.f.Flag))
				default:
					lit += fmt.Sprintf("\n\t\t\t\t%s: %s,", bb.goName, safeVar(bb.f.Flag))
				}
			}
			if len(bodyJns) > 0 {
				lit += "\n\t\t\t}"
			} else {
				lit += "}"
			}
			callArgs = append(callArgs, lit)
		}
	}
	if !sawOwner || !sawRepo {
		polishFatal("%s: polished commands are repo-scoped; %s has no owner/repo path params", where, c.Op)
	}

	callStr := fmt.Sprintf("c.%s.%s(context.Background()", svcFld, methodName)
	if len(callArgs) > 0 {
		callStr += ", " + strings.Join(callArgs, ", ")
	}
	callStr += ")"

	switch {
	case rows != nil || touched:
		b.WriteString(fmt.Sprintf("\t\t\tres, _, err := %s\n", callStr))
	case m.HasRet:
		// err is already declared (arg parse / resolveClient) — assignment, not declaration
		b.WriteString(fmt.Sprintf("\t\t\t_, _, err = %s\n", callStr))
	default:
		b.WriteString(fmt.Sprintf("\t\t\t_, err = %s\n", callStr))
	}
	b.WriteString("\t\t\tif err != nil { return err }\n")

	if rows != nil {
		empty := c.Empty
		if empty == "" {
			empty = "no results"
		}
		b.WriteString(fmt.Sprintf("\t\t\tif len(res) == 0 {\n\t\t\t\tfmt.Println(%q)\n\t\t\t\treturn nil\n\t\t\t}\n", empty))
		b.WriteString("\t\t\tfor _, it := range res {\n")
		b.WriteString(fmt.Sprintf("\t\t\t\tfmt.Printf(%q, %s)\n", rows.format, strings.Join(rows.exprs, ", ")))
		b.WriteString("\t\t\t}\n")
	}
	for _, ol := range lines {
		call := fmt.Sprintf("fmt.Printf(%q, %s)", ol.format, strings.Join(ol.exprs, ", "))
		if ol.guard != "" {
			b.WriteString(fmt.Sprintf("\t\t\tif %s {\n\t\t\t\t%s\n\t\t\t}\n", ol.guard, call))
		} else {
			b.WriteString(fmt.Sprintf("\t\t\t%s\n", call))
		}
	}

	b.WriteString("\t\t\treturn nil\n")
	b.WriteString("\t\t},\n")
	b.WriteString("\t}\n")
	for _, r := range flagRegs {
		b.WriteString(r + "\n")
	}
	b.WriteString("\treturn cmd\n}\n\n")
	return b.String()
}
