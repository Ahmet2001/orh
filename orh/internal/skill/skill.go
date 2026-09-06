// Package skill parses SKILL.md files: a skill's optional YAML frontmatter
// (delimited by "---" lines) plus its instruction body.
package skill

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/pertevniyalai/orh/internal/spec"
)

const frontmatterDelim = "---"

type frontmatter struct {
	Description string `yaml:"description"`
}

// ParseFile reads and parses a SKILL.md file from disk.
func ParseFile(path string) (*spec.Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return Parse(data)
}

// Parse splits raw SKILL.md bytes into an optional YAML frontmatter block
// and the instruction body that follows it. A file with no frontmatter
// (doesn't start with a "---" line) is treated as pure instructions with no
// description.
func Parse(data []byte) (*spec.Skill, error) {
	content := string(data)

	if !strings.HasPrefix(content, frontmatterDelim) {
		return &spec.Skill{Instructions: strings.TrimSpace(content)}, nil
	}

	rest := content[len(frontmatterDelim):]
	rest = strings.TrimPrefix(rest, "\n")

	end := strings.Index(rest, "\n"+frontmatterDelim)
	if end == -1 {
		return nil, fmt.Errorf("frontmatter block is not closed with a trailing %q line", frontmatterDelim)
	}

	fmBlock := rest[:end]
	body := rest[end+len("\n"+frontmatterDelim):]
	body = strings.TrimPrefix(body, "\n")

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(fmBlock), &fm); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	return &spec.Skill{
		Description:  fm.Description,
		Instructions: strings.TrimSpace(body),
	}, nil
}

// Validate checks a parsed Skill for structural correctness.
func Validate(s *spec.Skill) []string {
	var errs []string
	if s.Instructions == "" {
		errs = append(errs, "skill has no instructions (empty body)")
	}
	return errs
}
