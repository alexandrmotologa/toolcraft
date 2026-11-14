package generator

import (
	"fmt"

	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/domain"
	gogen "github.com/alexandrmotologa/toolcraft/internal/generator/golang"
	tsgen "github.com/alexandrmotologa/toolcraft/internal/generator/typescript"
)

// Generator represents a code synthesizer targeting a specific programming language.
type Generator interface {
	Generate(spec *domain.Spec, cfg *config.Config) error
}

// GetGenerator returns the synthesizer corresponding to the configured language.
func GetGenerator(lang string) (Generator, error) {
	switch lang {
	case "go", "golang":
		return gogen.NewGenerator(), nil
	case "ts", "typescript":
		return tsgen.NewGenerator(), nil
	default:
		return nil, fmt.Errorf("unsupported output language '%s'; supported targets: go, typescript", lang)
	}
}
