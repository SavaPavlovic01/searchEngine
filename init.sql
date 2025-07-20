CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    title TEXT,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS terms (
    id SERIAL PRIMARY KEY,
    term TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS inverted_index (
    term_id INTEGER NOT NULL REFERENCES terms(id),
    document_id INTEGER NOT NULL REFERENCES documents(id),
    tf INTEGER NOT NULL,
    positions INTEGER[] DEFAULT '{}',
    PRIMARY KEY (term_id, document_id)
);

CREATE TABLE IF NOT EXISTS links (
    from_doc INTEGER NOT NULL REFERENCES documents(id),
    to_doc INTEGER NOT NULL REFERENCES documents(id),
    PRIMARY KEY (from_doc, to_doc)
);

CREATE INDEX IF NOT EXISTS idx_terms_term ON terms(term);
CREATE INDEX IF NOT EXISTS idx_inverted_index_term_id ON inverted_index(term_id);
CREATE INDEX IF NOT EXISTS idx_inverted_index_document_id ON inverted_index(document_id);
CREATE INDEX IF NOT EXISTS idx_links_from_doc ON links(from_doc);
CREATE INDEX IF NOT EXISTS idx_links_to_doc ON links(to_doc);
