package main

import (
	"context"
	"log"
	"time"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/chromedp/cdproto/browser"
)

const (
	baseURL = "https://paineis.cnj.jus.br/QvAJAXZfc/opendoc.htm?document=qvw_l%2FPainelCNJ.qvw&host=QVS%40neodimio03&anonymous=true&sheet=shPORT63Relatorios"
	timeout = 60 * time.Second
	path_root = "/html/body/div[2]/input"
	month = "01"
	year = "2020"
	court = "tjrj"
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
	var result string
	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.WaitVisible(`//*[@title='Tribunal']`, chromedp.BySearch),
		chromedp.Sleep(10 * time.Second),

		//seleciona o orgão
		chromedp.Click(`//*[@title='Tribunal']//*[@title='Pesquisar']`, chromedp.BySearch),
		chromedp.WaitVisible(path_root),
		chromedp.SetValue(path_root, court),
		chromedp.SendKeys(path_root, kb.Enter),
		chromedp.Sleep(10 * time.Second),

		//seleciona o ano
		chromedp.Click(`//*[@title='Ano']//*[@title='Pesquisar']`, chromedp.BySearch),
		chromedp.WaitVisible(path_root),
		chromedp.SetValue(path_root, year),
		chromedp.SendKeys(path_root, kb.Enter),
		chromedp.Sleep(10 * time.Second),

		//seleciona o mes
		chromedp.Click(`//*[@title='Mês Referencia']//*[@title='Pesquisar']`, chromedp.BySearch),
		chromedp.WaitVisible(path_root),
		chromedp.SetValue(path_root, month),
		chromedp.SendKeys(path_root, kb.Enter),
		chromedp.Sleep(10 * time.Second),

		//altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath("./output").
			WithEventsEnabled(true),

		chromedp.Click(`//*[@title='Enviar para Excel']`, chromedp.BySearch),
		chromedp.Sleep(10 * time.Second),

		chromedp.Stop(),
	)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("result %q", result)
}
