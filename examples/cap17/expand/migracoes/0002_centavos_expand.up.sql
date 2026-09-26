-- Expand: a coluna nova nasce opcional, e um trigger a preenche quando
-- quem escreve ainda é o código antigo, que só conhece valor.
ALTER TABLE pedido ADD COLUMN valor_centavos BIGINT;

CREATE FUNCTION pedido_centavos() RETURNS trigger AS $$
BEGIN
    IF NEW.valor_centavos IS NULL AND NEW.valor IS NOT NULL THEN
        NEW.valor_centavos := round(NEW.valor * 100);
    END IF;
    RETURN NEW;
END $$ LANGUAGE plpgsql;

CREATE TRIGGER pedido_centavos BEFORE INSERT OR UPDATE ON pedido
    FOR EACH ROW EXECUTE FUNCTION pedido_centavos();
