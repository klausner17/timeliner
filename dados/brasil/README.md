# Timelines do Brasil em CSV

Timelines prontas para carregar no banco com `backend/cmd/importar-csv`. Cada evento cita como
fonte o artigo correspondente da Wikipédia em português.

| Slug | Timeline | Categoria | Eventos | Períodos |
| --- | --- | --- | --- | --- |
| `marcos-da-historia-do-brasil` | Marcos da história do Brasil | História | 54 | Colônia, Período joanino, Primeiro Reinado, Regências, Segundo Reinado, República Velha, Era Vargas, Estado Novo, República de 1946, Ditadura militar, Nova República |
| `biografia-de-lula` | Biografia de Lula | Política | 45 | Infância, sindicalismo, PT e oposição, três mandatos, depois da Presidência |
| `biografia-de-jair-bolsonaro` | Biografia de Jair Bolsonaro | Política | 37 | Infância, Exército, vereador, deputado, Presidência, depois da Presidência |
| `revoltas-armadas-no-brasil` | Revoltas armadas no Brasil | História | 44 | Os mesmos regimes da timeline de marcos, de 1500 a 1985 |
| `musica-brasileira` | Música brasileira | Cultura | 42 | Choro, Era do Rádio, bossa nova, festivais, Jovem Guarda, Tropicália, rock dos anos 80 |
| `classicos-da-literatura-brasileira` | Clássicos da literatura brasileira | Cultura | 46 | Escolas literárias, do Quinhentismo à literatura contemporânea |
| `selecao-brasileira-nas-copas-do-mundo` | Seleção brasileira nas Copas do Mundo | Esporte | 21 | Era Pelé, jejum de 24 anos, era do penta |

## Carregando

Com o Postgres do `docker compose` rodando e o usuário `klausner` criado (o serviço `seed` cria):

```bash
cd backend
go run ./cmd/importar-csv -testar   # confere os arquivos sem gravar nada
go run ./cmd/importar-csv           # cria as 7 timelines no banco
```

Opções: `-dono <handle>` muda o dono, `-so slug1,slug2` importa só algumas, `-substituir` apaga e
recria as que já existem e `-dir` aponta para outra pasta com o mesmo formato. O teste
`internal/seedcsv` valida estes arquivos no CI.

## Formato

Tudo em UTF-8, com vírgula como separador e aspas em textos que têm vírgula.

**`timelines.csv`**: uma linha por timeline.

| Coluna | O que é |
| --- | --- |
| `slug` | Precisa ser o slug que a API gera a partir do título (o importador avisa se não for). Dá nome aos arquivos de eventos e períodos. |
| `titulo`, `descricao` | Título (até 120 caracteres) e descrição (até 2000). |
| `categoria` | História, Política, Esporte, Ciência e tecnologia, Cultura, Ficção ou Pessoal. |
| `cor` | Cor da trilha: `a`, `b`, `c` ou `d`. |
| `tags` | Até 10, separadas por `\|`. |
| `visibilidade`, `contribuicoes` | `publica`/`link`/`privada` e `sugestoes`/`convidados`/`fechada`. Vazio usa `publica` e `sugestoes`. |
| `escala`, `ciclica` | Opcionais. Vazios numa timeline real. Numa timeline fixa, `escala` é `dias`, `semanas`, `meses` ou `anos` e `ciclica` é `sim` ou `nao`; datas de eventos e períodos viram só a posição a partir do início (ex.: `60`). |

**`periodos/<slug>.csv`** (opcional): `nome,inicio,fim,cor`. Início e fim em `AAAA`, `AAAA-MM` ou
`AAAA-MM-DD`; fim vazio continua até hoje. Cor: azul, laranja, verde, roxo, amarelo, vermelho ou
cinza. Períodos podem se sobrepor (o Estado Novo fica dentro da Era Vargas).

**`eventos/<slug>.csv`**: `data,ano_fim,hora,titulo,descricao,periodo,fonte`. A `data` usa o
mesmo formato dos períodos; `ano_fim` marca eventos que duram anos (guerras, revoltas); `hora`
é `HH:MM` e pede o dia; `periodo` precisa ser o nome de um período do arquivo de períodos; `fonte`
é um link.

## Gerando timelines novas com outra LLM

[prompt-para-llm.md](../prompt-para-llm.md) tem um prompt pronto para outra LLM gerar os três
arquivos de um assunto qualquer a partir da Wikipédia, já neste formato.

## Sobre o conteúdo

Os textos foram escritos a partir dos artigos da Wikipédia em português, mas a Wikipédia não
estava acessível no ambiente em que os arquivos foram gerados, então datas e links não foram
conferidos um a um com as páginas. Vale revisar antes de publicar. Nas biografias políticas, os
processos judiciais trazem a decisão e a posição da defesa, sem juízo de valor. Os eventos vão até
o fim de 2025.
