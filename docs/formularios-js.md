# Formularios y JavaScript

Este documento recoge las reglas que hay que seguir al tocar formularios o al añadir JavaScript que los sustituye. Surgieron del bug en el que completar una tarea no se guardaba (ver el PR de `fix/toggle-persistencia`).

## Envíos PUT y DELETE

Los formularios se envían siempre con `method="POST"`, un campo oculto `_method` (`PUT` o `DELETE`) y otro `authenticity_token` para CSRF:

```html
<form action="/tasks/<%= t.ID %>" method="POST">
  <input type="hidden" name="authenticity_token" value="<%= authenticity_token %>">
  <input type="hidden" name="_method" value="DELETE">
</form>
```

- `csrf.New` (en `actions/app.go`) rechaza las peticiones que no son `GET` sin token válido.
- Buffalo sustituye el método por el valor de `_method` (`MethodOverride`, activo por defecto). Por eso el servidor ve un `PUT` o un `DELETE` aunque el navegador envíe un `POST`.

## Formularios interceptados por JavaScript

`public/assets/app.js` intercepta el envío de `[data-task-toggle]`: cancela el envío nativo, actualiza la UI al instante y hace su propia petición. Esa petición tiene que ser equivalente a la del formulario:

```js
fetch(form.action, {
  method: "POST",
  body: new FormData(form),
  credentials: "same-origin",
})
```

Si el `fetch` se limita a `fetch(form.action)`, es un `GET` sin `_method` ni token. El servidor no tiene ruta para ese `GET`, no ejecuta el cambio, y la tarea parece completada hasta que se refresca la página.

Reglas para cualquier formulario que sustituya el JS:

1. Enviar exactamente lo que enviaría el formulario: `POST`, `FormData(form)` (incluye `_method` y el token) y `credentials: "same-origin"`.
2. Comprobar `res.ok` y revertir la UI optimista si la petición falla. Para ello, pintar el estado con una única función (`renderToggle` en `app.js`) y volver a llamarla con el valor anterior.
3. No asumir que el servidor aceptó el cambio solo porque la UI cambió.

### Reorden de tareas (`[data-task-reorder]`)

`templates/tasks/index.plush.html` incluye un formulario oculto `[data-task-reorder]` con `action="/tasks/reorder"`. No tiene botón ni se envía nunca de forma nativa: solo aporta `_method=PUT` y `authenticity_token`. Al soltar una tarea arrastrada por su asa (o al moverla con `ArrowUp`/`ArrowDown` desde el asa), `app.js` construye `new FormData(form)`, le añade un campo `ids` repetido con todos los IDs en el nuevo orden y hace el mismo `fetch` que en la regla 1. El servidor (`TasksReorder`) responde `422` si `ids` no contiene cada tarea exactamente una vez.

La UI optimista se revierte con `renderOrder(list, previous)`, la única función que pinta el orden, igual que `renderToggle` para el toggle.

## Tests

Los tests de `actions` llaman a las rutas directamente con `as.HTML(...).Put(...)`. Eso no ejecuta el JavaScript, no lee el token de la página renderizada ni comprueba el `fetch`. Un bug como el de la tarea pasa la suite en verde.

El repo no tiene harness de JavaScript. Para cambios en `public/assets/app.js`, la validación es manual en el navegador:

- Completar, refrescar y comprobar el estado.
- Reabrir, refrescar y comprobar el estado.
- Forzar un fallo de la petición (por ejemplo, parar la base de datos o alterar el token en DevTools) y comprobar que la UI vuelve al estado anterior.
- Revisar la pestaña Red: la petición debe ser `POST` con `_method=PUT` en el cuerpo y respuesta `303` o `200`.

Para el reorden:

- Arrastrar una tarea por el asa a otra posición, refrescar y comprobar que el orden se mantiene.
- Repetirlo con la emulación táctil de DevTools: la tarea se mueve y la página no hace scroll.
- Enfocar un asa con Tab y pulsar `ArrowDown`/`ArrowUp`: la tarea se mueve, el foco sigue en el asa y el orden persiste al refrescar.
- Revisar en Red que `POST /tasks/reorder` lleva `_method=PUT`, `authenticity_token` y varios `ids`.
- Alterar el token del formulario oculto en DevTools y reordenar: la lista vuelve al orden anterior.
- Crear una tarea después de reordenar: aparece la última.
