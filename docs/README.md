# Coleta no painel do CNJ

O coletor está sendo adaptado do painel antigo para o painel novo do CNJ:

- Painel antigo: <https://paineis.cnj.jus.br/QvAJAXZfc/opendoc.htm?document=qvw_l%2FPainelCNJ.qvw&host=QVS%40neodimio03&anonymous=true&sheet=shPORT63Relatorios>
- Painel novo: <https://paineisanalytics.cnj.jus.br/single/?appid=8ccb93ca-848f-4ffe-bf9b-96d57878c4d7&sheet=a710d8e5-fddb-4fa7-9098-4dbca30a391a&lang=pt-BR&theme=Mix_Theme&opt=ctxmenu,currsel>

## Execução local

Exemplo para coletar o TJAP em maio de 2025:

```powershell
$env:COURT = "TJAP"
$env:YEAR = "2025"
$env:MONTH = "05"
$env:TIME_BETWEEN_STEPS = "10s"
$env:OUTPUT_FOLDER = Join-Path $PWD "output"

go run .
```

ou

```powershell
$env:COURT = "TJAP"
$env:YEAR = "2025"
$env:MONTH = "05"
$env:TIME_BETWEEN_STEPS = "7s"
$env:OUTPUT_FOLDER = Join-Path $PWD ("output-amapa-NOVO" + (Get-Date -Format "yyyyMMdd-HHmmss"))
$env:GENERAL_TIMEOUT = "15m"

go run .
```

## Seletores do painel novo

Os Full XPaths e seletores do painel antigo foram substituídos por atributos do HTML do painel novo:

| Elemento | Antes | Depois |
| --- | --- | --- |
| Tribunal | `//*[@title='Tribunal']` | `h6[aria-label="Tribunal"]` |
| Ano | `//*[@title='Ano']` | `h6[aria-label="Ano"]` |
| Mês | `//*[@title='Mês Referencia']` | `h6[aria-label="Mês Referência"]` |
| Campo de pesquisa | `/html/body/div[6]/div[3]/div/div/div[2]/div[1]/div/input` | `input[data-testid="search-input-field"][placeholder="Pesquisar na caixa de listagem"]` |
| Confirmar filtro | Não existia no fluxo antigo | `button[data-testid="actions-toolbar-confirm"][title="Confirmar seleção"]` |
| Controle de envio | `/html/body/div[4]/div/div[2]/div/article/div/div[14]/div/article/div[1]/div/div/div/button` | `//button[normalize-space(.)='Controle de envio']` |
| Relatórios | `/html/body/div[4]/div/div[2]/div/article/div/div[15]/div/article/div[1]/div/div/div/button` | `//button[normalize-space(.)='Relatórios']` |
| Controle de Arquivos | `/html/body/div[5]/div/div[55]/div[2]/table/tbody/tr/td` | `h2[title="Controle de Arquivos"]` |
| Aba Contracheque | `/html/body/div[5]/div/div[49]/div[2]/table/tbody/tr/td` | `span.lui-tab__text[title="Contracheque"]` |

Foram priorizados atributos como `aria-label`, `data-testid`, `title`, `tid` e `q-translation`. Full XPaths dependem da posição fixa dos elementos (`div[6]`, `div[14]` etc.), que no painel mudam quando o Qlik cria menus, modais ou reorganiza o painel.

## Alteração de `exportaExcel`

No painel antigo, a função clicava diretamente em `//*[@title='Enviar para Excel']`. No painel novo, a exportação é um fluxo assíncrono aberto pelo menu de contexto, por isso a função passou a receber também o seletor do relatório:

```go
func (c crawler) exportaExcel(ctx context.Context, reportSelector, fName string) error
```

| Etapa | Seletor atual |
| --- | --- |
| Opção Exportar dados | `//*[normalize-space(text())='Exportar dados']` |
| Modal | `[role="dialog"]` |
| Botão Exportar | `button[tid="table-export"][name="confirm"]` |
| Link do arquivo | `a.export-url[q-translation="Export.CompletedLink"][href*="/tempcontent/"]` |
| Botão Fechar | `button[q-translation="Common.Close"][ng-click="close()"]` |

