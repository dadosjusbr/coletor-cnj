package main

import (
	"context"
	"log"
	// "time"
	"github.com/chromedp/chromedp"
)

func main() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.DisableGPU,
        chromedp.Flag("headless", false),
    )
	alctor, cancel := chromedp.NewExecAllocator(
		context.Background(),
		opts...
	)
	defer cancel()
	ctx, cancel := chromedp.NewContext(
		alctor,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()
	// ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
 	// defer cancel()
	var result string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://paineis.cnj.jus.br/QvAJAXZfc/opendoc.htm?document=qvw_l%2FPainelCNJ.qvw&host=QVS%40neodimio03&anonymous=true&sheet=shPORT63Relatorios`),
		chromedp.WaitVisible(`body < #PageContainer`),
		chromedp.Stop(),
	)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("result %q", result)
}