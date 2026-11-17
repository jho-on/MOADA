package templates

import "time"

// StatusComponent is the presentation of a single health check result.
type StatusComponent struct {
	Name    string
	Healthy bool
}

// StatusView carries health data to the status partial.
type StatusView struct {
	Healthy    bool
	Components []StatusComponent
	CheckedAt  time.Time
}

// Label returns the human-readable name of the component.
func (c StatusComponent) Label() string {
	switch c.Name {
	case "database":
		return "Banco de dados"
	case "storage":
		return "Armazenamento de objetos"
	default:
		return c.Name
	}
}

// StatusLabel describes the overall state.
func (v StatusView) StatusLabel() string {
	if v.Healthy {
		return "Todos os serviços estão operacionais."
	}
	return "Algum serviço está indisponível."
}

// CheckedAtLabel formats the check time in the server's local timezone.
func (v StatusView) CheckedAtLabel() string {
	return v.CheckedAt.Local().Format("02/01/2006 às 15:04:05")
}
