package main

import (
	"os"
	"log"
	"fmt"
	"time"
	"context"
	"io/ioutil"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/chromedp/cdproto/browser"
)

const (
	baseURL   = "https://paineis.cnj.jus.br/QvAJAXZfc/opendoc.htm?document=qvw_l%2FPainelCNJ.qvw&host=QVS%40neodimio03&anonymous=true&sheet=shPORT63Relatorios"
	timeout   = 700 * time.Second
	path_root = "/html/body/div[2]/input"
	month     = "01"
	year      = "2020"
	court     = "tjrj"
	timezinho = 10
)

func main() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.Flag("headless", false),
	)
	alctor, cancel := chromedp.NewExecAllocator(
		context.Background(),
		opts...,
	)
	defer cancel()

	ctx, cancel := chromedp.NewContext(
		alctor,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	// Pagina do contracheque
	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.WaitVisible(`//*[@title='Tribunal']`, chromedp.BySearch),
		chromedp.Sleep(timezinho*time.Second),

		// Seleciona o orgão
		chromedp.Click(`//*[@title='Tribunal']//*[@title='Pesquisar']`, chromedp.BySearch, chromedp.NodeReady),
		chromedp.WaitVisible(path_root),
		chromedp.SetValue(path_root, court),
		chromedp.SendKeys(path_root, kb.Enter, chromedp.NodeSelected),
		chromedp.Sleep(timezinho*time.Second),

		// Seleciona o ano
		chromedp.Click(`//*[@title='Ano']//*[@title='Pesquisar']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.WaitVisible(path_root),
		chromedp.SetValue(path_root, year),
		chromedp.SendKeys(path_root, kb.Enter),
		chromedp.Sleep(timezinho*time.Second),

		// Seleciona o mes
		chromedp.Click(`//*[@title='Mês Referencia']//*[@title='Pesquisar']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.WaitVisible(path_root),
		chromedp.SetValue(path_root, month),
		chromedp.SendKeys(path_root, kb.Enter),
		chromedp.Sleep(timezinho*time.Second),

		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath("./output").
			WithEventsEnabled(true),

		chromedp.Click(`//*[@title='Enviar para Excel']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(20*time.Second),
		// Fecha o modal
		// chromedp.Click(`/html/body/div[15]/div[2]/button`),

	)

	iden_file("Contracheque")
	if err != nil {
		log.Println(err)
		return
	}

	// Pagina dos direitos pessoais
	err1 := chromedp.Run(ctx,
		chromedp.Sleep(timezinho*time.Second),
		chromedp.Click(`/html/body/div[5]/div/div[31]/div[2]/table/tbody/tr/td`),
		chromedp.WaitVisible(`//*[@title='Tribunal']`, chromedp.BySearch),
		chromedp.Sleep(timezinho*time.Second),

		
		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath("./output").
			WithEventsEnabled(true),

		chromedp.Click(`//*[@title='Enviar para Excel']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(20*time.Second),
		// Fecha o modal
		// chromedp.Click(`/html/body/div[15]/div[2]/button`),

		// chromedp.Stop(),
	)

	iden_file("Direitos pessoais")
	if err1 != nil {
		log.Println(err1)
		return
	}

	// Pagina indenizações
	err2 := chromedp.Run(ctx,
		chromedp.Sleep(timezinho*time.Second),
		chromedp.Click(`/html/body/div[5]/div/div[25]/div[2]/table/tbody/tr/td`),
		chromedp.WaitVisible(`//*[@title='Tribunal']`, chromedp.BySearch),
		chromedp.Sleep(timezinho*time.Second),

		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath("./output").
			WithEventsEnabled(true),

		chromedp.Click(`//*[@title='Enviar para Excel']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(20*time.Second),	
	)
	iden_file("Indenizações")
	if err2 != nil {
		log.Println(err2)
		return
	}

	// Verbas
	err3 := chromedp.Run(ctx,
		chromedp.Sleep(timezinho*time.Second),
		chromedp.Click(`/html/body/div[5]/div/div[28]/div[2]/table/tbody/tr/td`),
		chromedp.WaitVisible(`//*[@title='Tribunal']`, chromedp.BySearch),
		chromedp.Sleep(timezinho*time.Second),
		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath("./output").
			WithEventsEnabled(true),

		chromedp.Click(`//*[@title='Enviar para Excel']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(timezinho*time.Second),

		chromedp.Stop(),
	)
	iden_file("verbas")
	if err3 != nil {
		log.Println(err3)
		return
	}
}

func iden_file(flag string){
	// Identifica qual foi o ultimo arquivo
	dir := "output/"
	files, _ := ioutil.ReadDir(dir)
	var newestFile string
	var newestTime int64 = 0
	for _, f := range files {
		fi, err := os.Stat(dir + f.Name())
		if err != nil {
			fmt.Println(err)
		}
		currTime := fi.ModTime().Unix()
		if currTime > newestTime {
			newestTime = currTime
			newestFile = f.Name()
		}
	}
	// Renomeia o ultimo arquivo
	e := os.Rename("output/"+newestFile, "output/"+flag+"-"+court+"-"+year+"-"+month+".xslx")
	if e != nil {
		log.Fatal(e)
	}
}
