package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/dadosjusbr/status"
	"github.com/xuri/excelize/v2"
)

const (
	baseURL                      = "https://paineisanalytics.cnj.jus.br/single/?appid=8ccb93ca-848f-4ffe-bf9b-96d57878c4d7&sheet=a710d8e5-fddb-4fa7-9098-4dbca30a391a&lang=pt-BR&theme=Mix_Theme&opt=ctxmenu,currsel"
	identificadorControle        = `h2[title="Controle de Arquivos"]`
	contrachequeTabSelector      = `span.lui-tab__text[title="Contracheque"]`
	direitosPessoaisTabSelector  = `span.lui-tab__text[title="Direitos Pessoais"]`
	indenizacoesTabSelector      = `span.lui-tab__text[title="Indenizações"]`
	direitosEventuaisTabSelector = `span.lui-tab__text[title="Direitos Eventuais"]`
	botaoRelatoriosXPATH         = `//button[normalize-space(.)='Relatórios']`
	botaoControleXPATH           = `//button[normalize-space(.)='Controle de envio']`
)

type crawler struct {
	downloadTimeout   time.Duration
	collectionTimeout time.Duration
	timeBetweenSteps  time.Duration
	court             string
	year              string
	month             string
	output            string
}

func (c crawler) crawl() ([]string, error) {
	// Pegar variáveis de ambiente

	// Chromedp setup.
	log.SetOutput(os.Stderr) // Enviando logs para o stderr para não afetar a execução do coletor.

	alloc, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true), // mude para false para executar com navegador visível.
			chromedp.WindowSize(1920, 1080),
			chromedp.NoSandbox,
			chromedp.DisableGPU,
		)...,
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(
		alloc,
		chromedp.WithLogf(log.Printf), // remover comentário para depurar
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, c.collectionTimeout)
	defer cancel()

	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1920, 1080),
		chromedp.Navigate(baseURL),
		c.configuraDownloads(),
	); err != nil {
		return nil, verificaErro(err)
	}

	// NOTA IMPORTANTE: os prefixos dos nomes dos arquivos tem que ser igual
	// ao esperado no parser CNJ.

	// Planilha de controle
	ceFname := c.downloadFilePath("controle-de-arquivos")
	log.Printf("Fazendo download do controle de arquivos (%s)...", ceFname)
	if err := c.clicaAba(ctx, botaoControleXPATH); err != nil {
		return nil, err
	}
	if err := c.clicaAba(ctx, identificadorControle); err != nil {
		return nil, err
	}
	if err := c.exportaExcel(ctx, identificadorControle, ceFname); err != nil {
		return nil, err
	}
	log.Printf("Download do controle de arquivos realizado com sucesso!\n")

	// Verificando se tem dados
	if err := validate(c); err != nil {
		return nil, err
	}

	// Contracheque
	cqFname := c.downloadFilePath("contracheque")
	log.Printf("Fazendo download do contracheque (%s)...", cqFname)
	if err := c.clicaAba(ctx, botaoRelatoriosXPATH); err != nil {
		return nil, err
	}
	// Selecionando contracheque...
	if err := c.clicaAba(ctx, contrachequeTabSelector); err != nil {
		return nil, err
	}
	if err := c.exportaExcel(ctx, contrachequeTabSelector, cqFname); err != nil {
		return nil, err
	}

	// Verificando se os dados estão agregados
	if err := validateSourceFile(cqFname); err != nil {
		return nil, err
	}

	log.Printf("Download realizado com sucesso!\n")

	// Direitos pessoais
	dpFname := c.downloadFilePath("direitos-pessoais")
	log.Printf("Fazendo download dos direitos pessoais (%s)...", dpFname)
	if err := c.clicaAba(ctx, direitosPessoaisTabSelector); err != nil {
		return nil, err
	}
	if err := c.exportaExcel(ctx, direitosPessoaisTabSelector, dpFname); err != nil {
		return nil, err
	}
	log.Printf("Download realizado com sucesso!\n")

	// Indenizações
	iFname := c.downloadFilePath("indenizacoes")
	log.Printf("Fazendo download das indenizações (%s)...", iFname)
	if err := c.clicaAba(ctx, indenizacoesTabSelector); err != nil {
		return nil, err
	}
	if err := c.exportaExcel(ctx, indenizacoesTabSelector, iFname); err != nil {
		return nil, err
	}
	log.Printf("Download realizado com sucesso!\n")

	// Verbas
	deFname := c.downloadFilePath("direitos-eventuais")
	log.Printf("Fazendo download dos direitos eventuais (%s)...", deFname)
	if err := c.clicaAba(ctx, direitosEventuaisTabSelector); err != nil {
		return nil, err
	}
	if err := c.exportaExcel(ctx, direitosEventuaisTabSelector, deFname); err != nil {
		return nil, err
	}
	log.Printf("Download dos direitos eventuais realizado com sucesso!\n")

	// Retorna caminhos completos dos arquivos baixados.
	return []string{cqFname, dpFname, deFname, iFname, ceFname}, nil
}

