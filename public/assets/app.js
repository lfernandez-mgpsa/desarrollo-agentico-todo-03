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

// Estado del arrastre en curso: tarea, lista y orden previo para poder revertir.
var drag = null;

// Devuelve los IDs de las tareas en el orden actual del DOM.
function taskOrder(list) {
  return Array.prototype.map.call(list.querySelectorAll(":scope > .task"), function (task) {
    return task.dataset.taskId;
  });
}

// Pinta el orden de la lista según ids; es la única función que reordena el DOM.
function renderOrder(list, ids) {
  ids.forEach(function (id) {
    var task = list.querySelector(':scope > .task[data-task-id="' + id + '"]');
    if (task) {
      list.appendChild(task);
    }
  });
}

// Escribe un mensaje en la región viva para lectores de pantalla.
function announce(text) {
  var status = document.querySelector("[data-task-reorder-status]");
  if (status) {
    status.textContent = text;
  }
}

// Envía el orden actual como el formulario oculto y revierte si el servidor no lo acepta.
function saveOrder(list, previous) {
  if (previous.join() === taskOrder(list).join()) {
    return;
  }

  var form = document.querySelector("[data-task-reorder]");
  var body = new FormData(form);
  taskOrder(list).forEach(function (id) {
    body.append("ids", id);
  });

  fetch(form.action, { method: "POST", body: body, credentials: "same-origin" })
    .then(function (res) {
      if (!res.ok) {
        throw new Error("HTTP " + res.status);
      }
    })
    .catch(function () {
      renderOrder(list, previous);
      announce("No se pudo guardar el orden.");
    });
}

// Termina el arrastre; si se cancela, devuelve la lista al orden previo sin guardar.
function endDrag(cancelled) {
  if (!drag) {
    return;
  }
  var current = drag;
  drag = null;

  current.task.classList.remove("task-dragging");
  current.list.classList.remove("task-list-dragging");

  if (cancelled) {
    renderOrder(current.list, current.previous);
  } else {
    saveOrder(current.list, current.previous);
  }
}

// Empieza el arrastre al pulsar el asa con el botón principal.
document.addEventListener("pointerdown", function (event) {
  var handle = event.target.closest(".task-drag-handle");
  if (!handle || event.button !== 0) {
    return;
  }

  var task = handle.closest(".task");
  var list = task.parentElement;
  drag = { task: task, list: list, previous: taskOrder(list) };

  task.classList.add("task-dragging");
  list.classList.add("task-list-dragging");
  handle.setPointerCapture(event.pointerId);
});

// Coloca la tarea arrastrada antes del primer hermano cuyo centro queda por debajo del puntero.
document.addEventListener("pointermove", function (event) {
  if (!drag) {
    return;
  }

  var siblings = drag.list.querySelectorAll(":scope > .task:not(.task-dragging)");
  var before = null;
  for (var i = 0; i < siblings.length; i++) {
    var rect = siblings[i].getBoundingClientRect();
    if (rect.top + rect.height / 2 > event.clientY) {
      before = siblings[i];
      break;
    }
  }

  if (before) {
    if (drag.task.nextElementSibling !== before) {
      drag.list.insertBefore(drag.task, before);
    }
  } else if (drag.list.lastElementChild !== drag.task) {
    drag.list.appendChild(drag.task);
  }
});

// Guarda el orden al soltar.
document.addEventListener("pointerup", function () {
  endDrag(false);
});

// Revierte el orden si el navegador cancela el arrastre.
document.addEventListener("pointercancel", function () {
  endDrag(true);
});

// Mueve la tarea una posición con las flechas cuando el asa tiene el foco.
document.addEventListener("keydown", function (event) {
  var handle = event.target.closest && event.target.closest(".task-drag-handle");
  if (!handle || (event.key !== "ArrowUp" && event.key !== "ArrowDown")) {
    return;
  }
  event.preventDefault();

  var task = handle.closest(".task");
  var list = task.parentElement;
  var previous = taskOrder(list);

  if (event.key === "ArrowUp" && task.previousElementSibling) {
    list.insertBefore(task, task.previousElementSibling);
  } else if (event.key === "ArrowDown" && task.nextElementSibling) {
    list.insertBefore(task.nextElementSibling, task);
  } else {
    return;
  }

  handle.focus();
  saveOrder(list, previous);
  var order = taskOrder(list);
  announce("Tarea movida a la posición " + (order.indexOf(task.dataset.taskId) + 1) + " de " + order.length + ".");
});
