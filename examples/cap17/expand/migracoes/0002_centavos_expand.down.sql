DROP TRIGGER pedido_centavos ON pedido;
DROP FUNCTION pedido_centavos();
ALTER TABLE pedido DROP COLUMN valor_centavos;
