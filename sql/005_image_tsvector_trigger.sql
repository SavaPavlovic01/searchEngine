CREATE OR REPLACE FUNCTION image_search_vector_update() RETURNS trigger AS $$
BEGIN
  IF NEW.alt_text = '' THEN
    NEW.image_vector := to_tsvector('english', LEFT(NEW.nearby_text, 1000000));
  ELSE
    NEW.image_vector := to_tsvector('english', LEFT(NEW.alt_text, 1000000));
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS imageVectorTrigger ON images;

CREATE TRIGGER imageVectorTrigger 
BEFORE INSERT OR UPDATE ON images 
FOR EACH ROW
EXECUTE FUNCTION image_search_vector_update();