O fluxo faz o clique direito na visualização, abre **Exportar dados**, confirma a exportação, clica no link gerado, fecha o modal e chama `nomeiaDownload` com o mesmo nome de destino usado anteriormente.

Exemplo para o controle de arquivos:

```go
c.clicaAba(ctx, controleXPATH)
c.exportaExcel(ctx, controleXPATH, ceFname)
```

O seletor passado a `exportaExcel` deve representar a visualização que recebe o clique direito, não apenas sua aba.

 `contrachequeTabSelector` já identifica a aba **Contracheque**

## Comparação das planilhas dos painéis antigo e novo

A análise reproduzível está no notebook [`analisa-planilhas-novo-e-antigo.ipynb`](analisa-planilhas-novo-e-antigo.ipynb). Foram comparados todos os dez arquivos disponíveis: cinco em `output-antigo/` e os cinco correspondentes em `output-novo/`, referentes ao TJAP em maio de 2025.

### Resposta objetiva

As planilhas **não são todas iguais como arquivos**, mas os quatro relatórios com dados de remuneração preservam integralmente o conteúdo. Em `contracheque`, `direitos-eventuais`, `direitos-pessoais` e `indenizacoes`, os 82 registros aparecem na mesma ordem, com as mesmas colunas e os mesmos valores em todas as células.

A única diferença de conteúdo tabular está em `controle-de-arquivos`: o arquivo antigo traz duas linhas de totalização que não existem no novo. A linha detalhada do TJAP foi preservada sem mudança de valor. Portanto, a migração foi predominantemente estrutural, mas não houve apenas mudança no HTML/site: o formato da exportação XLSX, a apresentação, alguns tipos de célula e a tabela de controle também mudaram.

### Dimensões, registros e tamanho

| Arquivo | Intervalo antigo | Intervalo novo | Registros antigo/novo | Colunas antigo/novo | Conteúdo normalizado | Tamanho antigo | Tamanho novo | Variação |
| --- | --- | --- | ---: | ---: | --- | ---: | ---: | ---: |
| `contracheque` | `A1:R83` | `A1:R83` | 82 / 82 | 18 / 18 | Idêntico | 16.261 B | 15.562 B | -699 B (-4,30%) |
| `controle-de-arquivos` | `A1:G4` | `A1:G2` | 3 / 1 | 7 / 7 | Diferente | 5.659 B | 5.578 B | -81 B (-1,43%) |
| `direitos-eventuais` | `A1:R83` | `A1:R83` | 82 / 82 | 18 / 18 | Idêntico | 13.191 B | 12.333 B | -858 B (-6,50%) |
| `direitos-pessoais` | `A1:I83` | `A1:I83` | 82 / 82 | 9 / 9 | Idêntico | 10.053 B | 9.637 B | -416 B (-4,14%) |
| `indenizacoes` | `A1:P83` | `A1:P83` | 82 / 82 | 16 / 16 | Idêntico | 11.780 B | 11.307 B | -473 B (-4,02%) |

O conjunto antigo ocupa 56.944 bytes e o novo, 54.417 bytes: redução de 2.527 bytes, ou 4,44%. Todos os hashes físicos SHA-256 são diferentes, consequência das mudanças de serialização e formatação; isso não representa, por si só, diferença nos dados.

### Diferenças em `controle-de-arquivos`

O arquivo antigo possui estas duas linhas que foram removidas no novo:

| Tribunal | Mês/Ano | Arquivo | Registros | Arquivos | Tribunais enviados | Tribunais |
| --- | --- | --- | ---: | ---: | ---: | ---: |
| Total | vazio | vazio | 82 | 1 | 1 | 1 |
| TJAP | Total | vazio | 82 | 1 | 1 | 1 |

