package models

import (
	"strings"
	"time"
)

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
}

func (ms *ModelSuite) Test_Task_Create_AssignsNextPosition() {
	first := &Task{Title: "Primera"}
	verrs, err := ms.DB.ValidateAndCreate(first)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	second := &Task{Title: "Segunda"}
	verrs, err = ms.DB.ValidateAndCreate(second)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.Equal(1, first.Position)
	ms.Equal(2, second.Position)
}

func (ms *ModelSuite) Test_Task_Create_KeepsExplicitPosition() {
	task := &Task{Title: "Con posición", Position: 7}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NoError(ms.DB.Reload(task))
	ms.Equal(7, task.Position)
}

func (ms *ModelSuite) Test_Task_DueDate_OptionalByDefault() {
	task := &Task{Title: "Sin fecha"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NoError(ms.DB.Reload(task))
	ms.False(task.DueDate.Valid)
}

func (ms *ModelSuite) Test_Task_DueDate_PersistsDay() {
	due, err := ParseDueDate("2026-10-30")
	ms.NoError(err)
	task := &Task{Title: "Con fecha", DueDate: due}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NoError(ms.DB.Reload(task))
	ms.Equal("2026-10-30", task.DueDateValue())
}

func (ms *ModelSuite) Test_ParseDueDate() {
	for _, raw := range []string{"", "   "} {
		d, err := ParseDueDate(raw)
		ms.NoError(err, raw)
		ms.False(d.Valid, raw)
	}

	d, err := ParseDueDate("2026-10-30")
	ms.NoError(err)
	ms.True(d.Valid)

	for _, raw := range []string{"30/10/2026", "2026-02-30", "2026-10-30T10:00", "mañana"} {
		_, err := ParseDueDate(raw)
		ms.Error(err, raw)
	}
}

func (ms *ModelSuite) Test_Task_DueDateLabel() {
	due, err := ParseDueDate("2026-10-30")
	ms.NoError(err)
	ms.Equal("30/10/2026", Task{DueDate: due}.DueDateLabel())
	ms.Equal("", Task{}.DueDateLabel())
}

func (ms *ModelSuite) Test_Task_IsOverdueOn() {
	today := time.Date(2026, 10, 22, 23, 59, 0, 0, time.FixedZone("x", -5*3600))
	cases := map[string]bool{"2026-10-21": true, "2026-10-22": false, "2026-10-23": false}
	for raw, want := range cases {
		due, err := ParseDueDate(raw)
		ms.NoError(err)
		ms.Equal(want, Task{DueDate: due}.IsOverdueOn(today), raw)
	}
	ms.False(Task{}.IsOverdueOn(today))
}
