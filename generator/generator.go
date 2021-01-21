package generator

import (
	"fmt"
	"io"
	"strings"

	"github.com/MouseHatGames/protoc-gen-mice/models"
)

type generator struct {
	w io.Writer
	f *models.File
}

func Generate(file *models.File) string {
	str := &strings.Builder{}
	gen := &generator{
		w: str,
		f: file,
	}

	gen.Write()

	return str.String()
}

func (g *generator) Write() {
	fmt.Fprintf(g.w, `package %s

import (
	"context"

	"github.com/MouseHatGames/mice/server"
	"github.com/MouseHatGames/mice/client"
)

`, g.f.GoPackage)

	for _, svc := range g.f.Services {
		g.writeServiceInterface(svc, true)
		g.writeServiceInterface(svc, false)
		g.writeServiceClient(svc)
		g.writeRegisterFunction(svc)
	}
}

func (g *generator) writeServiceInterface(svc *models.Service, server bool) {
	if server {
		fmt.Fprintf(g.w, "type %sServer interface {\n", svc.Name)
	} else {
		fmt.Fprintf(g.w, "type %s interface {\n", svc.Name)
	}

	for _, m := range svc.Methods {
		fmt.Fprint(g.w, "\t")
		g.writeMethodDefinition(m, server)
		fmt.Fprint(g.w, "\n")
	}

	fmt.Fprintln(g.w, "}")
}

func (g *generator) writeMethodDefinition(m *models.Method, server bool) {
	if server {
		fmt.Fprintf(g.w, "%s(ctx context.Context, req *%s, resp *%s) error", m.Name, m.InType, m.OutType)
	} else {
		fmt.Fprintf(g.w, "%s(ctx context.Context, req *%s) (*%s, error)", m.Name, m.InType, m.OutType)
	}
}

func (g *generator) writeServiceClient(svc *models.Service) {
	fmt.Fprintf(g.w, `type impl%s struct {
	c client.Client
	s string
}

`, svc.Name)

	for _, m := range svc.Methods {
		fmt.Fprintf(g.w, `func (c *impl%s) `, svc.Name)
		g.writeMethodDefinition(m, false)

		fmt.Fprintf(g.w, ` {
	resp := new(%s)
	if err := c.c.Call(c.s, "%s", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

`, m.OutType, m.Name)
	}

	fmt.Fprintf(g.w, `func New%sClient(cl client.Client) %s {
	return &impl%s{
		c: cl,
		s: "%s",
	}
}

`, svc.Name, svc.Name, svc.Name, svc.UglyName)

	fmt.Fprintf(g.w, `func New%sClientWithHost(host string, cl client.Client) %s {
	return &impl%s{
		c: cl,
		s: host,
	}
}

`, svc.Name, svc.Name, svc.Name)
}

func (g *generator) writeRegisterFunction(svc *models.Service) {
	fmt.Fprintf(g.w, "func Register%sHandler(srv server.Server, handler %sServer) {\n", svc.Name, svc.Name)

	fmt.Fprint(g.w, "\tsrv.AddHandler(handler, ")

	for i, m := range svc.Methods {
		fmt.Fprintf(g.w, `"%s"`, m.Name)

		if i < len(svc.Methods)-1 {
			fmt.Fprint(g.w, ", ")
		}
	}

	fmt.Fprintln(g.w, ")\n}")
}
