package gameconsole

import (
	"fmt"
	"strings"
)

func (p *Profile) Command(id string) (Command, bool) {
	for _, c := range p.Commands {
		if c.ID == id {
			return c, true
		}
	}
	return Command{}, false
}

func (c Command) Render(args map[string]string) (string, error) {
	out := c.Template
	for _, a := range c.Args {
		value := strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(args[a.Name]))
		if value == "" && !a.Optional {
			return "", fmt.Errorf("не указан аргумент «%s»", a.Label)
		}
		out = strings.ReplaceAll(out, "{"+a.Name+"}", value)
	}
	return strings.TrimSpace(out), nil
}
