SELECT `id`, `created_by_user_id`, `status`, `schedule_run_id`, `created_at_raw` FROM (SELECT * FROM  (
SELECT conversation.created_at_raw,
       conversation.activity_raw,
       conversation.last_turn_id,
       conversation.stage,
       conversation.list_mode,
       conversation.id,
       conversation.summary,
       conversation.last_activity,
       conversation.usage_input_tokens,
       conversation.usage_output_tokens,
       conversation.usage_embedding_tokens,
       conversation.created_at,
       conversation.updated_at,
       conversation.created_by_user_id,
       conversation.agent_id,
       conversation.default_model_provider,
       conversation.default_model,
       conversation.default_model_params,
       conversation.title,
       conversation.conversation_parent_id,
       conversation.conversation_parent_turn_id,
       conversation.metadata,
       conversation.visibility,
       conversation.shareable,
       conversation.scheduled,
       conversation.schedule_id,
       conversation.schedule_run_id,
       conversation.schedule_kind,
       conversation.schedule_timezone,
       conversation.schedule_cron_expr,
       conversation.external_task_ref,
       CASE WHEN ? AND conversation.latest_turn_status IN ('completed','success','done') THEN 'succeeded'
            WHEN ? AND conversation.latest_turn_status <> '' THEN conversation.latest_turn_status
            ELSE conversation.status END AS status
FROM (
SELECT t.*,CAST(t.created_at AS CHAR) AS created_at_raw,CAST(COALESCE(t.last_activity,t.updated_at,t.created_at) AS CHAR) AS activity_raw,
          (SELECT LOWER(COALESCE(z.status,'')) FROM turn z WHERE z.conversation_id=t.id ORDER BY z.created_at DESC,z.id DESC LIMIT 1) AS latest_turn_status,
          (SELECT id
           FROM turn
           WHERE conversation_id = t.id
           ORDER BY created_at DESC, CASE WHEN ? THEN id END DESC
           LIMIT 1) AS last_turn_id,
    CASE WHEN ? THEN CASE
      WHEN COALESCE((SELECT LOWER(COALESCE(z.status,'')) FROM turn z WHERE z.conversation_id=t.id ORDER BY z.created_at DESC,z.id DESC LIMIT 1),LOWER(COALESCE(t.status,''))) IN ('failed','error','terminated') THEN 'error'
      WHEN COALESCE((SELECT LOWER(COALESCE(z.status,'')) FROM turn z WHERE z.conversation_id=t.id ORDER BY z.created_at DESC,z.id DESC LIMIT 1),LOWER(COALESCE(t.status,''))) IN ('canceled','cancelled') THEN 'canceled'
      WHEN COALESCE((SELECT LOWER(COALESCE(z.status,'')) FROM turn z WHERE z.conversation_id=t.id ORDER BY z.created_at DESC,z.id DESC LIMIT 1),LOWER(COALESCE(t.status,''))) IN ('completed','succeeded','success','done','compacted','pruned') THEN 'done'
      WHEN COALESCE((SELECT LOWER(COALESCE(z.status,'')) FROM turn z WHERE z.conversation_id=t.id ORDER BY z.created_at DESC,z.id DESC LIMIT 1),LOWER(COALESCE(t.status,''))) IN ('waiting_for_user','blocked') THEN 'elicitation'
      WHEN COALESCE((SELECT LOWER(COALESCE(z.status,'')) FROM turn z WHERE z.conversation_id=t.id ORDER BY z.created_at DESC,z.id DESC LIMIT 1),LOWER(COALESCE(t.status,''))) IN ('running','thinking','processing','in_progress','queued','pending','open') THEN 'executing'
      ELSE '' END ELSE '' END AS stage, ? AS list_mode
    FROM conversation t
    WHERE 1=1
     AND ( ( ( t.id IN (?) ) ) )
    AND (NOT ? OR (t.conversation_parent_id IS NULL OR (t.conversation_parent_turn_id IS NOT NULL AND EXISTS (SELECT 1 FROM conversation p WHERE p.id=t.conversation_parent_id) AND EXISTS (SELECT 1 FROM turn pt WHERE pt.id=t.conversation_parent_turn_id AND pt.conversation_id=t.conversation_parent_id))))
    AND (NOT ? OR NOT ? OR (COALESCE(t.visibility, '') <> 'private' OR t.created_by_user_id = NULLIF(?, '')))
    AND (NOT ? OR (TRIM(COALESCE(t.conversation_parent_id,''))='' AND TRIM(COALESCE(t.conversation_parent_turn_id,''))='' AND COALESCE(t.last_activity,t.updated_at,t.created_at) IS NOT NULL AND NOT EXISTS(SELECT 1 FROM message maintenance_link WHERE maintenance_link.linked_conversation_id=t.id)
      AND ((?='interactive' AND NOT (COALESCE(t.scheduled,0)<>0 OR TRIM(COALESCE(t.schedule_id,''))<>'' OR TRIM(COALESCE(t.schedule_run_id,''))<>'' OR TRIM(COALESCE(t.schedule_kind,''))<>'' OR EXISTS(SELECT 1 FROM run maintenance_run WHERE maintenance_run.conversation_id=t.id AND COALESCE(maintenance_run.run_kind,'execution')='execution' AND LOWER(TRIM(COALESCE(maintenance_run.conversation_kind,'')))='scheduled'))) OR (? IN ('scheduled','scheduled_fallback') AND (COALESCE(t.scheduled,0)<>0 OR TRIM(COALESCE(t.schedule_id,''))<>'' OR TRIM(COALESCE(t.schedule_run_id,''))<>'' OR TRIM(COALESCE(t.schedule_kind,''))<>'' OR EXISTS(SELECT 1 FROM run maintenance_run WHERE maintenance_run.conversation_id=t.id AND COALESCE(maintenance_run.run_kind,'execution')='execution' AND LOWER(TRIM(COALESCE(maintenance_run.conversation_kind,'')))='scheduled'))))
      AND (?<>'scheduled_fallback' OR NOT EXISTS(SELECT 1 FROM run maintenance_any_run WHERE maintenance_any_run.conversation_id=t.id AND COALESCE(maintenance_any_run.run_kind,'execution')='execution'))))
    
    
    ORDER BY 
      
    
       
      CASE WHEN ? THEN t.id END ASC, CASE WHEN ? AND ? THEN COALESCE(t.last_activity,t.updated_at,t.created_at) END ASC,
             CASE WHEN ? AND NOT ? THEN COALESCE(t.last_activity,t.updated_at,t.created_at) END DESC,
             CASE WHEN ? AND ? THEN t.id END ASC,
             t.id DESC
    
) conversation
)  conversation) AS datly_view
