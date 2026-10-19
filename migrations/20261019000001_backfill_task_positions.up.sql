UPDATE tasks
SET position = ordered.rn
FROM (
  SELECT id, ROW_NUMBER() OVER (ORDER BY created_at ASC, id ASC) AS rn
  FROM tasks
) AS ordered
WHERE tasks.id = ordered.id;