func (c crawler) downloadFilePath(prefix string) string {
	return filepath.Join(c.output, fmt.Sprintf("%s-%s-%s-%s.xlsx", prefix, c.court, c.year, c.month))
}

type filtro struct {
	campo, seletor, valor string
}

func (c crawler) filtros(reportSelector string) []filtro {
	// As folhas usam campos distintos para a referência de envio e do contracheque.
	referencia := "P63_CC_REFERENCIA_"
	if reportSelector == identificadorControle {
		referencia = "P63_CE_REFERENCIA_"
	}
	return []filtro{
		{"P63_CE_TRIBUNAL", `h6[aria-label="Tribunal"]`, strings.ToUpper(c.court)},
		{referencia + "ANO", `h6[aria-label="Ano"]`, c.year},
		{referencia + "MES", `h6[aria-label="Mês Referência"]`, c.month},
	}
}

// Confere o valor efetivamente confirmado, não apenas o texto digitado na busca.
const filtroAplicadoJS = `(campo, valor) => {
	const item = document.querySelector('#selection-list [data-csid="' + campo + '"] [tid="current-selection-item-text"]');
	return !!item && item.textContent.trim() === valor;
}`

func filtroAplicado(ctx context.Context, f filtro) (bool, error) {
	var aplicado bool
	err := chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`(%s)(%q, %q)`, filtroAplicadoJS, f.campo, f.valor), &aplicado))
	return aplicado, err
}

// A navegação pode limpar as seleções. Reaplica somente campos ausentes ou divergentes;
// clicar novamente em um valor já selecionado no Qlik o desmarca.
func (c crawler) garanteFiltros(ctx context.Context, reportSelector string) error {
	if err := chromedp.Run(ctx, chromedp.WaitVisible(reportSelector, chromedp.ByQuery)); err != nil {
		return err
	}
	for _, f := range c.filtros(reportSelector) {
		if err := chromedp.Run(ctx, chromedp.WaitVisible(f.seletor, chromedp.ByQuery)); err != nil {
			return err
		}
		aplicado, err := filtroAplicado(ctx, f)
		if err != nil {
			return err
		}
		if aplicado {
			continue
		}
		// Uma seleção anterior diferente deve ser substituída, não somada à nova.
		item := fmt.Sprintf(`#selection-list [data-csid=%q]`, f.campo)
		var existentes []*cdp.Node
		if err := chromedp.Run(ctx, chromedp.Nodes(item+` .remove`, &existentes, chromedp.ByQueryAll, chromedp.AtLeast(0))); err != nil {
			return err
		}
		if len(existentes) > 0 {
			if err := chromedp.Run(ctx, chromedp.Click(item+` .remove`, chromedp.ByQuery), chromedp.WaitNotPresent(item, chromedp.ByQuery)); err != nil {
				return err
			}
		}
		log.Printf("Aplicando filtro %s=%s", f.campo, f.valor)
		if err := c.aplicaFiltro(ctx, f); err != nil {
			return err
		}
		if err := chromedp.Run(ctx, chromedp.PollFunction(filtroAplicadoJS, nil, chromedp.WithPollingArgs(f.campo, f.valor))); err != nil {
			return fmt.Errorf("filtro %s=%s não confirmado: %w", f.campo, f.valor, err)
		}
	}
	for _, f := range c.filtros(reportSelector) {
		aplicado, err := filtroAplicado(ctx, f)
		if err != nil {
			return err
		}
		if !aplicado {
			return fmt.Errorf("filtro %s=%s perdido após seleção", f.campo, f.valor)
		}
	}
	return nil
}

const (
	searchInputSelector = `input[data-testid="search-input-field"][placeholder="Pesquisar na caixa de listagem"]`
	tentativasFiltro    = 3
	pausaFiltro         = 2 * time.Second
)

