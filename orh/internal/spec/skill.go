package spec

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SkillRef is one entry in an agent's `skills:` list. It accepts two YAML
// shapes:
//
//	skills:
//	  - review              # a plain string: a dependency alias pointing
//	                         # at a `kind: skill` package
//	  - text: |              # a mapping: the skill's instructions written
//	      Always be concise.  # directly inline, no separate package needed
//	    description: brief   # (description is optional, for either form)
//
// Exactly one of Alias or Inline is set after parsing.
type SkillRef struct {
	// Alias is a key in the owning package's orh.yaml dependencies,
	// referencing a `kind: skill` package to pull and append.
	Alias string
	// Inline is instruction text written directly in the .orh file,
	// requiring no separate skill package at all.
	Inline string
	// Description is an optional one-line summary for an Inline skill
	// (a referenced package supplies its own via SKILL.md frontmatter).
	Description string
}

// UnmarshalYAML implements yaml.Unmarshaler, accepting either a bare
// string (Alias) or a mapping with a `text:` key (Inline).
func (s *SkillRef) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		s.Alias = value.Value
		return nil
	}

	var inline struct {
		Text        string `yaml:"text"`
		Description string `yaml:"description"`
	}
	if err := value.Decode(&inline); err != nil {
		return fmt.Errorf("skills entry: %w", err)
	}
	if inline.Text == "" {
		return fmt.Errorf("skills entry: inline skill requires a non-empty \"text\" field")
	}
	s.Inline = inline.Text
	s.Description = inline.Description
	return nil
}

// Skill is the root document parsed from a SKILL.md file: a reusable
// instruction module an agent can attach via its `skills:` list. Unlike a
// toolbox or node, a skill has no runtime of its own — its entire effect is
// text appended to an agent's system prompt.
type Skill struct {
	// Description is a short, one-line summary from the file's optional
	// YAML frontmatter (used by `orh info` and future skill-discovery UX).
	Description string
	// Instructions is the skill's body: everything after the frontmatter
	// block (or the whole file, if it has none). This is what gets
	// appended to an agent's prompt.
	Instructions string
}
