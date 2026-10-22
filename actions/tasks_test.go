package actions

import (
	"net/url"
	"strings"
	"time"

	"todo/models"

	"github.com/gofrs/uuid"
)

func (as *ActionSuite) Test_TasksIndex_EmptyState() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "No hay tareas todavía")
}

func (as *ActionSuite) Test_TasksIndex_ListsTasks() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "Comprar material de oficina")
	as.Contains(body, "Llamar al gestor")
	as.Contains(body, "Publicar la oferta de empleo")
	as.Contains(body, "Completada")
}

func (as *ActionSuite) Test_TasksCreate_Valid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "  Nueva tarea de prueba  "})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Nueva tarea de prueba", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksCreate_EmptyTitle_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: ""})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_OnlySpaces_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "     "})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksToggle_CompletesAndReopens() {
	task := &models.Task{Title: "Pendiente de completar"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)

	res = as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksToggle_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s/toggle", id).Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksToggle_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/toggle").Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksDestroy() {
	task := &models.Task{Title: "Tarea que se elimina"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Delete()
	as.Equal(303, res.Code)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksDestroy_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s", id).Delete()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) taskForEditing(completed bool) *models.Task {
	task := &models.Task{Title: "Título original", Completed: completed}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) Test_TasksIndex_EditLink() {
	task := as.taskForEditing(false)
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "/tasks/"+task.ID.String()+"/edit")
	as.Contains(res.Body.String(), "Editar tarea: Título original")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsTitle() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="Título original"`)
	as.Contains(res.Body.String(), `value="PUT"`)
	as.Contains(res.Body.String(), "Guardar cambios")
	as.Contains(res.Body.String(), "Cancelar")
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksEdit_NotFound() {
	res := as.HTML("/tasks/%s/edit", uuid.Must(uuid.NewV4())).Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/edit").Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_EscapesTitle() {
	task := &models.Task{Title: `<img src=x onerror="alert(1)">`}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), `<img src=x`)
	as.Contains(res.Body.String(), "&lt;img")
}

func (as *ActionSuite) Test_TasksUpdate_PersistsTitleOnly() {
	task := as.taskForEditing(true)
	id, createdAt, updatedAt := task.ID, task.CreatedAt, task.UpdatedAt
	res := as.HTML("/tasks/%s", id).Put(url.Values{
		"title": {"  Título actualizado  "}, "completed": {"false"},
		"id": {uuid.Must(uuid.NewV4()).String()}, "created_at": {"2000-01-01T00:00:00Z"},
	})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.Equal("Título actualizado", task.Title)
	as.Equal(id, task.ID)
	as.True(task.Completed)
	as.True(task.CreatedAt.Equal(createdAt))
	as.False(task.UpdatedAt.Before(updatedAt))
	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(1, count)
}

func (as *ActionSuite) Test_TasksUpdate_PendingTaskRemainsPending() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo título"}, "completed": {"true"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo título", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_EmptyTitle() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {""}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.Contains(res.Body.String(), `aria-describedby="title-error"`)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_OnlySpaces() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"   "}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_TitleTooLong() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength+1)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título no puede superar los 200 caracteres.")
	as.Contains(res.Body.String(), title)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_TitleAtLimit() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(title, task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_NotFound() {
	res := as.HTML("/tasks/%s", uuid.Must(uuid.NewV4())).Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid").Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) taskAt(title string, position int) *models.Task {
	task := &models.Task{Title: title, Position: position}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	return task
}

func (as *ActionSuite) assertTitlesInOrder(body string, titles ...string) {
	last := -1
	for _, title := range titles {
		i := strings.Index(body, title)
		as.Greater(i, last, "%q debe aparecer después de la tarea anterior", title)
		last = i
	}
}

func (as *ActionSuite) assertPositions(tasks ...*models.Task) {
	for i, task := range tasks {
		as.NoError(as.DB.Reload(task))
		as.Equal(i+1, task.Position, task.Title)
	}
}

func (as *ActionSuite) Test_TasksIndex_OrdersByPosition() {
	as.taskAt("Tarea A", 2)
	as.taskAt("Tarea B", 1)

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.assertTitlesInOrder(res.Body.String(), "Tarea B", "Tarea A")
}

func (as *ActionSuite) Test_TasksIndex_RendersReorderControls() {
	task := as.taskAt("Tarea movible", 1)

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "data-task-reorder")
	as.Contains(body, `action="/tasks/reorder"`)
	as.Contains(body, `data-task-id="`+task.ID.String()+`"`)
	as.Contains(body, "Mover tarea: Tarea movible")
}

func (as *ActionSuite) Test_TasksCreate_AppendsAtEnd() {
	as.taskAt("Primera", 1)
	as.taskAt("Segunda", 2)

	res := as.HTML("/tasks").Post(&models.Task{Title: "Tercera"})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.Where("title = ?", "Tercera").First(task))
	as.Equal(3, task.Position)
}

func (as *ActionSuite) Test_TasksReorder_Valid() {
	a, b, c := as.taskAt("Tarea A", 1), as.taskAt("Tarea B", 2), as.taskAt("Tarea C", 3)

	res := as.HTML("/tasks/reorder").Put(url.Values{"ids": {c.ID.String(), a.ID.String(), b.ID.String()}})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
	as.assertPositions(c, a, b)

	res = as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.assertTitlesInOrder(res.Body.String(), "Tarea C", "Tarea A", "Tarea B")
}

func (as *ActionSuite) Test_TasksReorder_MissingTask_Invalid() {
	a, b, c := as.taskAt("Tarea A", 1), as.taskAt("Tarea B", 2), as.taskAt("Tarea C", 3)

	res := as.HTML("/tasks/reorder").Put(url.Values{"ids": {c.ID.String(), a.ID.String()}})
	as.Equal(422, res.Code)
	as.assertPositions(a, b, c)
}

func (as *ActionSuite) Test_TasksReorder_UnknownTask_Invalid() {
	a, b, c := as.taskAt("Tarea A", 1), as.taskAt("Tarea B", 2), as.taskAt("Tarea C", 3)
	unknown := uuid.Must(uuid.NewV4())

	res := as.HTML("/tasks/reorder").Put(url.Values{"ids": {c.ID.String(), a.ID.String(), unknown.String()}})
	as.Equal(422, res.Code)
	as.assertPositions(a, b, c)
}

func (as *ActionSuite) Test_TasksReorder_Duplicate_Invalid() {
	a, b, c := as.taskAt("Tarea A", 1), as.taskAt("Tarea B", 2), as.taskAt("Tarea C", 3)

	res := as.HTML("/tasks/reorder").Put(url.Values{"ids": {a.ID.String(), a.ID.String(), b.ID.String()}})
	as.Equal(422, res.Code)
	as.assertPositions(a, b, c)
}

func (as *ActionSuite) Test_TasksReorder_InvalidID_Invalid() {
	a, b, c := as.taskAt("Tarea A", 1), as.taskAt("Tarea B", 2), as.taskAt("Tarea C", 3)

	res := as.HTML("/tasks/reorder").Put(url.Values{"ids": {a.ID.String(), b.ID.String(), "no-es-un-uuid"}})
	as.Equal(422, res.Code)
	as.assertPositions(a, b, c)
}

func (as *ActionSuite) Test_TasksReorder_DoesNotChangeCompletion() {
	done := &models.Task{Title: "Tarea hecha", Completed: true, Position: 1}
	verrs, err := as.DB.ValidateAndCreate(done)
	as.NoError(err)
	as.False(verrs.HasAny())
	other := as.taskAt("Tarea pendiente", 2)

	res := as.HTML("/tasks/reorder").Put(url.Values{"ids": {other.ID.String(), done.ID.String()}})
	as.Equal(303, res.Code)
	as.assertPositions(other, done)
	as.True(done.Completed)
}

func (as *ActionSuite) taskWithDue(title, due string) *models.Task {
	d, err := models.ParseDueDate(due)
	as.NoError(err)
	task := &models.Task{Title: title, DueDate: d}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) Test_TasksCreate_WithDueDate() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Con fecha"}, "due_date": {"2026-10-30"}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("2026-10-30", task.DueDateValue())
}

func (as *ActionSuite) Test_TasksCreate_WithoutDueDate() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Sin fecha"}, "due_date": {""}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.False(task.DueDate.Valid)
}

func (as *ActionSuite) Test_TasksCreate_InvalidDueDate() {
	for _, raw := range []string{"30/10/2026", "2026-02-30", "mañana"} {
		res := as.HTML("/tasks").Post(url.Values{"title": {"Tarea"}, "due_date": {raw}})
		as.Equal(422, res.Code, raw)
		as.Contains(res.Body.String(), "La fecha límite no es válida.", raw)
	}

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_InvalidDueDateAndEmptyTitle() {
	res := as.HTML("/tasks").Post(url.Values{"title": {""}, "due_date": {"mañana"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.Contains(res.Body.String(), "La fecha límite no es válida.")
}

func (as *ActionSuite) Test_TasksIndex_ShowsDueDate() {
	as.taskWithDue("Con fecha", "2026-10-30")
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `datetime="2026-10-30"`)
	as.Contains(res.Body.String(), "30/10/2026")
}

func (as *ActionSuite) Test_TasksIndex_MarksOverdue() {
	as.taskWithDue("Vencida", time.Now().AddDate(0, 0, -1).Format(models.DueDateLayout))
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "task-due-overdue")
	as.Contains(res.Body.String(), "Vencida")
}

func (as *ActionSuite) Test_TasksIndex_FutureDueDateNotOverdue() {
	as.taskWithDue("Futura", time.Now().AddDate(0, 0, 1).Format(models.DueDateLayout))
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), "task-due-overdue")
	as.NotContains(res.Body.String(), "Vencida")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsDueDate() {
	task := as.taskWithDue("Con fecha", "2026-10-30")
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="2026-10-30"`)
}

func (as *ActionSuite) Test_TasksUpdate_SetsDueDate() {
	task := as.taskForEditing(true)
	position := task.Position
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Título nuevo"}, "due_date": {"2026-11-05"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título nuevo", task.Title)
	as.Equal("2026-11-05", task.DueDateValue())
	as.True(task.Completed)
	as.Equal(position, task.Position)
}

func (as *ActionSuite) Test_TasksUpdate_ClearsDueDate() {
	task := as.taskWithDue("Con fecha", "2026-10-30")
	position := task.Position
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Con fecha"}, "due_date": {""}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.False(task.DueDate.Valid)
	as.False(task.Completed)
	as.Equal(position, task.Position)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidDueDate() {
	task := as.taskWithDue("Original", "2026-10-30")
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Cambiado"}, "due_date": {"30/10/2026"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "La fecha límite no es válida.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Original", task.Title)
	as.Equal("2026-10-30", task.DueDateValue())
}

func (as *ActionSuite) Test_TasksToggle_KeepsDueDate() {
	task := as.taskWithDue("Con fecha", "2026-10-30")
	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
	as.Equal("2026-10-30", task.DueDateValue())
}
