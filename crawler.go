package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

const (
	timeout             = 5 * time.Minute
	tempoEsperaDownload = 15 * time.Second
	tempoAcao           = 5 * time.Second

	direitosPessoaisXPATH = "/html/body/div[5]/div/div[31]/div[2]/table/tbody/tr/td"
	indenizacoesXPATH     = "/html/body/div[5]/div/div[25]/div[2]/table/tbody/tr/td"
	verbasXPATH           = "/html/body/div[5]/div/div[28]/div[2]/table/tbody/tr/td"
	controleXPATH         = "/html/body/div[5]/div/div[52]/div[2]/table/tbody/tr/td"
)

func crawl(court, year, month, output string) ([]string, error) {
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

	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Printf("Realizando seleção (%s/%s/%s)...", court, month, year)
	if err := selectionaOrgaoMesAno(ctx, court, year, month, output); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Seleção realizada com sucesso!\n")

	// NOTA IMPORTANTE: os prefixos dos nomes dos arquivos tem que ser igual
	// ao esperado no parser CNJ.

	// O contra cheque é a aba padrão, por isso não precisa haver clique.
	cqFname := filepath.Join(output, fmt.Sprintf("contracheque-%s-%s-%s.xlsx", court, year, month))
	log.Printf("Fazendo download do contracheque (%s)...", cqFname)
	if err := exportaExcel(ctx, output, cqFname); err != nil {
		log.Fatalf("Erro fazendo download do contracheque: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Direitos pessoais
	dpFname := filepath.Join(output, fmt.Sprintf("direitos-pessoais-%s-%s-%s.xlsx", court, year, month))
	log.Printf("Fazendo download dos direitos pessoais (%s)...", dpFname)
	if err := clicaAba(ctx, direitosPessoaisXPATH); err != nil {
		log.Fatalf("Erro clicando na aba de direitos pessoais: %v", err)
	}
	if err := exportaExcel(ctx, output, dpFname); err != nil {
		log.Fatalf("Erro fazendo download dos direitos pessoais: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// // Indenizações
	iFname := filepath.Join(output, fmt.Sprintf("indenizacoes-%s-%s-%s.xlsx", court, year, month))
	log.Printf("Fazendo download das indenizações (%s)...", iFname)
	if err := clicaAba(ctx, indenizacoesXPATH); err != nil {
		log.Fatalf("Erro clicando na aba de indenizações: %v", err)
	}
	if err := exportaExcel(ctx, output, iFname); err != nil {
		log.Fatalf("Erro fazendo download dos indenizações: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Verbas
	deFname := filepath.Join(output, fmt.Sprintf("direitos-eventuais-%s-%s-%s.xlsx", court, year, month))
	log.Printf("Fazendo download das verbas (%s)...", deFname)
	if err := clicaAba(ctx, verbasXPATH); err != nil {
		log.Fatalf("Erro clicando na aba de direitos eventuais: %v", err)
	}
	if err := exportaExcel(ctx, output, deFname); err != nil {
		log.Fatalf("Erro fazendo download dos direitos eventuais: %v", err)
	}
	log.Printf("Download das direitos eventuais realizado com sucesso!\n")

	// Planilha de controle
	ceFname := filepath.Join(output, fmt.Sprintf("controle-de-arquivos-%s-%s-%s.xlsx", court, year, month))
	log.Printf("Fazendo download das controle de arquivos (%s)...", ceFname)
	if err := clicaAba(ctx, controleXPATH); err != nil {
		log.Fatalf("Erro clicando na aba de controle de arquivos: %v", err)
	}
	if err := exportaExcel(ctx, output, ceFname); err != nil {
		log.Fatalf("Erro fazendo download docontrole de arquivos: %v", err)
	}
	log.Printf("Download do controle de arquivos realizado com sucesso!\n")

	// Retorna caminhos completos dos arquivos baixados.
	return []string{cqFname, dpFname, deFname, iFname, ceFname}, nil
}

func selectionaOrgaoMesAno(ctx context.Context, court, year, month, output string) error {
	const (
		pathRoot = "/html/body/div[2]/input"
		baseURL  = "https://paineis.cnj.jus.br/QvAJAXZfc/opendoc.htm?document=qvw_l%2FPainelCNJ.qvw&host=QVS%40neodimio03&anonymous=true&sheet=shPORT63Relatorios"
	)
	return chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.WaitVisible(`//*[@title='Tribunal']`, chromedp.BySearch),
		chromedp.Sleep(tempoAcao),

		// Seleciona o orgão
		chromedp.Click(`//*[@title='Tribunal']//*[@title='Pesquisar']`, chromedp.BySearch, chromedp.NodeReady),
		chromedp.WaitVisible(pathRoot),
		chromedp.SetValue(pathRoot, court),
		chromedp.SendKeys(pathRoot, kb.Enter, chromedp.NodeSelected),
		chromedp.Sleep(tempoAcao),

		// Seleciona o ano
		chromedp.Click(`//*[@title='Ano']//*[@title='Pesquisar']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.WaitVisible(pathRoot),
		chromedp.SetValue(pathRoot, year),
		chromedp.SendKeys(pathRoot, kb.Enter),
		chromedp.Sleep(tempoAcao),

		// Seleciona o mes
		chromedp.Click(`//*[@title='Mês Referencia']//*[@title='Pesquisar']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.WaitVisible(pathRoot),
		chromedp.SetValue(pathRoot, month),
		chromedp.SendKeys(pathRoot, kb.Enter),
		chromedp.Sleep(tempoAcao),

		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(output).
			WithEventsEnabled(true),
	)
}

// exportaExcel clica no botão correto para exportar para excel, espera um tempo para download renomeia o arquivo.
func exportaExcel(ctx context.Context, output, fName string) error {
	err := chromedp.Run(ctx,
		chromedp.Click(`//*[@title='Enviar para Excel']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(tempoAcao),
	)
	if err != nil {
		return fmt.Errorf("erro clicando no botão de download: %v", err)
	}

	time.Sleep(tempoEsperaDownload)

	if err := nomeiaDownload(output, fName); err != nil {
		return fmt.Errorf("erro renomeando arquivo (%s): %v", fName, err)
	}
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return fmt.Errorf("download do arquivo de %s não realizado", fName)
	}
	return nil
}

// clicaAba clica na aba referenciada pelo XPATH passado como parâmetro.
// Também espera até o título Tribunal estar visível.
func clicaAba(ctx context.Context, xpath string) error {
	return chromedp.Run(ctx,
		chromedp.Click(xpath),
		chromedp.Sleep(tempoAcao),
	)
}

// nomeiaDownload dá um nome ao último arquivo modificado dentro do diretório
// passado como parâmetro nomeiaDownload dá pega um arquivo
func nomeiaDownload(output, fName string) error {
	// Identifica qual foi o ultimo arquivo
	files, err := os.ReadDir(output)
	if err != nil {
		return fmt.Errorf("erro lendo diretório %s: %v", output, err)
	}
	var newestFPath string
	var newestTime int64 = 0
	for _, f := range files {
		fPath := filepath.Join(output, f.Name())
		fi, err := os.Stat(fPath)
		if err != nil {
			return fmt.Errorf("erro obtendo informações sobre arquivo %s: %v", fPath, err)
		}
		currTime := fi.ModTime().Unix()
		if currTime > newestTime {
			newestTime = currTime
			newestFPath = fPath
		}
	}
	// Renomeia o ultimo arquivo modificado.
	if err := os.Rename(newestFPath, fName); err != nil {
		return fmt.Errorf("erro renomeando último arquivo modificado (%s)->(%s): %v", newestFPath, fName, err)
	}
	return nil
}
