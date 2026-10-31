# Prompt para gerar uma timeline em CSV com outra LLM

Use este prompt em outra LLM (de preferência uma com acesso à web) para gerar os CSVs de uma timeline
nova a partir da Wikipédia. Troque `{ASSUNTO}` pelo tema e copie o resultado para a pasta de dados:

```bash
# os três arquivos que a LLM devolver vão para dados/brasil/ (ou outra pasta com -dir)
cd backend
go run ./cmd/importar-csv -testar -so <slug>   # valida sem gravar
go run ./cmd/importar-csv -so <slug>           # grava no banco
```

Se a pasta já tiver um `timelines.csv`, acrescente só a linha nova a ele (sem repetir o cabeçalho).

Timelines ficcionais (com calendário próprio, como a Terra-média) ainda não entram por CSV. Crie essas
pelo site.

---

## O prompt

````text
Você vai montar uma timeline sobre: {ASSUNTO}

Use a Wikipédia em português (pt.wikipedia.org) como fonte principal. Se faltar algo nela, use a
Wikipédia em inglês. Não invente fatos, datas nem links. Se não tiver certeza de um dia ou de um mês,
use só o ano. Cada evento precisa citar a página da Wikipédia de onde saiu.

O resultado vai ser importado direto num banco de dados por um programa que valida tudo. Siga o
formato exatamente.

## Tipo de timeline

Escolha um destes tipos:

- **Real**: acontecimentos com data no calendário (história, biografias, carreiras, obras).
- **Fixa**: etapas sem data no calendário, contadas a partir do início, como o ciclo de vida de um
  animal, as fases de uma gestação ou o processo de um vinho. As posições são números inteiros numa
  unidade (dias, semanas, meses ou anos) a partir de 0. Se o fim volta ao começo (o adulto põe ovos),
  ela é cíclica.

## Saída

Responda só com três blocos de código CSV, cada um precedido pelo nome do arquivo, nesta ordem:

1. `timelines.csv`
2. `periodos/<slug>.csv`
3. `eventos/<slug>.csv`

Regras de CSV: UTF-8, separador vírgula, primeira linha é o cabeçalho exatamente como abaixo. Coloque
entre aspas duplas todo texto que tenha vírgula, aspas ou quebra de linha. Aspas dentro do texto
viram aspas duplas duplicadas (""assim""). Deixe a célula vazia quando não houver valor, sem
escrever "null" nem "-".

### 1. timelines.csv (uma linha)

Cabeçalho:

slug,titulo,descricao,categoria,cor,tags,visibilidade,contribuicoes,escala,ciclica

- slug: gerado a partir do título assim: tudo minúsculo, sem acentos (ç vira c), cada sequência de
  caracteres que não seja letra ou número vira um único hífen, sem hífen no começo ou no fim, no
  máximo 60 caracteres. Exemplo: "Revoltas armadas no Brasil" vira "revoltas-armadas-no-brasil", e
  "Ciclo de vida do sapo" vira "ciclo-de-vida-do-sapo". Se o slug não bater com o título, a
  importação falha.
- titulo: de 1 a 120 caracteres.
- descricao: até 2000 caracteres, uma ou duas frases sobre o que a timeline cobre.
- categoria: exatamente uma destas: História, Política, Esporte, Ciência e tecnologia, Cultura,
  Natureza, Ficção, Pessoal.
- cor: a, b, c ou d.
- tags: até 10, em minúsculas, separadas por | (barra vertical), cada uma com até 30 caracteres.
  Exemplo: brasil|história|música
- visibilidade: publica.
- contribuicoes: sugestoes.
- escala: vazio numa timeline real. Numa fixa: dias, semanas, meses ou anos.
- ciclica: vazio ou nao numa timeline real. Numa fixa: sim ou nao.

### 2. periodos/<slug>.csv (de 0 a 60 linhas)

Períodos são faixas com nome que agrupam os eventos: eras, mandatos, fases, movimentos, escolas.
Use quando ajudarem a ler a timeline. Podem se sobrepor (um período dentro de outro).

Cabeçalho:

nome,inicio,fim,cor

- nome: de 1 a 80 caracteres, sem repetir.
- inicio: na timeline real, AAAA, AAAA-MM ou AAAA-MM-DD. Na fixa, só o número inteiro da posição.
- fim: mesmo formato do início, igual ou depois dele. Vazio quando o período continua até hoje.
  Um fim só com o ano cobre o ano inteiro.
- cor: azul, laranja, verde, roxo, amarelo, vermelho, cinza ou vazio.

### 3. eventos/<slug>.csv (de 1 a 500 linhas, em ordem cronológica)

Cabeçalho:

