package models

import (
	"path/filepath"
	"strings"

	"github.com/MouseHatGames/protoc-gen-mice/options"
	"google.golang.org/protobuf/types/descriptorpb"
)

type File struct {
	Services  []*Service
	GoPackage string
	Package   string
}

type Service struct {
	Name     string
	UglyName string
	Methods  []*Method
}

type Method struct {
	Name    string
	InType  string
	OutType string
}

func NewFileFromProto(desc *descriptorpb.FileDescriptorProto, opts *options.Options) *File {
	file := &File{
		GoPackage: getPackageName(*desc.GetOptions().GoPackage),
		Package:   desc.GetPackage(),
	}

	for _, svc := range desc.Service {
		fname := strings.TrimSuffix(desc.GetName(), filepath.Ext(desc.GetName()))
		fname = strings.TrimPrefix(fname, opts.FilePrefix)

		file.Services = append(file.Services, file.newServiceFromProto(svc, fname))
	}

	return file
}

func (f *File) newServiceFromProto(desc *descriptorpb.ServiceDescriptorProto, fileName string) *Service {
	svc := &Service{
		Name:     desc.GetName(),
		UglyName: fileName,
	}

	for _, met := range desc.Method {
		svc.Methods = append(svc.Methods, f.newMethodFromProto(met))
	}

	return svc
}

func (f *File) newMethodFromProto(desc *descriptorpb.MethodDescriptorProto) *Method {
	return &Method{
		Name:    desc.GetName(),
		InType:  f.getGoType(desc.GetInputType()),
		OutType: f.getGoType(desc.GetOutputType()),
	}
}

func (f *File) getGoType(t string) string {
	if t[0] == '.' {
		t = t[1:]
	}

	if f.Package != "" && strings.HasPrefix(t, f.Package+".") {
		return t[len(f.Package)+1:]
	}

	return t
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
