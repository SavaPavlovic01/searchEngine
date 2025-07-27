CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    title TEXT,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS inverted_index (
    term TEXT UNIQUE NOT NULL,
    document_url TEXT NOT NULL REFERENCES documents(url),
    tf INTEGER NOT NULL,
    positions INTEGER[] DEFAULT '{}',
    PRIMARY KEY (term, document_url)
);

CREATE TABLE IF NOT EXISTS links (
    from_doc TEXT NOT NULL,
    to_doc TEXT NOT NULL, 
    PRIMARY KEY (from_doc, to_doc)
);

CREATE TABLE tf_idf (
  document_url TEXT,
  term TEXT,
  tf REAL,
  idf REAL,
  tf_idf REAL,
  PRIMARY KEY (document_url, term),
  FOREIGN KEY (document_url, term) REFERENCES inverted_index (document_url, term)
);

CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;

CREATE INDEX IF NOT EXISTS idx_terms_term ON terms(term);
CREATE INDEX IF NOT EXISTS idx_inverted_index_term_id ON inverted_index(term_id);
CREATE INDEX IF NOT EXISTS idx_inverted_index_document_id ON inverted_index(document_id);
CREATE INDEX IF NOT EXISTS idx_links_from_doc ON links(from_doc);
CREATE INDEX IF NOT EXISTS idx_links_to_doc ON links(to_doc);
