package newsubcmd

import (
	"embed"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"github.com/iancoleman/strcase"
	"github.com/wxio/acli/internal/types"
)

type newsubcmdOpt struct {
	rt    *types.Root
	Debug bool

	Name       []string `opts:"mode=arg"`
	Parent     string   `help:"path of parent commands eg foo/bar"`
	Org        string
	Project    string
	ModulePath string `help:"the parent path of the internal src directory"`
	Overwrite  bool

	err error
}

// New constructor for init
func New(rt *types.Root) interface{} {
	parts := strings.Split(os.Args[0], "/")
	in := newsubcmdOpt{
		rt:      rt,
		Org:     "wxio",
		Project: parts[len(parts)-1],
	}
	absPath, err := os.Executable()
	if err == nil {
		parts := strings.Split(absPath, "/")
		in.ModulePath = strings.Join(parts[0:len(parts)-1], "/")
	} else {
		in.err = err
	}
	return &in
}

//go:embed subcmd.tmpl
var fs embed.FS

func (in *newsubcmdOpt) Run() error {
	in.rt.Config(in)
	if in.err != nil {
		fmt.Printf("could get executable's path %v\n", in.err)
		os.Exit(1)
	}
	if len(in.Name) == 0 {
		return fmt.Errorf("Name(s) required\n")
	}
	funcMap := template.FuncMap{
		"ToUpper":      strings.ToUpper,
		"ToCamel":      strcase.ToCamel,
		"ToLowerCamel": strcase.ToLowerCamel,
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(fs, "*.tmpl")
	if err != nil {
		fmt.Printf("Template error %v\n", err)
		glog.Fatalf("Template error %v", err)
	}
	data := struct {
		Name    string
		Parent  []string
		Org     string
		Project string
	}{
		// Name:    Name,
		Parent:  strings.Split(in.Parent, "/"),
		Org:     in.Org,
		Project: in.Project,
	}
	for _, name := range in.Name {
		data.Name = name
		in.makeStarter(name, tmpl, data)
	}
	fmt.Printf("\nMain needs to be manually modified, sample below\n")
	fmt.Printf("``` golang\n")
	if in.Parent == "" {
		for _, name := range in.Name {
			data.Name = name
			err := tmpl.Lookup("mainreg").Execute(os.Stdout, data)
			if err != nil {
				fmt.Printf("template exec error %v\n", err)
			}
		}
	} else {
		data := struct {
			Names   []string
			Parent  []string
			Org     string
			Project string
		}{
			Names:   in.Name,
			Parent:  strings.Split(in.Parent, "/"),
			Org:     in.Org,
			Project: in.Project,
		}
		tmpl.Lookup("mainregwithparent").Execute(os.Stdout, data)
	}
	fmt.Printf("```\n")
	return nil
}

func (in *newsubcmdOpt) makeStarter(name string, tmpl *template.Template, data any) {
	dirname := in.ModulePath + "/internal/" + name
	if in.Parent != "" {
		dirname = in.ModulePath + "/internal/" + in.Parent + "/" + name
	}
	err := os.MkdirAll(dirname, os.ModePerm)
	if err != nil {
		fmt.Printf("create dir error %v\n", err)
	}
	fname := dirname + "/" + name + ".go"
	if !in.Overwrite {
		if _, err = os.Open(fname); err == nil {
			fmt.Printf("Exiting. File already exists. Use --overwrite to ignore.\n")
			fmt.Printf("  %s\n", fname)
			os.Exit(3)
		}
	}
	fh, err := os.Create(fname)
	if err != nil {
		fmt.Printf("create file error %v\n", err)
		os.Exit(1)
	}
	tmpl.Lookup("newsubcmd").Execute(fh, data)
	fh.Close()
	fmt.Printf("written starter code to '%v'\n", fname)
}