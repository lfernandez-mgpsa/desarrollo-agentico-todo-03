package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// TasksListOrder es el orden en que se muestra la lista de tareas.
const TasksListOrder = "position asc, created_at asc, id asc"

// DueDateLayout es el formato de la fecha límite en formularios y en base de datos.
const DueDateLayout = "2006-01-02"

// DueDateInvalidMessage es el error cuando la fecha límite no tiene formato válido.
const DueDateInvalidMessage = "La fecha límite no es válida."

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID  `json:"id" db:"id" form:"-"`
	Title     string     `json:"title" db:"title" form:"title"`
	Completed bool       `json:"completed" db:"completed" form:"-"`
	Position  int        `json:"position" db:"position" form:"-"`
	DueDate   nulls.Time `json:"due_date" db:"due_date" form:"-"`
	CreatedAt time.Time  `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at" form:"-"`
}

// ParseDueDate convierte el valor del formulario en fecha límite. Vacío significa sin fecha.
func ParseDueDate(raw string) (nulls.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nulls.Time{}, nil
	}
	d, err := time.Parse(DueDateLayout, raw)
	if err != nil {
		return nulls.Time{}, err
	}
	return nulls.NewTime(d), nil
}

// DueDateValue devuelve la fecha límite como YYYY-MM-DD, o "" si no tiene.
func (t Task) DueDateValue() string {
	if !t.DueDate.Valid {
		return ""
	}
	return t.DueDate.Time.Format(DueDateLayout)
}

// DueDateLabel devuelve la fecha límite como dd/mm/aaaa, o "" si no tiene.
func (t Task) DueDateLabel() string {
	if !t.DueDate.Valid {
		return ""
	}
	return t.DueDate.Time.Format("02/01/2006")
}

// IsOverdueOn indica si la fecha límite es anterior al día de today. No mira Completed.
func (t Task) IsOverdueOn(today time.Time) bool {
	if !t.DueDate.Valid {
		return false
	}
	y, m, d := today.Date()
	todayDay := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	dy, dm, dd := t.DueDate.Time.Date()
	due := time.Date(dy, dm, dd, 0, 0, 0, 0, time.UTC)
	return due.Before(todayDay)
}

// IsOverdue es IsOverdueOn con la fecha local actual.
func (t Task) IsOverdue() bool { return t.IsOverdueOn(time.Now()) }

// Tasks es una colección de Task.
type Tasks []Task

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	return verrs, nil
}

// BeforeCreate coloca la tarea nueva al final de la lista.
func (t *Task) BeforeCreate(tx *pop.Connection) error {
	if t.Position != 0 {
		return nil
	}
	return tx.Store.Get(&t.Position, "SELECT COALESCE(MAX(position), 0) + 1 FROM tasks")
}
