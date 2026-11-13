package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/dadosjusbr/status"
)

// navegadorTeste abre um Chrome headless local. Os testes que dependem dele só rodam
// com CNJ_BROWSER_TESTS=1.
func navegadorTeste(t *testing.T, limite time.Duration) context.Context {
	t.Helper()
	if os.Getenv("CNJ_BROWSER_TESTS") != "1" {
		t.Skip("defina CNJ_BROWSER_TESTS=1 para os testes com Chrome")
	}
	alloc, stop := chromedp.NewExecAllocator(context.Background(), append(chromedp.DefaultExecAllocatorOptions[:], chromedp.NoSandbox)...)
	t.Cleanup(stop)
	ctx, cancel := chromedp.NewContext(alloc)
	t.Cleanup(cancel)
	ctx, timeout := context.WithTimeout(ctx, limite)
	t.Cleanup(timeout)
	if err := chromedp.Run(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx
}

func paginaTeste(t *testing.T, ctx context.Context, html string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	t.Cleanup(server.Close)
	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL)); err != nil {
		t.Fatal(err)
	}
}

func TestVerificaErro(t *testing.T) {
	tests := []struct {
		nome   string
		err    error
		codigo status.Code // 0: o erro não deve virar *status.Error
	}{
		{"conexao encerrada por tempo", errors.New("page load error net::ERR_CONNECTION_TIMED_OUT"), status.ConnectionError},
		{"dns", errors.New("page load error net::ERR_NAME_NOT_RESOLVED"), status.ConnectionError},
		{"conexao fechada", errors.New("net::ERR_CONNECTION_CLOSED"), status.ConnectionError},
		{"prazo esgotado", fmt.Errorf("x: %w", context.DeadlineExceeded), status.DeadlineExceeded},
		{"prazo esgotado em texto", errors.New("waiting for function failed: context deadline exceeded"), status.DeadlineExceeded},
		{"status preservado", status.NewError(status.InvalidFile, errors.New("planilha")), status.InvalidFile},
		{"status com prazo esgotado dentro", status.NewError(status.ConnectionError, fmt.Errorf("filtro: %w", context.DeadlineExceeded)), status.ConnectionError},
		{"outro", errors.New("outro"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			var se *status.Error
			ok := errors.As(verificaErro(tt.err), &se)
			if tt.codigo == 0 {
				if ok {
					t.Fatalf("virou status %d", se.Code)
				}
				return
			}
			if !ok || se.Code != tt.codigo {
				t.Fatalf("esperado %d, recebido %v", tt.codigo, se)
			}
		})
	}
	if verificaErro(nil) != nil {
		t.Fatal("nil deve continuar nil")
	}
}

func TestClicaAbaFalhaRapido(t *testing.T) {
	ctx := navegadorTeste(t, time.Minute)
	paginaTeste(t, ctx, `<p id="existe">painel</p>`)
	c := crawler{stepTimeout: 2 * time.Second}
	inicio := time.Now()
	err := c.clicaAba(ctx, "#nao-existe")
	var se *status.Error
	if !errors.As(err, &se) || se.Code != status.DeadlineExceeded {
		t.Fatalf("esperado código %d, recebido %v", status.DeadlineExceeded, err)
	}
	if d := time.Since(inicio); d > 10*time.Second {
		t.Fatalf("levou %s", d)
	}
	// O prazo da etapa não pode encerrar a aba usada pelas etapas seguintes.
	if err := c.clicaAba(ctx, "#existe"); err != nil {
		t.Fatalf("aba encerrada após o prazo da etapa: %v", err)
	}
}

func TestPollingRespeitaTimeout(t *testing.T) {
	ctx := navegadorTeste(t, time.Minute)
	paginaTeste(t, ctx, `<p>filtro que nunca confirma</p>`)
	c := crawler{pollTimeout: time.Second}
	inicio := time.Now()
	err := c.aguarda(ctx, "filtro P63_CE_TRIBUNAL", `(campo) => false`, nil, "P63_CE_TRIBUNAL")
	if err == nil {
		t.Fatal("espera sem confirmação terminou sem erro")
	}
	if d := time.Since(inicio); d > 5*time.Second {
		t.Fatalf("levou %s", d)
	}
	var pronto bool
	if err := c.aguarda(ctx, "página carregada", `() => document.readyState === 'complete'`, &pronto); err != nil || !pronto {
		t.Fatalf("espera que confirma falhou: %v, %v", pronto, err)
	}
}

func TestPrazoDaAberturaDoPainel(t *testing.T) {
	// A abertura levou 57,4 s numa coleta válida (TJAP 05/2025, 13/11/2026); as outras etapas, até 17 s.
	if p := (crawler{}).prazoAbertura(); p != 2*defaultStepTimeout {
		t.Fatalf("padrão = %s", p)
	}
	if p := (crawler{stepTimeout: 45 * time.Second}).prazoAbertura(); p != 90*time.Second {
		t.Fatalf("com STEP_TIMEOUT=45s = %s", p)
	}
}
