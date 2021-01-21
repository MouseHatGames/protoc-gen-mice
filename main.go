package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/MouseHatGames/protoc-gen-mice/generator"
	"github.com/MouseHatGames/protoc-gen-mice/models"
	"github.com/MouseHatGames/protoc-gen-mice/options"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

const OutputFileExtension = ".pb.mice.go"

func main() {
	inFile := flag.String("input", "", "")
	flag.Parse()

	opts := options.ReadOptions()

	var in []byte
	var err error

	log.SetOutput(os.Stderr)

	if *inFile != "" {
		in, err = ioutil.ReadFile(*inFile)
	} else {
		in, err = ioutil.ReadAll(os.Stdin)
	}

	if err != nil {
		log.Fatalf("failed to read input: %s", err)
	}

	out, err := run(in, opts)
	if err != nil {
		log.Fatalf("failed to run: %s", err)
	}

	os.Stdout.Write(out)
}

func run(in []byte, opts *options.Options) ([]byte, error) {
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(in, req); err != nil {
		return nil, fmt.Errorf("decode input: %w", err)
	}

	gen := &responseGenerator{}

	for _, file := range req.ProtoFile {
		gen.Append(file, opts)
	}

	return proto.Marshal(&gen.resp)
}

type responseGenerator struct {
	resp pluginpb.CodeGeneratorResponse
}

func (g *responseGenerator) error(err string) {
	g.resp.Error = &err
}

func (g *responseGenerator) Append(fdesc *descriptorpb.FileDescriptorProto, opts *options.Options) error {
	if fdesc.Options == nil || fdesc.Options.GoPackage == nil {
		g.error(fmt.Sprintf("file %s is missing go_package option", fdesc.GetName()))
		return nil
	}

	model := models.NewFileFromProto(fdesc, opts)
	if len(model.Services) == 0 {
		return nil
	}

	content := generator.Generate(model)

	// Replace extension with .pb.mice.go
	outname := strings.TrimSuffix(fdesc.GetName(), filepath.Ext(fdesc.GetName())) + OutputFileExtension

	g.resp.File = append(g.resp.File, &pluginpb.CodeGeneratorResponse_File{
		Name:    &outname,
		Content: &content,
	})

	return nil
}
