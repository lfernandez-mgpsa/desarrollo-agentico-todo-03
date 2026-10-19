// Pide confirmación antes de enviar formularios marcados con data-confirm.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
    event.preventDefault();
  }

  if (!form.matches("[data-task-toggle]")) {
    return;
  }

  event.preventDefault();

  var task = form.closest(".task");
  var completed = !task.classList.contains("task-completed");
  renderToggle(task, completed);

  // Envía la misma petición que el formulario: POST con _method=PUT y authenticity_token.
  // Si el servidor no confirma el cambio, se devuelve la tarea a su estado anterior.
  fetch(form.action, {
    method: "POST",
    body: new FormData(form),
    credentials: "same-origin",
  })
    .then(function (res) {
      if (!res.ok) {
        throw new Error("HTTP " + res.status);
      }
    })
    .catch(function () {
      renderToggle(task, !completed);
    });
});

// Pinta el estado completado o pendiente de una tarea en el DOM.
function renderToggle(task, completed) {
  task.classList.toggle("task-completed", completed);
  var button = task.querySelector(".task-toggle-button");
  var status = task.querySelector(".task-status");

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title").textContent);
  button.setAttribute("title", label + " tarea");

  if (completed && !status) {
    status = document.createElement("span");
    status.className = "task-status";
    status.textContent = "Completada";
    task.insertBefore(status, task.querySelector(".task-actions"));
  } else if (!completed && status) {
    status.remove();
  }
}
