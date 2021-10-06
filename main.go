package main

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	baseURL = "https://paineis.cnj.jus.br/QvAJAXZfc/opendoc.htm?document=qvw_l%2FPainelCNJ.qvw&host=QVS%40neodimio03&anonymous=true&sheet=shPORT63Relatorios"
	timeout = 30 * time.Second
)

func main() {
	// Cria instância do chrome.
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		chromedp.WithDebugf(log.Printf), // descomentar para debug.
	)
	defer cancel()

	// Configura timeout
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL),

		// Abrindo a barra de busca - Tribunal.
		chromedp.WaitVisible("//*[@title='Tribunal']", chromedp.BySearch),
		chromedp.Click("//*[@title='Tribunal']//*[@title='Pesquisar']", chromedp.BySearch), // Clica na parte de pesquisa, dentro da parte tribunal.

	)
	if err != nil {
		log.Fatal(err)
	}
}
