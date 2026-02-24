WITH params AS (
    SELECT
        DATE_SUB(CURRENT_DATE(), INTERVAL $EndDayInterval DAY) AS end_date,
        DATE_SUB(CURRENT_DATE(), INTERVAL $StartDayInterval DAY) AS start_date,
        CAST(GREATEST($Page, 1) AS INT64) AS page_number,
        CAST(COALESCE($PageSize, 25) AS INT64) AS page_size
),

perf AS (
    SELECT
        p.agency_id,
        p.advertiser_id,
        p.campaign_id,
        p.order_id AS ad_order_id,
        p.site_id,
        SUM(p.bids) AS bids,
        SUM(p.impressions) AS imps,
        SUM(p.clicks) AS clicks,
        SUM(IFNULL(p.total_spend, 0)) AS spend
    FROM `viant-performance.metrics.fact_performance_hour` p
    JOIN params prm ON TRUE
    WHERE p.event_date BETWEEN prm.start_date AND prm.end_date
    GROUP BY 1,2,3,4,5
),

active_domains AS (
    SELECT DISTINCT
        s.ID AS site_id,
        COALESCE(
            NULLIF(TRIM(s.DISPLAY_NAME), ''),
            NULLIF(TRIM(s.NAME), ''),
            CAST(s.ID AS STRING)
        ) AS site_name,
        (LOWER(REGEXP_REPLACE(COALESCE(NULLIF(TRIM(s.NAME), ''), NULLIF(TRIM(s.MOBILE_URL), ''), ''),r'^(?:https?://)?(?:www\.)?', ''))) AS site_domain
    FROM `viant-adelphic.ci_ads.CI_SITE` s
    JOIN perf p ON p.site_id = s.ID
),

jounce AS (
    SELECT
        j.root_domain,
        ARRAY_AGG(j.jounce_classification ORDER BY j.share_of_demand DESC LIMIT 1)[OFFSET(0)] AS jounce_classification,
        ARRAY_AGG(j.jounce_directness ORDER BY j.share_of_demand DESC LIMIT 1)[OFFSET(0)] AS jounce_directness,
        MAX(j.share_of_supply) AS share_of_supply,
        MAX(j.share_of_demand) AS share_of_demand
    FROM (
        SELECT * FROM `viant-ad-ops.jounce.monetization_v3_*`
        WHERE _TABLE_SUFFIX BETWEEN
            FORMAT_DATE('%Y%m%d', DATE_SUB(CURRENT_DATE(), INTERVAL 3 DAY))
            AND FORMAT_DATE('%Y%m%d', CURRENT_DATE())
        QUALIFY _TABLE_SUFFIX = MAX(_TABLE_SUFFIX) OVER()
    ) j
    JOIN active_domains ad ON j.root_domain = ad.site_domain
    GROUP BY j.root_domain
),

enriched AS (
    SELECT
        p.agency_id,
        p.advertiser_id,
        p.campaign_id,
        p.ad_order_id,
        p.site_id,
        ad.site_name,
        ad.site_domain,
        COALESCE(j.jounce_classification, 'Unknown') AS jounce_classification,
        COALESCE(j.jounce_directness, 'Unknown') AS jounce_directness,
        IFNULL(j.share_of_supply, 0) AS share_of_supply,
        IFNULL(j.share_of_demand, 0) AS share_of_demand,
        p.bids,
        p.imps,
        p.clicks,
        p.spend,
        SAFE_DIVIDE(p.clicks, NULLIF(p.imps, 0)) AS ctr,
        SAFE_DIVIDE(p.spend, NULLIF(p.imps, 0)) * 1000 AS ecpm
    FROM perf p
    JOIN active_domains ad ON p.site_id = ad.site_id
    LEFT JOIN jounce j ON ad.site_domain = j.root_domain
),

ranked AS (
    SELECT
        e.*,
        ROW_NUMBER() OVER (ORDER BY e.spend DESC, e.site_id) AS rn
    FROM enriched e
    WHERE e.spend > 0
),

paged AS (
    SELECT r.*
    FROM ranked r
    JOIN params prm ON TRUE
    WHERE r.rn BETWEEN ((prm.page_number - 1) * prm.page_size + 1)
                   AND (prm.page_number * prm.page_size)
)

SELECT v.*
FROM paged v
ORDER BY v.rn
