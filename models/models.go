package models

import (
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
)

type File struct {
	Services  []*Service
	GoPackage string
	Package   string
}

type Service struct {
	Name    string
	Methods []*Method
}

type Method struct {
	Name    string
	InType  string
	OutType string
}

func NewFileFromProto(desc *descriptorpb.FileDescriptorProto) *File {
	file := &File{
		GoPackage: getPackageName(*desc.GetOptions().GoPackage),
		Package:   desc.GetPackage(),
	}

	for _, svc := range desc.Service {
		file.Services = append(file.Services, file.newServiceFromProto(svc))
	}

	return file
}

func (f *File) newServiceFromProto(desc *descriptorpb.ServiceDescriptorProto) *Service {
	svc := &Service{
		Name: desc.GetName(),
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
	if strings.HasPrefix(t, ".") {
		return t[1:]
	}
	if f.Package != "" && strings.HasPrefix(t, f.Package) {
		return t[len(f.Package):]
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