// O Qlik recria os elementos do painel enquanto recalcula uma seleção. Se isso ocorre entre a
// localização do nó e o clique do chromedp, o CDP responde "Could not find node with given id".
func (c crawler) aplicaFiltro(ctx context.Context, f filtro) error {
	for tentativa := 1; ; tentativa++ {
		err := c.selecionaFiltro(ctx, f.seletor, f.valor)
		if err == nil {
			return nil
		}
		var cdpErr *cdproto.Error
		if !errors.As(err, &cdpErr) || cdpErr.Message != "Could not find node with given id" || tentativa == tentativasFiltro {
			return fmt.Errorf("erro selecionando %s=%s: %w", f.campo, f.valor, err)
		}
		log.Printf("Painel recriado ao selecionar %s=%s (tentativa %d/%d): %v", f.campo, f.valor, tentativa, tentativasFiltro, err)
		// Esc fecha a caixa de listagem descartando a seleção pendente.
		if err := chromedp.Run(ctx,
			chromedp.KeyEvent(kb.Escape),
			chromedp.WaitNotPresent(searchInputSelector, chromedp.ByQuery),
			chromedp.Sleep(pausaFiltro),
		); err != nil {
			return fmt.Errorf("erro cancelando seleção de %s=%s: %w", f.campo, f.valor, err)
		}
		// Se a confirmação chegou a ocorrer, um novo clique desmarcaria o valor.
		aplicado, err := filtroAplicado(ctx, f)
		if err != nil {
			return err
		}
		if aplicado {
			return nil
		}
	}
}

func (c crawler) selecionaFiltro(ctx context.Context, headerSelector, value string) error {
	const confirmSelector = `button[data-testid="actions-toolbar-confirm"][title="Confirmar seleção"]`
	// Localiza, na caixa de listagem aberta, o item cujo título corresponde ao valor pesquisado para selecioná-lo antes da confirmação.
	resultSelector := fmt.Sprintf(
		`//*[contains(@class, "listbox-popover")]//*[@data-testid="listbox.item"][.//*[@title=%q]]`,
		strings.ToUpper(strings.TrimSpace(value)),
	)

	return chromedp.Run(ctx,
		chromedp.Click(headerSelector, chromedp.ByQuery),
		chromedp.WaitVisible(searchInputSelector, chromedp.ByQuery),
		chromedp.Click(searchInputSelector, chromedp.ByQuery),
		chromedp.SendKeys(searchInputSelector, value, chromedp.ByQuery),
		chromedp.WaitVisible(resultSelector, chromedp.BySearch),
		chromedp.Click(resultSelector, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.WaitVisible(confirmSelector, chromedp.ByQuery),
		chromedp.Click(confirmSelector, chromedp.ByQuery),
		chromedp.WaitNotPresent(searchInputSelector, chromedp.ByQuery),
	)
}

// exportaExcel abre o menu da visualização indicada, confirma a exportação, baixa, fecha o modal e renomeia o arquivo.
func (c crawler) exportaExcel(ctx context.Context, reportSelector, fName string) error {
	const (
		exportarDadosSelector = `//*[normalize-space(text())='Exportar dados']`
		exportDialogSelector  = `[role="dialog"]`
		exportButtonSelector  = `button[tid="table-export"][name="confirm"]`
		downloadLinkSelector  = `a.export-url[q-translation="Export.CompletedLink"][href*="/tempcontent/"]`
		closeButtonSelector   = `button[q-translation="Common.Close"][ng-click="close()"]`
	)
	if err := c.garanteFiltros(ctx, reportSelector); err != nil {
		return status.NewError(status.ConnectionError, fmt.Errorf("erro garantindo filtros: %w", err))
	}
	if err := c.garanteAba(ctx, reportSelector); err != nil {
		return status.NewError(status.ConnectionError, fmt.Errorf("erro selecionando relatório após os filtros: %w", err))
	}
	// A barra de seleções pode atualizar antes da tabela. Aguarda os dados do recorte,
	// sem um Sleep fixo que exportaria resultados antigos em um painel mais lento.
	var tabelaSelector string
	if err := chromedp.Run(ctx, chromedp.PollFunction(`(seletor, tribunal, referencia, controle) => {
		const titulo = document.querySelector(seletor);
		const objeto = titulo && titulo.closest('article.qv-object');
		if (!objeto) return false;
		const aba = titulo.closest('[role="tab"]');
		if (aba && aba.getAttribute('aria-selected') !== 'true') return false;
		const tabela = aba ? objeto.querySelector('article.qv-object-' + CSS.escape(aba.getAttribute('aria-controls'))) : objeto;
		if (!tabela) return false;
		const linhas = Array.from(tabela.querySelectorAll('tbody tr.qv-st-data-row')).filter(l => l.getClientRects().length);
		const pronto = linhas.length > 0 && linhas.every(linha => {
			const celulas = linha.querySelectorAll('td .qv-st-value');
			return celulas.length > 2 && celulas[0].getAttribute('title') === tribunal &&
				celulas[controle ? 1 : 2].getAttribute('title') === referencia;
		});
		const classe = Array.from(tabela.classList).find(c => c.startsWith('qv-object-') && c !== 'qv-object-table');
		return pronto && classe ? 'article.' + CSS.escape(classe) : false;
	}`, &tabelaSelector, chromedp.WithPollingArgs(reportSelector, strings.ToUpper(c.court), c.month+"/"+c.year, reportSelector == identificadorControle))); err != nil {
		return status.NewError(status.ConnectionError, fmt.Errorf("tabela não atualizou para o recorte solicitado: %w", err))
	}

	var nodes []*cdp.Node
	err := chromedp.Run(ctx,
		chromedp.Nodes(tabelaSelector, &nodes, chromedp.ByQuery, chromedp.AtLeast(1)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.MouseClickNode(nodes[0], chromedp.ButtonRight).Do(ctx)
		}),
		chromedp.WaitVisible(exportarDadosSelector, chromedp.BySearch),
		chromedp.Click(exportarDadosSelector, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.WaitVisible(exportDialogSelector, chromedp.ByQuery),
		chromedp.WaitVisible(exportButtonSelector, chromedp.ByQuery),
		chromedp.Click(exportButtonSelector, chromedp.ByQuery),
		chromedp.WaitVisible(downloadLinkSelector, chromedp.ByQuery),
	)
	if err != nil {
		return status.NewError(status.ConnectionError, fmt.Errorf("erro exportando planilha: %v", err))
	}

	if err := c.baixaPlanilha(ctx, downloadLinkSelector, fName); err != nil {
		return err
	}
	if err := chromedp.Run(ctx,
		chromedp.Click(closeButtonSelector, chromedp.ByQuery),
		chromedp.WaitNotPresent(exportButtonSelector, chromedp.ByQuery),
	); err != nil {
		return status.NewError(status.ConnectionError, fmt.Errorf("erro fechando exportação: %w", err))
	}
	return c.validaRecorte(fName)
}

func (c crawler) clicaAba(ctx context.Context, seletor string) error {
	return chromedp.Run(ctx, chromedp.Click(seletor, chromedp.NodeVisible))
}

// Alterar os filtros pode fazer o contêiner voltar à primeira aba.
func (c crawler) garanteAba(ctx context.Context, seletor string) error {
	if seletor == identificadorControle {
		return nil
	}
	var ativa bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`document.querySelector(%q)?.closest('[role="tab"]')?.getAttribute('aria-selected') === 'true'`, seletor), &ativa)); err != nil {
		return err
	}
	if ativa {
		return nil
	}
	return c.clicaAba(ctx, seletor)
}

