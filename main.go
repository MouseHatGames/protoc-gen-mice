package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

const OutputFileExtension = ".pb.mice.go"

func main() {
	inFile := flag.String("input", "", "")
	flag.Parse()

	var in []byte
	var err error

	log.SetOutput(os.Stderr)

	if *inFile != "" {
		in, err = ioutil.ReadFile(*inFile)
	} else {
		in, err = ioutil.ReadAll(os.Stdin)
		ioutil.WriteFile("data.bin", in, 0)
	}

	if err != nil {
		log.Fatalf("failed to read input: %s", err)
	}

	out, err := run(in)
	if err != nil {
		log.Fatalf("failed to run: %s", err)
	}

	os.Stdout.Write(out)
}

func run(in []byte) ([]byte, error) {
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(in, req); err != nil {
		return nil, fmt.Errorf("decode input: %w", err)
	}

	gen := &generator{}

	for _, file := range req.ProtoFile {
		gen.Generate(file)
	}

	return proto.Marshal(&gen.resp)
}

type generator struct {
	resp pluginpb.CodeGeneratorResponse
}

func (g *generator) error(err string) {
	g.resp.Error = &err
}

func (g *generator) Generate(fdesc *descriptorpb.FileDescriptorProto) error {
	if fdesc.Options == nil || fdesc.Options.GoPackage == nil {
		g.error(fmt.Sprintf("file %s is missing go_package option", *fdesc.Name))
		return nil
	}

	content := &strings.Builder{}
	gen := &fileGenerator{
		gopkg: getPackageName(fdesc.Options.GetGoPackage()),
		pkg:   fdesc.GetPackage(),
		w:     content,
	}

	outname := strings.TrimSuffix(*fdesc.Name, filepath.Ext(*fdesc.Name)) + OutputFileExtension
	file := pluginpb.CodeGeneratorResponse_File{
		Name: &outname,
	}
	g.resp.File = append(g.resp.File, &file)

	fmt.Fprintf(gen.w, `package %s

import (
	"context"

	"github.com/MouseHatGames/mice/server"
	"github.com/MouseHatGames/mice/client"
)

`, gen.gopkg)

	for _, s := range fdesc.Service {
		gen.writeServiceInterface(s)

		gen.writeServiceClient(s)

		fmt.Fprintln(gen.w)

		gen.writeRegisterFunction(s)
	}

	str := content.String()
	file.Content = &str
	return nil
}

type fileGenerator struct {
	w     io.Writer
	pkg   string
	gopkg string
}

func getPackageName(pkg string) string {
	idx := strings.IndexRune(pkg, ';')
	if idx == -1 {
		idx = strings.LastIndexFunc(pkg, func(r rune) bool { return r == '/' })
	}
	if idx == -1 {
		return pkg
	}

	return pkg[idx+1:]
}

func (f *fileGenerator) writeServiceInterface(svc *descriptorpb.ServiceDescriptorProto) error {
	fmt.Fprintf(f.w, "type %s interface {\n", svc.GetName())

	for _, m := range svc.Method {
		fmt.Fprint(f.w, "\t")
		f.writeMethodDefinition(m)
		fmt.Fprint(f.w, "\n")
	}

	fmt.Fprintln(f.w, "}")

	return nil
}

func (f *fileGenerator) writeMethodDefinition(m *descriptorpb.MethodDescriptorProto) error {
	inType := f.getGoType(m.GetInputType())
	outType := f.getGoType(m.GetOutputType())

	fmt.Fprintf(f.w, "%s(ctx context.Context, req *%s) (*%s, error)", m.GetName(), inType, outType)
	return nil
}

func (f *fileGenerator) writeRegisterFunction(svc *descriptorpb.ServiceDescriptorProto) error {
	fmt.Fprintf(f.w, "func Register%sHandler(srv server.Server, handler %s) {\n", svc.GetName(), svc.GetName())

	fmt.Fprint(f.w, "\tsrv.AddHandler(handler, ")

	for i, m := range svc.Method {
		fmt.Fprintf(f.w, `"%s"`, m.GetName())

		if i < len(svc.Method)-1 {
			fmt.Fprint(f.w, ", ")
		}
	}

	fmt.Fprintln(f.w, ")\n}")

	return nil
}

func (f *fileGenerator) writeServiceClient(svc *descriptorpb.ServiceDescriptorProto) error {
	fmt.Fprintf(f.w, `type impl%s struct {
	c client.Client
	s string
}

`, svc.GetName())

	for _, m := range svc.Method {
		fmt.Fprintf(f.w, `func (c *impl%s) `, svc.GetName())
		f.writeMethodDefinition(m)

		fmt.Fprintf(f.w, ` {
	resp := &%s{}
	if err := c.c.Call(c.s, "%s", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

`, f.getGoType(m.GetOutputType()), m.GetName())
	}

	fmt.Fprintf(f.w, `func New%sClient(svc string, cl client.Client) %s {
	return &impl%s{
		c: cl,
		s: svc,
	}
}`, svc.GetName(), svc.GetName(), svc.GetName())

	return nil
}

func (f *fileGenerator) getGoType(t string) string {
	if strings.HasPrefix(t, ".") {
		return t[1:]
	}
	if f.pkg != "" && strings.HasPrefix(t, f.pkg) {
		return t[len(f.pkg):]
	}

	return t
}