data,ano_fim,hora,titulo,descricao,periodo,fonte

- data: na timeline real, AAAA, AAAA-MM ou AAAA-MM-DD, com o mês e o dia com dois dígitos. Antes de
  Cristo, use ano negativo (-44 para 44 a.C.). Anos entre -100000 e 3000. O dia precisa existir no
  mês. Na timeline fixa, só o número inteiro da posição, de 0 a 1000000 (ex.: 60 é o dia 60).
- ano_fim: só para eventos que duram mais de um ano (guerras, governos, construções): o ano em que
  terminou, igual ou depois do ano de início. Na fixa, a posição do fim. Vazio nos outros casos.
- hora: HH:MM em 24 horas, só se a hora for conhecida e o dia estiver preenchido. Quase sempre vazio.
  Sempre vazio na fixa.
- titulo: de 1 a 160 caracteres, curto e direto (ex.: "Lei Áurea", "Eclosão dos ovos").
- descricao: até 4000 caracteres. Uma a três frases com o que aconteceu e por que importa.
- periodo: vazio ou o nome exato de um período do arquivo de períodos que contenha a data do
  evento.
- fonte: o link completo da página da Wikipédia usada, começando com https://. Prefira a página
  específica do evento; se ela não existir, use a página do tema principal.

## Conteúdo

- De 20 a 60 eventos costuma ser um bom tamanho. Cubra o assunto inteiro, do começo ao fim, sem
  concentrar tudo num só trecho.
- Escreva em português do Brasil, em frases completas e com tom de enciclopédia.
- Em temas políticos ou polêmicos, seja neutro: descreva o fato e quem decidiu. Quando houver
  acusação ou condenação, diga também a posição da defesa. Não use adjetivos de juízo de valor.
- Não repita o mesmo acontecimento em dois eventos.

## Exemplo de timeline real

timelines.csv
```csv
slug,titulo,descricao,categoria,cor,tags,visibilidade,contribuicoes,escala,ciclica
abolicao-da-escravidao-no-brasil,Abolição da escravidão no Brasil,"As leis e os movimentos que levaram ao fim da escravidão no Brasil, no século XIX.",História,a,brasil|escravidão|império,publica,sugestoes,,
```

periodos/abolicao-da-escravidao-no-brasil.csv
```csv
nome,inicio,fim,cor
Segundo Reinado,1840-07-23,1889-11-15,roxo
```

eventos/abolicao-da-escravidao-no-brasil.csv
```csv
data,ano_fim,hora,titulo,descricao,periodo,fonte
1850-09-04,,,Lei Eusébio de Queirós,"A lei proíbe o tráfico de africanos escravizados para o Brasil.",Segundo Reinado,https://pt.wikipedia.org/wiki/Lei_Eusébio_de_Queirós
1871-09-28,,,Lei do Ventre Livre,"Filhos de mulheres escravizadas nascidos a partir da lei passam a ser considerados livres.",Segundo Reinado,https://pt.wikipedia.org/wiki/Lei_do_Ventre_Livre
1888-05-13,,,Lei Áurea,"A princesa Isabel assina a lei que extingue a escravidão no Brasil.",Segundo Reinado,https://pt.wikipedia.org/wiki/Lei_Áurea
```

## Exemplo de timeline fixa

timelines.csv
```csv
slug,titulo,descricao,categoria,cor,tags,visibilidade,contribuicoes,escala,ciclica
ciclo-de-vida-do-sapo,Ciclo de vida do sapo,"Do ovo ao sapo adulto, em dias aproximados a partir da postura.",Natureza,c,anfíbios|natureza|ciclo de vida,publica,sugestoes,dias,sim
```

periodos/ciclo-de-vida-do-sapo.csv
```csv
nome,inicio,fim,cor
Girino,7,60,verde
```

eventos/ciclo-de-vida-do-sapo.csv
```csv
data,ano_fim,hora,titulo,descricao,periodo,fonte
0,,,Postura dos ovos,"A fêmea deposita os ovos na água, em cordões ou massas gelatinosas.",,https://pt.wikipedia.org/wiki/Sapo
7,,,Eclosão,"Saem dos ovos os girinos, que respiram por brânquias.",Girino,https://pt.wikipedia.org/wiki/Girino
60,,,Metamorfose completa,"O girino perde a cauda, desenvolve pulmões e patas e sai da água.",Girino,https://pt.wikipedia.org/wiki/Metamorfose
```

Antes de responder, confira: o slug bate com o título; todos os eventos têm fonte; todo período
citado num evento existe no arquivo de períodos e contém a data do evento; nenhuma data é inválida;
as linhas estão em ordem cronológica.
````
