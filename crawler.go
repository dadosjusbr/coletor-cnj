package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/dadosjusbr/status"
	"github.com/xuri/excelize/v2"
)

const (
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

	log.Printf("Realizando seleção (%s/%s/%s)...", c.court, c.month, c.year)
	if err := c.selectionaOrgaoMesAno(ctx); err != nil {
		verificaErro(err)
	}
	log.Printf("Seleção realizada com sucesso!\n")

	// NOTA IMPORTANTE: os prefixos dos nomes dos arquivos tem que ser igual
	// ao esperado no parser CNJ.

	// Planilha de controle
	ceFname := c.downloadFilePath("controle-de-arquivos")
	log.Printf("Fazendo download do controle de arquivos (%s)...", ceFname)
	if err := c.clicaAba(ctx, botaoControleXPATH); err != nil {
		status.ExitFromError(err)
	}
	if err := c.clicaAba(ctx, identificadorControle); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, identificadorControle, ceFname); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Download do controle de arquivos realizado com sucesso!\n")

	// Verificando se tem dados
	if err := validate(c); err != nil {
		status.ExitFromError(err)
	}

	// Contracheque
	cqFname := c.downloadFilePath("contracheque")
	log.Printf("Fazendo download do contracheque (%s)...", cqFname)
	if err := c.clicaAba(ctx, botaoRelatoriosXPATH); err != nil {
		status.ExitFromError(err)
	}
	// Selecionando contracheque...
	if err := c.clicaAba(ctx, contrachequeTabSelector); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, contrachequeTabSelector, cqFname); err != nil {
		status.ExitFromError(err)
	}

	// Verificando se os dados estão agregados
	if err := validateSourceFile(cqFname); err != nil {
		status.ExitFromError(err)
	}

	log.Printf("Download realizado com sucesso!\n")

	// Direitos pessoais
	dpFname := c.downloadFilePath("direitos-pessoais")
	log.Printf("Fazendo download dos direitos pessoais (%s)...", dpFname)
	if err := c.clicaAba(ctx, direitosPessoaisTabSelector); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, direitosPessoaisTabSelector, dpFname); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Indenizações
	iFname := c.downloadFilePath("indenizacoes")
	log.Printf("Fazendo download das indenizações (%s)...", iFname)
	if err := c.clicaAba(ctx, indenizacoesTabSelector); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, indenizacoesTabSelector, iFname); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Verbas
	deFname := c.downloadFilePath("direitos-eventuais")
	log.Printf("Fazendo download dos direitos eventuais (%s)...", deFname)
	if err := c.clicaAba(ctx, direitosEventuaisTabSelector); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, direitosEventuaisTabSelector, deFname); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Download dos direitos eventuais realizado com sucesso!\n")

	// Retorna caminhos completos dos arquivos baixados.
	return []string{cqFname, dpFname, deFname, iFname, ceFname}, nil
}

func (c crawler) downloadFilePath(prefix string) string {
	return filepath.Join(c.output, fmt.Sprintf("%s-%s-%s-%s.xlsx", prefix, c.court, c.year, c.month))
}

func (c crawler) selectionaOrgaoMesAno(ctx context.Context) error {
	const (
		// Seletor do orgão (tribunal).
		tribunalSelector = `h6[aria-label="Tribunal"]`

		// Seletor do ano e campo de pesquisa.
		anoSelector = `h6[aria-label="Ano"]`

		// Seletor do mes e campo de pesquisa.
		mesSelector = `h6[aria-label="Mês Referência"]`

		baseURL = "https://paineisanalytics.cnj.jus.br/single/?appid=8ccb93ca-848f-4ffe-bf9b-96d57878c4d7&sheet=a710d8e5-fddb-4fa7-9098-4dbca30a391a&lang=pt-BR&theme=Mix_Theme&opt=ctxmenu,currsel"
	)
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1920, 1080),
		chromedp.Navigate(baseURL),
		chromedp.WaitVisible(tribunalSelector, chromedp.ByQuery),
		chromedp.Sleep(c.timeBetweenSteps),
	); err != nil {
		return err
	}

	if err := c.selecionaFiltro(ctx, tribunalSelector, c.court); err != nil {
		return err
	}
	if err := c.selecionaFiltro(ctx, anoSelector, c.year); err != nil {
		return err
	}
	if err := c.selecionaFiltro(ctx, mesSelector, c.month); err != nil {
		return err
	}

	return chromedp.Run(ctx,
		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(c.output).
			WithEventsEnabled(true),
	)
}

func (c crawler) selecionaFiltro(ctx context.Context, headerSelector, value string) error {
	const (
		searchInputSelector = `input[data-testid="search-input-field"][placeholder="Pesquisar na caixa de listagem"]`
		confirmSelector     = `button[data-testid="actions-toolbar-confirm"][title="Confirmar seleção"]`
	)
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
		chromedp.Sleep(c.timeBetweenSteps),
		chromedp.WaitVisible(resultSelector, chromedp.BySearch),
		chromedp.Click(resultSelector, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
		chromedp.WaitVisible(confirmSelector, chromedp.ByQuery),
		chromedp.Click(confirmSelector, chromedp.ByQuery),
		chromedp.Sleep(c.timeBetweenSteps),
	)
}

