-- Contract: só depois que nenhum código lê ou escreve valor.
ALTER TABLE pedido ALTER COLUMN valor_centavos SET NOT NULL;
DROP TRIGGER pedido_centavos ON pedido;
DROP FUNCTION pedido_centavos();
ALTER TABLE pedido DROP COLUMN valor;
