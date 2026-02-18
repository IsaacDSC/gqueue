-- CONSUMERS INSCRITOS NOS TOPICOS
SELECT
  e.name as topic,
  CONCAT(elem->>'host', elem->>'path') AS consumer
FROM events AS e
CROSS JOIN LATERAL jsonb_array_elements(e.consumers) AS elem
WHERE e.deleted_at IS NULL
  AND jsonb_typeof(e.consumers) = 'array'
  AND elem ? 'host'
ORDER BY topic desc;