// exportaExcel abre o menu da visualização indicada, confirma a exportação,
// baixa, fecha o modal e renomeia o arquivo.
func (c crawler) exportaExcel(ctx context.Context, reportSelector, fName string) error {
	const (
		exportarDadosSelector = `//*[normalize-space(text())='Exportar dados']`
		exportDialogSelector  = `[role="dialog"]`
		exportButtonSelector  = `button[tid="table-export"][name="confirm"]`
		downloadLinkSelector  = `a.export-url[q-translation="Export.CompletedLink"][href*="/tempcontent/"]`
		closeButtonSelector   = `button[q-translation="Common.Close"][ng-click="close()"]`
	)

	var nodes []*cdp.Node
	err := chromedp.Run(ctx,
		chromedp.Nodes(reportSelector, &nodes, chromedp.BySearch, chromedp.AtLeast(1)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.MouseClickNode(nodes[0], chromedp.ButtonRight).Do(ctx)
		}),
		chromedp.WaitVisible(exportarDadosSelector, chromedp.BySearch),
		chromedp.Click(exportarDadosSelector, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.WaitVisible(exportDialogSelector, chromedp.ByQuery),
		chromedp.Sleep(c.timeBetweenSteps),
		chromedp.WaitVisible(exportButtonSelector, chromedp.ByQuery),
		chromedp.Click(exportButtonSelector, chromedp.ByQuery),
		chromedp.Sleep(c.timeBetweenSteps),
		chromedp.WaitVisible(downloadLinkSelector, chromedp.ByQuery),
		chromedp.Click(downloadLinkSelector, chromedp.ByQuery),
		chromedp.Sleep(c.timeBetweenSteps),
		chromedp.WaitVisible(closeButtonSelector, chromedp.ByQuery),
		chromedp.Click(closeButtonSelector, chromedp.ByQuery),
		chromedp.Sleep(c.timeBetweenSteps),
	)
	if err != nil {
		return status.NewError(status.ConnectionError, fmt.Errorf("erro exportando planilha: %v", err))
	}

	time.Sleep(c.downloadTimeout)

	if err := nomeiaDownload(c.output, fName); err != nil {
		return err
	}
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return status.NewError(status.DataUnavailable, fmt.Errorf("download do arquivo de %s não realizado", fName))
	}

	return nil
}

// clicaAba clica na aba referenciada pelo XPATH passado como parâmetro.
// Também espera até o título Tribunal estar visível.
func (c crawler) clicaAba(ctx context.Context, xpath string) error {
	return chromedp.Run(ctx,
		chromedp.Click(xpath),
		chromedp.Sleep(c.timeBetweenSteps),
	)
}

// nomeiaDownload dá um nome ao último arquivo modificado dentro do diretório
// passado como parâmetro nomeiaDownload dá pega um arquivo
func nomeiaDownload(output, fName string) error {
	// Identifica qual foi o ultimo arquivo
	files, err := os.ReadDir(output)
	if err != nil {
		return status.NewError(status.SystemError, fmt.Errorf("erro lendo diretório %s: %v", output, err))
	}
	var newestFPath string
	var newestTime int64 = 0
	for _, f := range files {
		fPath := filepath.Join(output, f.Name())
		fi, err := os.Stat(fPath)
		if err != nil {
			return status.NewError(status.SystemError, fmt.Errorf("erro obtendo informações sobre arquivo %s: %v", fPath, err))
		}
		currTime := fi.ModTime().Unix()
		if currTime > newestTime {
			newestTime = currTime
			newestFPath = fPath
		}
	}
	// Renomeia o ultimo arquivo modificado.
	if err := os.Rename(newestFPath, fName); err != nil {
		return status.NewError(status.DataUnavailable, fmt.Errorf("sem planilhas baixadas"))
	}
	return nil
}

func verificaErro(err error) {
	// Verificamos se o erro é de conexão ou 'context deadline exceeded',
	// caso contrário, será retornado status Unknown (6)
	if strings.Contains(err.Error(), "ERR_CONNECTION_CLOSED") || strings.Contains(err.Error(), "ERR_CONNECTION_RESET") {
		err = status.NewError(status.ConnectionError, err)
	} else if strings.Contains(err.Error(), "context deadline exceeded") {
		err = status.NewError(status.DeadlineExceeded, err)
	}
	status.ExitFromError(err)
}

func validate(c crawler) error {
	xlsx := filepath.Join(c.output, fmt.Sprintf("controle-de-arquivos-%s-%s-%s.xlsx", c.court, c.year, c.month))
	file, err := excelize.OpenFile(xlsx)
	if err != nil {
		return status.NewError(status.InvalidFile, fmt.Errorf("erro abrindo planilha: %w", err))
	}
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
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return status.NewError(status.InvalidFile, fmt.Errorf("erro lendo planilha: %w", err))
	}

	// Quando os dados estão agregados, a planilha de contracheques possui apenas 2 linhas: o cabeçalho e 1 linha com dados agregados.
	// Além disso, quando os dados estão agregados, o órgão tende a colocar '0' como nome do magistrado.
	// Aqui verificamos se a planilha possui apenas 1 linha de dados ou se o nome do magistrado é '0'
	if len(rows) == 2 || rows[1][1] == "0" {
		return status.NewError(status.DataUnavailable, fmt.Errorf("dados agregados"))
	}

	return nil
}
