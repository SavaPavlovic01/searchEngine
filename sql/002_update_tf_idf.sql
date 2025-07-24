CREATE OR REPLACE PROCEDURE update_tf_idf()
LANGUAGE SQL
AS $$
INSERT INTO tf_idf (document_url, term, tf, idf, tf_idf)
SELECT
  ii.document_url,
  ii.term,
  ii.tf * 1.0 / document_total.total_terms AS tf,
  LOG(document_counts.total_documents * 1.0 / term_document_counts.df) AS idf,
  (ii.tf * 1.0 / document_total.total_terms) *
    LOG(document_counts.total_documents * 1.0 / term_document_counts.df) AS tf_idf
FROM inverted_index ii
JOIN (
  SELECT document_url, SUM(tf) AS total_terms
  FROM inverted_index
  GROUP BY document_url
) document_total ON ii.document_url = document_total.document_url
JOIN (
  SELECT term, COUNT(DISTINCT document_url) AS df
  FROM inverted_index
  GROUP BY term
) term_document_counts ON ii.term = term_document_counts.term
JOIN (
  SELECT COUNT(DISTINCT document_url) AS total_documents
  FROM inverted_index
) document_counts ON TRUE
ON CONFLICT (document_url, term) DO UPDATE SET
  tf = EXCLUDED.tf,
  idf = EXCLUDED.idf,
  tf_idf = EXCLUDED.tf_idf;
$$;
