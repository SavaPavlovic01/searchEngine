CREATE OR REPLACE FUNCTION documents_search_vector_update() RETURNS trigger AS $$
BEGIN
  NEW.content_vector := to_tsvector('english', LEFT(NEW.content, 1000000));
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS tsvectorupdate ON documents;

CREATE TRIGGER tsvectorupdate
BEFORE INSERT OR UPDATE ON documents
FOR EACH ROW
EXECUTE FUNCTION documents_search_vector_update();