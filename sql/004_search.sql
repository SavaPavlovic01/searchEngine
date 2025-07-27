
CREATE OR REPLACE FUNCTION search_with_levenshtein(
    query_terms TEXT[],          
    max_distance INT DEFAULT 2,
    penalty_factor FLOAT DEFAULT 0.1,
    limit_results INT DEFAULT 10
)
RETURNS TABLE(doc_url TEXT, total_score FLOAT)
LANGUAGE plpgsql
AS $$
DECLARE
    query_term TEXT;
    rec RECORD;
BEGIN
    
    CREATE TEMP TABLE temp_scores (
        document_url TEXT PRIMARY KEY,
        score_sum FLOAT DEFAULT 0
    ) ON COMMIT DROP;
    
    FOREACH query_term IN ARRAY query_terms LOOP
        FOR rec IN
            SELECT tf_idf.document_url, tf_idf.tf_idf, levenshtein(LEFT(query_term, 255), LEFT(tf_idf.term, 255)) AS distance
            FROM tf_idf
            WHERE LENGTH(query_term) <= 255 
                AND LENGTH(tf_idf.term) <= 255
                AND levenshtein(LEFT(query_term, 255), LEFT(tf_idf.term, 255)) <= search_with_levenshtein.max_distance
        LOOP
            IF (rec.tf_idf - rec.distance * search_with_levenshtein.penalty_factor) > 0 THEN
                INSERT INTO temp_scores (document_url, score_sum)
                VALUES (rec.document_url, rec.tf_idf - rec.distance * search_with_levenshtein.penalty_factor)
                ON CONFLICT (document_url) DO UPDATE
                SET score_sum = temp_scores.score_sum + EXCLUDED.score_sum;
            END IF;
        END LOOP;
    END LOOP;
    
    RETURN QUERY
    SELECT temp_scores.document_url, temp_scores.score_sum
    FROM temp_scores
    ORDER BY temp_scores.score_sum DESC
    LIMIT search_with_levenshtein.limit_results;
END;
$$;