A linha detalhada presente nos dois arquivos é exatamente `TJAP | 05/2025 | TJAP_05_25.xls 02/06/2025 09:14:30 | 82 | 1 | 1 | 1`.

Quatro cabeçalhos mudaram somente a capitalização:

| Coluna | Antigo | Novo |
| --- | --- | --- |
| D | `Qtd. de registros` | `Qtd. de Registros` |
| E | `Qtd. de arquivos` | `Qtd. de Arquivos` |
| F | `Qtd. de tribunais enviados` | `Qtd. de Tribunais Enviados` |
| G | `Qtd. de tribunais` | `Qtd. de Tribunais` |

Embora a semântica seja a mesma, essa alteração pode afetar código que procura colunas com comparação sensível a maiúsculas e minúsculas.

### Estrutura das colunas

Nos quatro relatórios de remuneração, os nomes, a quantidade e a ordem das colunas são exatamente iguais entre os layouts. Nenhuma coluna foi criada, removida ou deslocada.

Há cabeçalhos repetidos nos dois layouts, uma característica anterior que continua no painel novo:

- `direitos-eventuais`: `Outra (R$)` em N/P e `Detalhe` em O/Q;
- `direitos-pessoais`: `Outra (R$)` em E/G e `Detalhe` em F/H;
- `indenizacoes`: `Outra (R$)` em J/L/N e `Detalhe` em K/M/O.

Essas repetições são ambíguas para processamento automático. Por exemplo, `pandas.read_excel` pode acrescentar sufixos `.1` e `.2` aos nomes repetidos. `contracheque` e `controle-de-arquivos` não têm cabeçalhos duplicados.

### Valores e tipos das células

- `contracheque`, `direitos-pessoais` e `indenizacoes`: valores e classes lógicas texto/número iguais célula a célula;
- `direitos-eventuais`: valores visíveis iguais, mas 161 zeros que eram texto no arquivo antigo passaram a ser números no novo — 79 células na coluna F, `Antecipação de férias (R$)`, e 82 na coluna H, `Antecipação de gratificação natalina (R$)`;
- `controle-de-arquivos`: a linha detalhada mantida conserva valores e tipos; a diferença decorre das duas linhas agregadas removidas.

A transformação de zero textual em zero numérico não altera o valor, e tende a facilitar cálculos, mas é observável por validadores ou pipelines que exijam tipo idêntico.

### Formatação e estrutura interna do XLSX

Todos os arquivos têm uma única aba chamada `Sheet1`. Nenhum contém fórmulas, células mescladas, autofiltro ou tabela estruturada do Excel.

As principais diferenças físicas são:

- o exportador antigo grava textos diretamente nas células (`inlineStr`); o novo usa `xl/sharedStrings.xml`;
- o antigo aplica Arial 8, cabeçalho em negrito, preenchimentos, bordas, quebra de linha, larguras específicas por coluna e linhas mais altas;
- o novo usa apresentação próxima do padrão Calibri 11, sem as bordas e preenchimentos antigos, largura uniforme aproximada de 15,71 e altura padrão 15;
- nos quatro relatórios antigos, o cabeçalho tem altura aproximada de 39,22 e as 82 linhas de dados, 26,15; no controle antigo, as quatro linhas têm aproximadamente 26,15;
- os formatos numéricos e IDs de estilo também foram reserializados.

### Conclusão

Não foi encontrada alteração em nomes, ordem, dados cadastrais ou valores monetários dos 82 magistrados nos quatro relatórios principais. A coleta no painel novo é equivalente à antiga para esses dados.

As ressalvas são a remoção das duas linhas agregadas de `controle-de-arquivos`, a capitalização de quatro cabeçalhos, a mudança de tipo de 161 zeros em `direitos-eventuais` e as diferenças de apresentação/codificação do XLSX. Testes automatizados devem comparar valores normalizados, esquema e tipos lógicos, em vez de tamanho do arquivo ou hash binário.