func (c crawler) configuraDownloads() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// O link do Qlik abre outra aba. Configura e escuta o navegador inteiro.
		return browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(c.output).WithEventsEnabled(true).
			Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Browser))
	})
}

// DOWNLOAD_TIMEOUT limita a transferência, em vez de impor uma pausa a cada arquivo.
// AllowAndName grava o GUID: nunca renomeamos um arquivo anterior como se fosse novo.
func (c crawler) baixaPlanilha(ctx context.Context, link, destino string) error {
	downloadCtx, cancel := context.WithTimeout(ctx, c.downloadTimeout)
	defer cancel()
	concluido := make(chan *browser.EventDownloadProgress, 1)
	var guid string
	chromedp.ListenBrowser(downloadCtx, func(event interface{}) {
		switch e := event.(type) {
		case *browser.EventDownloadWillBegin:
			guid = e.GUID
		case *browser.EventDownloadProgress:
			if e.GUID == guid && (e.State == browser.DownloadProgressStateCompleted || e.State == browser.DownloadProgressStateCanceled) {
				select {
				case concluido <- e:
				default:
				}
			}
		}
	})
	if err := chromedp.Run(downloadCtx, chromedp.Click(link, chromedp.ByQuery)); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return status.NewError(status.DeadlineExceeded, fmt.Errorf("tempo esgotado iniciando download: %w", err))
		}
		return status.NewError(status.ConnectionError, fmt.Errorf("erro iniciando download: %w", err))
	}
	select {
	case e := <-concluido:
		if e.State == browser.DownloadProgressStateCanceled {
			return status.NewError(status.ConnectionError, fmt.Errorf("download %s cancelado", e.GUID))
		}
		if err := os.Rename(filepath.Join(c.output, e.GUID), destino); err != nil {
			return status.NewError(status.SystemError, fmt.Errorf("erro renomeando download %s: %w", e.GUID, err))
		}
		return nil
	case <-downloadCtx.Done():
		return status.NewError(status.DeadlineExceeded, fmt.Errorf("download de %s não concluído: %w", destino, downloadCtx.Err()))
	}
}

