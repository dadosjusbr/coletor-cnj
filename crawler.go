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
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/dadosjusbr/status"
	"github.com/xuri/excelize/v2"
)

const (
	direitosPessoaisXPATH = "/html/body/div[5]/div/div[31]/div[2]/table/tbody/tr/td"
	indenizacoesXPATH     = "/html/body/div[5]/div/div[25]/div[2]/table/tbody/tr/td"
	verbasXPATH           = "/html/body/div[5]/div/div[28]/div[2]/table/tbody/tr/td"
	controleXPATH         = "/html/body/div[5]/div/div[55]/div[2]/table/tbody/tr/td"
	contrachequeXPATH     = "/html/body/div[5]/div/div[49]/div[2]/table/tbody/tr/td"
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
	if err := c.clicaAba(ctx, controleXPATH); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, ceFname); err != nil {
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
	if err := c.clicaAba(ctx, contrachequeXPATH); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, cqFname); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Direitos pessoais
	dpFname := c.downloadFilePath("direitos-pessoais")
	log.Printf("Fazendo download dos direitos pessoais (%s)...", dpFname)
	if err := c.clicaAba(ctx, direitosPessoaisXPATH); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, dpFname); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// // Indenizações
	iFname := c.downloadFilePath("indenizacoes")
	log.Printf("Fazendo download das indenizações (%s)...", iFname)
	if err := c.clicaAba(ctx, indenizacoesXPATH); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, iFname); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Verbas
	deFname := c.downloadFilePath("direitos-eventuais")
	log.Printf("Fazendo download dos direitos eventuais (%s)...", deFname)
	if err := c.clicaAba(ctx, verbasXPATH); err != nil {
		status.ExitFromError(err)
	}
	if err := c.exportaExcel(ctx, deFname); err != nil {
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
		pathRoot = "/html/body/div[2]/input"
		baseURL  = "https://paineis.cnj.jus.br/QvAJAXZfc/opendoc.htm?document=qvw_l%2FPainelCNJ.qvw&host=QVS%40neodimio03&anonymous=true&sheet=shPORT63Relatorios"
	)
	return chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.WaitVisible(`//*[@title='Tribunal']`, chromedp.BySearch),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona o orgão
		chromedp.Click(`//*[@title='Tribunal']//*[@title='Pesquisar']`, chromedp.BySearch, chromedp.NodeReady),
		chromedp.WaitVisible(pathRoot),
		chromedp.SetValue(pathRoot, c.court),
		chromedp.SendKeys(pathRoot, kb.Enter, chromedp.NodeSelected),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona o ano
		chromedp.Click(`//*[@title='Ano']//*[@title='Pesquisar']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.WaitVisible(pathRoot),
		chromedp.SetValue(pathRoot, c.year),
		chromedp.SendKeys(pathRoot, kb.Enter),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona o mes
		chromedp.Click(`//*[@title='Mês Referencia']//*[@title='Pesquisar']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.WaitVisible(pathRoot),
		chromedp.SetValue(pathRoot, c.month),
		chromedp.SendKeys(pathRoot, kb.Enter),
		chromedp.Sleep(c.timeBetweenSteps),

		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(c.output).
			WithEventsEnabled(true),
	)
}

// exportaExcel clica no botão correto para exportar para excel, espera um tempo para download renomeia o arquivo.
func (c crawler) exportaExcel(ctx context.Context, fName string) error {
	err := chromedp.Run(ctx,
		chromedp.Click(`//*[@title='Enviar para Excel']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
	)
	if err != nil {
		return status.NewError(status.ConnectionError, fmt.Errorf("erro clicando no botão de download: %v", err))
	}

	time.Sleep(c.downloadTimeout)

	if err := nomeiaDownload(c.output, fName); err != nil {
		status.ExitFromError(err)
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
		return status.NewError(status.DataUnavailable, fmt.Errorf("Sem planilhas baixadas."))
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
	xlsx := fmt.Sprintf("controle-de-arquivos-%s-%s-%s.xlsx", c.court, c.year, c.month)
	file, err := excelize.OpenFile(xlsx)
	if err != nil {
		return status.NewError(status.SystemError, fmt.Errorf("erro abrindo planilha"))
	}
	rows, err := file.GetRows("Sheet1")
	if err != nil {
		return status.NewError(status.SystemError, fmt.Errorf("erro lendo planilha"))
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
