//go:build ignore

// protorename moves the protobuf definitions of a vendored module into their
// own namespace. Used by sync-todaycore.sh.
//
// TodayCore carries the same .proto files as sing-box. Go's protobuf runtime
// keeps one global registry of files and message names and panics at start-up
// when two packages register the same ones, so linking both cores needs
// TodayCore's copies renamed: file "daemon/x.proto" becomes
// "todaycore/daemon/x.proto", package "daemon" becomes "todaycore.daemon".
//
// The names live inside the serialized descriptor embedded in every .pb.go, so
// a text replace would corrupt it (the strings are length-prefixed). This tool
// decodes the descriptor, renames, and encodes it again.
//
// Usage: go run scripts/protorename/main.go <dir> <prefix> <old-module> <new-module>
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "usage: protorename <dir> <prefix> <old-module> <new-module>")
		os.Exit(2)
	}
	dir, prefix, oldMod, newMod := os.Args[1], os.Args[2], os.Args[3], os.Args[4]

	packages := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".pb.go") || strings.HasSuffix(path, "_grpc.pb.go") {
			return err
		}
		pkgs, err := renameFile(path, prefix, oldMod, newMod)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for _, p := range pkgs {
			packages[p] = true
		}
		return nil
	})
	if err == nil {
		err = renameGRPC(dir, prefix, packages)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// renameFile rewrites every raw descriptor constant in one .pb.go file and
// returns the proto packages it renamed.
func renameFile(path, prefix, oldMod, newMod string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return nil, err
	}

	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	var renamed []string

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			if len(vs.Names) != 1 || !strings.HasSuffix(vs.Names[0].Name, "_rawDesc") || len(vs.Values) != 1 {
				continue
			}
			raw, err := concatLiterals(vs.Values[0])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", vs.Names[0].Name, err)
			}
			fd := &descriptorpb.FileDescriptorProto{}
			if err := proto.Unmarshal([]byte(raw), fd); err != nil {
				return nil, fmt.Errorf("%s: decode descriptor: %w", vs.Names[0].Name, err)
			}
			renamed = append(renamed, fd.GetPackage())
			renameDescriptor(fd, prefix, oldMod, newMod)
			out, err := proto.MarshalOptions{Deterministic: true}.Marshal(fd)
			if err != nil {
				return nil, err
			}
			edits = append(edits, edit{
				start: fset.Position(vs.Values[0].Pos()).Offset,
				end:   fset.Position(vs.Values[0].End()).Offset,
				text:  strconv.Quote(string(out)),
			})
		}
	}
	if len(edits) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	last := 0
	for _, e := range edits {
		buf.Write(src[last:e.start])
		buf.WriteString(e.text)
		last = e.end
	}
	buf.Write(src[last:])
	return renamed, os.WriteFile(path, buf.Bytes(), 0o644)
}

// concatLiterals evaluates a constant built as "a" + "b" + ... .
func concatLiterals(e ast.Expr) (string, error) {
	switch v := e.(type) {
	case *ast.BasicLit:
		return strconv.Unquote(v.Value)
	case *ast.BinaryExpr:
		a, err := concatLiterals(v.X)
		if err != nil {
			return "", err
		}
		b, err := concatLiterals(v.Y)
		return a + b, err
	case *ast.ParenExpr:
		return concatLiterals(v.X)
	}
	return "", fmt.Errorf("unexpected expression %T", e)
}

// wellKnown reports protobuf's own files and types, which stay shared.
func wellKnown(name string) bool {
	return strings.HasPrefix(name, "google/protobuf/") || strings.HasPrefix(name, ".google.protobuf.")
}

func renameDescriptor(fd *descriptorpb.FileDescriptorProto, prefix, oldMod, newMod string) {
	fd.Name = proto.String(prefix + "/" + fd.GetName())
	if fd.Package != nil {
		fd.Package = proto.String(prefix + "." + fd.GetPackage())
	}
	for i, dep := range fd.Dependency {
		if !wellKnown(dep) {
			fd.Dependency[i] = prefix + "/" + dep
		}
	}
	if o := fd.Options; o != nil && o.GoPackage != nil {
		o.GoPackage = proto.String(strings.Replace(o.GetGoPackage(), oldMod, newMod, 1))
	}
	ref := func(s *string) *string {
		if s == nil || !strings.HasPrefix(*s, ".") || wellKnown(*s) {
			return s
		}
		return proto.String("." + prefix + *s)
	}
	var fixMessage func(m *descriptorpb.DescriptorProto)
	fixMessage = func(m *descriptorpb.DescriptorProto) {
		for _, f := range m.Field {
			f.TypeName = ref(f.TypeName)
			f.Extendee = ref(f.Extendee)
		}
		for _, f := range m.Extension {
			f.TypeName = ref(f.TypeName)
			f.Extendee = ref(f.Extendee)
		}
		for _, n := range m.NestedType {
			fixMessage(n)
		}
	}
	for _, m := range fd.MessageType {
		fixMessage(m)
	}
	for _, f := range fd.Extension {
		f.TypeName = ref(f.TypeName)
		f.Extendee = ref(f.Extendee)
	}
	for _, s := range fd.Service {
		for _, m := range s.Method {
			m.InputType = ref(m.InputType)
			m.OutputType = ref(m.OutputType)
		}
	}
}

// renameGRPC updates the service names gRPC stubs carry as plain strings, so
// they match the renamed descriptors.
func renameGRPC(dir, prefix string, packages map[string]bool) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_grpc.pb.go") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out := string(src)
		for pkg := range packages {
			if pkg == "" {
				continue
			}
			out = strings.ReplaceAll(out, `"`+pkg+`.`, `"`+prefix+"."+pkg+".")
			out = strings.ReplaceAll(out, `"/`+pkg+`.`, `"/`+prefix+"."+pkg+".")
			out = strings.ReplaceAll(out, `Metadata: "`+pkg+`/`, `Metadata: "`+prefix+"/"+pkg+"/")
		}
		return os.WriteFile(path, []byte(out), 0o644)
	})
}