// A verificação no XLSX também rejeita uma tabela ainda desatualizada após a seleção.
func (c crawler) validaRecorte(path string) error {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return status.NewError(status.InvalidFile, fmt.Errorf("erro abrindo planilha: %w", err))
	}
	defer f.Close()
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return status.NewError(status.InvalidFile, fmt.Errorf("erro lendo planilha: %w", err))
	}
	if len(rows) < 2 {
		return status.NewError(status.DataUnavailable, fmt.Errorf("planilha %s sem registros", path))
	}
	tribunal, referencia := -1, -1
	for i, header := range rows[0] {
		switch header {
		case "Tribunal":
			tribunal = i
		case "Mês/Ano", "Mês/Ano Ref.":
			referencia = i
		}
	}
	if tribunal < 0 || referencia < 0 {
		return status.NewError(status.InvalidFile, fmt.Errorf("planilha %s sem colunas de tribunal/referência", path))
	}
	for i, row := range rows[1:] {
		if tribunal >= len(row) || referencia >= len(row) || !strings.EqualFold(row[tribunal], c.court) || row[referencia] != c.month+"/"+c.year {
			return status.NewError(status.InvalidFile, fmt.Errorf("planilha %s contém linha %d fora de %s/%s/%s", path, i+2, c.court, c.month, c.year))
		}
	}
	return nil
}

func verificaErro(err error) error {
	// Verificamos se o erro é de conexão ou 'context deadline exceeded',
	// caso contrário, será retornado status Unknown (6)
	if strings.Contains(err.Error(), "ERR_CONNECTION_CLOSED") || strings.Contains(err.Error(), "ERR_CONNECTION_RESET") {
		err = status.NewError(status.ConnectionError, err)
	} else if strings.Contains(err.Error(), "context deadline exceeded") {
		err = status.NewError(status.DeadlineExceeded, err)
	}
	return err
}

func validate(c crawler) error {
	xlsx := filepath.Join(c.output, fmt.Sprintf("controle-de-arquivos-%s-%s-%s.xlsx", c.court, c.year, c.month))
	file, err := excelize.OpenFile(xlsx)
	if err != nil {
		return status.NewError(status.InvalidFile, fmt.Errorf("erro abrindo planilha: %w", err))
	}
	defer file.Close()
	rows, err := file.GetRows("Sheet1")
	if err != nil {
		return status.NewError(status.InvalidFile, fmt.Errorf("erro lendo planilha: %w", err))
	}
	filenames := []string{
		fmt.Sprintf("%s_%s_%s.xls", c.court, c.month, c.year[2:]),
		fmt.Sprintf("%s_%s_%s.xls", c.court, strings.TrimLeft(c.month, "0"), c.year[2:]),
	}

	var ok bool
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		f := strings.ToLower(row[2])
		if strings.Contains(f, filenames[0]) || strings.Contains(f, filenames[1]) {
			ok = true
		}
	}
	if !ok {
		return status.NewError(status.DataUnavailable, fmt.Errorf("não há planilhas a serem baixadas"))
	}
	return nil
}

// Esta função verifica se os dados estão agregados
func validateSourceFile(file string) error {
	f, err := excelize.OpenFile(file)
	if err != nil {
		return status.NewError(status.InvalidFile, fmt.Errorf("erro abrindo planilha: %w", err))
	}
	defer f.Close()
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return status.NewError(status.InvalidFile, fmt.Errorf("erro lendo planilha: %w", err))
	}

	// Quando os dados estão agregados, a planilha de contracheques possui apenas 2 linhas: o cabeçalho e 1 linha com dados agregados.
	// Além disso, quando os dados estão agregados, o órgão tende a colocar '0' como nome do magistrado.
	// Aqui verificamos se a planilha possui apenas 1 linha de dados ou se o nome do magistrado é '0'
	if len(rows) <= 2 || (len(rows[1]) > 1 && rows[1][1] == "0") {
		return status.NewError(status.DataUnavailable, fmt.Errorf("dados agregados"))
	}
	for _, coluna := range []string{"Cargo", "Lotação"} {
		presente := false
		for _, header := range rows[0] {
			presente = presente || header == coluna
		}
		if !presente {
			return status.NewError(status.InvalidFile, fmt.Errorf("contracheque sem coluna %s", coluna))
		}
	}

	return nil
}
