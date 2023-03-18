package templates

import (
	"embed"
	"html/template"
	"io/fs"
	"math"
	"path/filepath"
	"regexp"

	"code.cloudfoundry.org/bytefmt"
	"github.com/xescugc/rebost/dashboard"
	"github.com/xescugc/rebost/state"
)

const (
	viewsDir  = "views"
	extension = "/*.tmpl"
)

var (
	layoutsDir = filepath.Join(viewsDir, "layouts")

	//go:embed views/layouts/* views/dashboard/*
	files embed.FS

	// Templates is the cache of all the templates we have
	Templates map[string]*template.Template

	idR = regexp.MustCompile(`^[^a-z]+|[^\w]+`)
)

func init() {
	if Templates == nil {
		Templates = make(map[string]*template.Template)
	}

	loadTemplates(viewsDir)

	return
}

func loadTemplates(path string) error {
	tmplFiles, err := fs.ReadDir(files, path)
	if err != nil {
		panic(err)
	}

	for _, tmpl := range tmplFiles {
		if tmpl.IsDir() {
			loadTemplates(filepath.Join(path, tmpl.Name()))
			continue
		}

		newpath := filepath.Join(path, tmpl.Name())

		if _, ok := Templates[newpath]; ok {
			continue
		}

		pt := template.New(tmpl.Name()).Funcs(template.FuncMap{
			"percentageNodesUsedSize": func(ns []*dashboard.Node) float64 {
				var (
					us int
					ts int
				)
				for _, n := range ns {
					for _, s := range n.State.Volumes {
						us += s.UsedSize()
						ts += s.TotalSize()
					}
				}
				return math.Round((float64(us) / float64(ts)) * 100)
			},
			"humanizeNodesUsedSize": func(ns []*dashboard.Node) string {
				var (
					us int
				)
				for _, n := range ns {
					for _, s := range n.State.Volumes {
						us += s.UsedSize()
					}
				}
				return bytefmt.ByteSize(uint64(us))
			},
			"humanizeNodesTotalSize": func(ns []*dashboard.Node) string {
				var (
					ts int
				)
				for _, n := range ns {
					for _, s := range n.State.Volumes {
						ts += s.TotalSize()
					}
				}
				return bytefmt.ByteSize(uint64(ts))
			},
			"percentageStateUsedSize": func(s state.State) float64 {
				return math.Round((float64(s.UsedSize()) / float64(s.TotalSize())) * 100)
			},
			"percentageUsedColor": func(p float64) string {
				color := "danger"
				if p < 50 {
					color = "success"
				} else if p < 80 {
					color = "warning"
				}
				return color
			},
			"humanizeUsedSize": func(s state.State) string {
				return bytefmt.ByteSize(uint64(s.UsedSize()))
			},
			"humanizeTotalSize": func(s state.State) string {
				return bytefmt.ByteSize(uint64(s.TotalSize()))
			},
		})

		pt, err := pt.ParseFS(files, newpath, filepath.Join(layoutsDir, extension))
		if err != nil {
			panic(err)
		}

		Templates[newpath] = pt
	}

	return nil
}
