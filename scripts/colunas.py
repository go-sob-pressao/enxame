#!/usr/bin/env python3
"""Rearruma comentários // que passam de 72 colunas (tabulação = 4).

As listagens do livro têm no máximo 72 colunas (Plano Mestre §16.7). O código
é reformatado pelo golines; este script cuida dos comentários, que o golines
não quebra. Um parágrafo é uma sequência de linhas // com a mesma indentação
e texto corrido. Ficam intactos: diretivas (//go:, //nolint), marcas livro:,
linhas de tabela (|), trechos indentados dentro do comentário (//<TAB> ou
//   ), listas (- , * , 1.) só são quebradas a partir do próprio item.

Uso: scripts/colunas.py arquivo.go [...]
"""
import re
import sys
import textwrap

LIMITE = 72
TAB = 4


def largura(s: str) -> int:
    return len(s.expandtabs(TAB))


def especial(texto: str) -> bool:
    t = texto
    return (
        t.startswith("go:") or t.startswith("nolint") or t.startswith("livro:")
        or t.startswith("\t") or t.startswith("   ") or "|" in t
        or t.strip() == "" or t.startswith("line ") or t.startswith("export ")
    )


def item_de_lista(texto: str) -> bool:
    return bool(re.match(r"^\s*([-*]|\d+[.)])\s", texto))


def rearrumar(linhas):
    saida = []
    i = 0
    padrao = re.compile(r"^(\s*)// ?(.*)$")
    while i < len(linhas):
        m = padrao.match(linhas[i])
        if not m or (linhas[i].strip() and not linhas[i].lstrip().startswith("//")):
            saida.append(linhas[i])
            i += 1
            continue
        indent, texto = m.group(1), m.group(2)
        if especial(texto) or linhas[i].lstrip().startswith("//go:"):
            saida.append(linhas[i])
            i += 1
            continue
        # junta o parágrafo
        paragrafo = [texto]
        j = i + 1
        while j < len(linhas):
            m2 = padrao.match(linhas[j])
            if not m2 or m2.group(1) != indent:
                break
            t2 = m2.group(2)
            if especial(t2) or item_de_lista(t2):
                break
            paragrafo.append(t2)
            j += 1
        bloco = linhas[i:j]
        if all(largura(l) <= LIMITE for l in bloco):
            saida.extend(bloco)
            i = j
            continue
        prefixo = indent + "// "
        recuo_item = ""
        mi = re.match(r"^(\s*(?:[-*]|\d+[.)])\s+)", paragrafo[0])
        if mi:
            recuo_item = " " * len(mi.group(1))
        disponivel = LIMITE - largura(prefixo)
        texto_corrido = " ".join(p.strip() for p in paragrafo)
        quebradas = textwrap.wrap(
            texto_corrido, width=disponivel, break_long_words=False,
            break_on_hyphens=False, subsequent_indent=recuo_item,
        )
        saida.extend(prefixo + q for q in quebradas)
        i = j
    return saida


def subir_comentarios_finais(linhas):
    """Código longo com comentário no fim: o comentário sobe para a linha de
    cima, com a mesma indentação. Diretivas (//nolint) ficam onde estão."""
    saida = []
    for l in linhas:
        m = re.match(r"^(\s*)(\S.*?\S)\s+// (.*)$", l)
        if (m and largura(l) > LIMITE and not m.group(3).startswith("nolint")
                and not m.group(2).lstrip().startswith("//") and '"' not in m.group(3)
                and m.group(2).count('"') % 2 == 0 and "`" not in m.group(2)):
            saida.append(m.group(1) + "// " + m.group(3))
            saida.append(m.group(1) + m.group(2))
        else:
            saida.append(l)
    return saida


def juntar_nolint(linhas):
    """O golines quebra `f(x) //nolint:regra // motivo` em três linhas, e o
    nolint deixa de valer para a chamada. Junta de volta: o motivo sobe para
    uma linha de comentário e o nolint fica na linha da chamada."""
    saida = []
    i = 0
    while i < len(linhas):
        if i + 2 < len(linhas) and linhas[i].rstrip().endswith("("):
            m = re.match(r"^\s*\)\s*//(nolint:\S+)(?:\s*//\s*(.*))?$", linhas[i + 2])
            if m:
                indent = re.match(r"^(\s*)", linhas[i]).group(1)
                chamada = linhas[i].strip() + linhas[i + 1].strip().rstrip(",") + ")"
                if m.group(2):
                    saida.append(indent + "// " + m.group(2))
                saida.append(indent + chamada + " //" + m.group(1))
                i += 3
                continue
        saida.append(linhas[i])
        i += 1
    return saida


def main():
    for caminho in sys.argv[1:]:
        with open(caminho, encoding="utf-8") as f:
            original = f.read()
        linhas = juntar_nolint(subir_comentarios_finais(original.split("\n")))
        novas = rearrumar(linhas)
        novo = "\n".join(novas)
        if novo != original:
            with open(caminho, "w", encoding="utf-8") as f:
                f.write(novo)


if __name__ == "__main__":
    main()
