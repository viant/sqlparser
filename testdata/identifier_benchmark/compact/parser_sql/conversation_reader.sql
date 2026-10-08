SELECT conversation.* FROM  (SELECT c.id,
       c.created_by_user_id,
       c.status,
       c.schedule_run_id,
       CAST(c.created_at AS CHAR) AS created_at_raw
FROM conversation c
WHERE 1 = 1  AND ( ( ( c.id IN (?) ) ) )
ORDER BY c.id DESC
)  conversation
