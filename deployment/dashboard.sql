-- CONSUMERS INSCRITOS NOS TOPICOS
SELECT
  e.unique_key as topic,
  CONCAT(elem->>'host', elem->>'path') AS consumer
FROM events AS e
CROSS JOIN LATERAL jsonb_array_elements(e.triggers) AS elem
WHERE e.deleted_at IS NULL
  AND jsonb_typeof(e.triggers) = 'array'
  AND elem ? 'host'
ORDER BY topic desc;
