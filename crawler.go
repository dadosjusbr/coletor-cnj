package main

import (
	"context"
	"log"
	"time"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
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
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var result string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://paineis.cnj.jus.br/QvAJAXZfc/opendoc.htm?document=qvw_l%2FPainelCNJ.qvw&host=QVS%40neodimio03&anonymous=true&sheet=shPORT63Relatorios`),
		chromedp.WaitVisible(`//*[@title='Tribunal']`, chromedp.BySearch),

		chromedp.Click(`//*[@title='Tribunal']//*[@title='Pesquisar']`, chromedp.NodeNotVisible),
		chromedp.WaitVisible(`/html/body/div[2]/input`),
		chromedp.SetValue(`/html/body/div[2]/input`, "TJRJ"),
		chromedp.SendKeys(`/html/body/div[2]/input`, kb.Enter),
		
		chromedp.Click(`//*[@title='Ano']//*[@title='Pesquisar']`, chromedp.NodeNotVisible),
		chromedp.WaitVisible(`/html/body/div[2]/input`),
		chromedp.SetValue(`/html/body/div[2]/input`, "2018"),
		chromedp.SendKeys(`/html/body/div[2]/input`, kb.Enter),

		chromedp.Click(`//*[@title='Mês Referencia']//*[@title='Pesquisar']`, chromedp.NodeNotVisible),
		chromedp.WaitVisible(`/html/body/div[2]/input`),
		chromedp.SetValue(`/html/body/div[2]/input`, "01"),
		chromedp.SendKeys(`/html/body/div[2]/input`, kb.Enter),

		chromedp.WaitVisible(`//*[@id="30"]`),

		chromedp.Stop(),
	)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("result %q", result)
}
