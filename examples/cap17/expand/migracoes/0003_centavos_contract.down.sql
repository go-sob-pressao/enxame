-- A volta reconstrói valor a partir de valor_centavos: nenhum dado se
-- perdeu no contract, porque a coluna nova já tinha tudo.
ALTER TABLE pedido ADD COLUMN valor NUMERIC(10,2);
UPDATE pedido SET valor = valor_centavos / 100.0;
ALTER TABLE pedido ALTER COLUMN valor_centavos DROP NOT NULL;

CREATE FUNCTION pedido_centavos() RETURNS trigger AS $$
BEGIN
    IF NEW.valor_centavos IS NULL AND NEW.valor IS NOT NULL THEN
        NEW.valor_centavos := round(NEW.valor * 100);
    END IF;
    RETURN NEW;
END $$ LANGUAGE plpgsql;

CREATE TRIGGER pedido_centavos BEFORE INSERT OR UPDATE ON pedido
    FOR EACH ROW EXECUTE FUNCTION pedido_centavos();
