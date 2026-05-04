package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal(err)
	}

	// Criar tabela se não existir
	db.Exec(`CREATE TABLE IF NOT EXISTS pedidos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cliente TEXT,
		whatsapp TEXT,
		tamanho TEXT,
		refrigerante TEXT,
		pagamento TEXT,
		status TEXT,
		valor REAL,
		pago BOOLEAN,
		data_pedido DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	r := gin.Default()

	// ROTA: Tela inicial
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
		<!DOCTYPE html>
		<html lang="pt-pt">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Fazer Pedido - Cantinho do Sabor</title>
			<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
			<style>
				body { background: #f4f7f6; font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; }
				.card { background: white; padding: 40px; border-radius: 20px; box-shadow: 0 10px 30px rgba(0,0,0,0.1); max-width: 450px; width: 100%; }
			</style>
		</head>
		<body>
			<div class="card">
				<h2 class="mb-4 text-center text-success">🍱 Fazer Pedido</h2>
				<form action="/pedido" method="POST">
					<div class="mb-3">
						<label class="form-label">Nome do Cliente:</label>
						<input type="text" class="form-control" name="cliente" required placeholder="Ex: João Silva">
					</div>
					<div class="mb-3">
						<label class="form-label">WhatsApp (com DDD, ex: 13996554848):</label>
						<input type="text" class="form-control" name="whatsapp" required placeholder="13996554848">
					</div>
					<div class="mb-3">
						<label class="form-label">Tamanho da Marmita:</label>
						<select class="form-select" name="tamanho">
							<option value="Média">Média (R$ 20,00)</option>
							<option value="Grande">Grande (R$ 25,00)</option>
						</select>
					</div>
					<div class="mb-3">
						<label class="form-label">Deseja Refrigerante?</label>
						<select class="form-select" name="refrigerante">
							<option value="Não">Não</option>
							<option value="Sim">Sim (+ R$ 5,00)</option>
						</select>
					</div>
					<div class="mb-3">
						<label class="form-label">Método de Pagamento:</label>
						<select class="form-select" name="pagamento">
							<option value="Pix">Pix / Dinheiro</option>
							<option value="Fiado">Fiado</option>
						</select>
					</div>
					<button type="submit" class="btn btn-success w-100 mt-2">Enviar Pedido</button>
				</form>
			</div>
		</body>
		</html>
		`)
	})

	// ROTA: Processa o pedido e abre o WhatsApp com a mensagem e o link de status
	r.POST("/pedido", func(c *gin.Context) {
		cliente := c.PostForm("cliente")
		whatsapp := c.PostForm("whatsapp")
		tamanho := c.PostForm("tamanho")
		refri := c.PostForm("refrigerante")
		pagamento := c.PostForm("pagamento")

		valor := 20.0
		if tamanho == "Grande" {
			valor = 25.0
		}
		if refri == "Sim" {
			valor += 5.0
		}

		pago := true
		if pagamento == "Fiado" {
			pago = false
		}

		_, err := db.Exec("INSERT INTO pedidos (cliente, whatsapp, tamanho, refrigerante, pagamento, status, valor, pago) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			cliente, whatsapp, tamanho, refri, pagamento, "Preparando", valor, pago)

		if err != nil {
			c.String(500, "Erro ao salvar pedido")
			return
		}

		// Substitua com o seu número de WhatsApp com código do país (55)
		seuNumero := "5535998022156"
		texto := fmt.Sprintf("🍱 *NOVO PEDIDO!*\n\n*Cliente:* %s\n*Marmita:* %s\n*Pagamento:* %s\n*Total:* R$ %.2f\n\n*Acompanhe o pedido aqui:* https://marmita-saas.onrender.com/status/%s",
			cliente, tamanho, pagamento, valor, whatsapp)

		linkWhats := "https://api.whatsapp.com/send?phone=" + seuNumero + "&text=" + url.QueryEscape(texto)
		c.Redirect(http.StatusFound, linkWhats)
	})

	// ROTA: Tela de acompanhamento do cliente
	r.GET("/status/:whatsapp", func(c *gin.Context) {
		whatsapp := c.Param("whatsapp")

		var p struct {
			Cliente string
			Tamanho string
			Status  string
		}

		err := db.QueryRow("SELECT cliente, tamanho, status FROM pedidos WHERE whatsapp = ? ORDER BY id DESC LIMIT 1", whatsapp).Scan(&p.Cliente, &p.Tamanho, &p.Status)

		if err != nil {
			err = db.QueryRow("SELECT cliente, tamanho, status FROM pedidos ORDER BY id DESC LIMIT 1").Scan(&p.Cliente, &p.Tamanho, &p.Status)

			if err != nil {
				c.Header("Content-Type", "text/html")
				c.String(404, `
				<!DOCTYPE html>
				<html>
				<head>
					<meta charset="UTF-8">
					<title>Pedido não encontrado</title>
					<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
				</head>
				<body class="bg-light d-flex align-items-center justify-content-center" style="height: 100vh;">
					<div class="card p-4 text-center shadow-sm" style="max-width: 400px; width: 100%;">
						<h3 class="text-danger">⚠️ Pedido não localizado</h3>
						<p class="text-muted">Nenhum pedido recente foi encontrado para este número.</p>
						<a href="/" class="btn btn-primary mt-3">Voltar ao Início</a>
					</div>
				</body>
				</html>
				`)
				return
			}
		}

		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="pt-pt">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Acompanhamento do Pedido - Marmita</title>
			<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
			<style>
				body { background: #f4f7f6; font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; }
				.card { background: white; padding: 40px; border-radius: 20px; box-shadow: 0 10px 30px rgba(0,0,0,0.1); max-width: 400px; width: 100%; text-align: center; }
				.status-badge { font-size: 1.3rem; padding: 10px 20px; border-radius: 20px; }
			</style>
		</head>
		<body>
			<div class="card">
				<h2 class="text-success mb-4">🍱 Pedido em Andamento</h2>
				<hr class="my-4">
				<div class="mb-3">
					<strong>Cliente:</strong><br>
					<span class="fs-4 text-primary">%s</span>
				</div>
				<div class="mb-4">
					<strong>Marmita:</strong><br>
					<span class="fs-5 text-muted">%s</span>
				</div>
				<div class="mb-4">
					<strong>Status Atual:</strong><br>
					<span class="badge bg-primary status-badge">%s</span>
				</div>
				<a href="/" class="btn btn-secondary mt-3 w-100">Fazer Novo Pedido</a>
			</div>
		</body>
		</html>
		`, p.Cliente, p.Tamanho, p.Status))
	})

	// ROTA: Painel Administrativo
	r.GET("/admin", func(c *gin.Context) {
		rows, _ := db.Query("SELECT id, cliente, tamanho, status, pagamento, valor, pago FROM pedidos ORDER BY id DESC")
		var rowsData string
		var totalGeral float64

		for rows.Next() {
			var id int
			var cli, tam, st, pag string
			var val float64
			var pg bool
			rows.Scan(&id, &cli, &tam, &st, &pag, &val, &pg)
			totalGeral += val

			pagoStr := "Não"
			if pg {
				pagoStr = "Sim"
			}

			rowsData += fmt.Sprintf(`
				<tr>
					<td>%d</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>R$ %.2f</td>
					<td>%s</td>
					<td>
						<form action="/update/%d" method="POST" class="d-inline">
							<button type="submit" class="btn btn-sm btn-warning">Mudar Status</button>
						</form>
					</td>
				</tr>
			`, id, cli, tam, st, pag, val, pagoStr, id)
		}

		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="pt-pt">
		<head>
			<meta charset="UTF-8">
			<title>Painel Administrativo</title>
			<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
		</head>
		<body class="bg-light p-4">
			<h2>Painel Administrativo</h2>
			<a href="/" class="btn btn-secondary mb-3">Voltar ao Início</a>
			<table class="table table-bordered table-striped bg-white">
				<thead>
					<tr>
						<th>ID</th>
						<th>Cliente</th>
						<th>Tamanho</th>
						<th>Status</th>
						<th>Pagamento</th>
						<th>Valor</th>
						<th>Pago?</th>
						<th>Ação</th>
					</tr>
				</thead>
				<tbody>
					%s
					<tr>
						<td colspan="8"><strong>Total Geral: R$ %.2f</strong></td>
					</tr>
				</tbody>
			</table>
		</body>
		</html>
		`, rowsData, totalGeral))
	})

	// ROTA: Atualiza status do pedido dinamicamente
	r.POST("/update/:id", func(c *gin.Context) {
		id := c.Param("id")

		var currentStatus string
		err := db.QueryRow("SELECT status FROM pedidos WHERE id = ?", id).Scan(&currentStatus)
		if err == nil {
			newStatus := "Entregue"
			if currentStatus == "Entregue" {
				newStatus = "Preparando"
			}
			db.Exec("UPDATE pedidos SET status = ? WHERE id = ?", newStatus, id)
		}
		c.Redirect(http.StatusFound, "/admin")
	})

	// ROTA: Fallback para URL's inválidas
	r.NoRoute(func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(404, `
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Página não encontrada</title>
			<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
		</head>
		<body class="bg-light d-flex align-items-center justify-content-center" style="height: 100vh;">
			<div class="card p-4 text-center shadow-sm" style="max-width: 400px; width: 100%;">
				<h3 class="text-danger">Página não encontrada</h3>
				<p>O endereço que está a tentar aceder não existe.</p>
				<a href="/" class="btn btn-primary mt-3">Voltar ao Início</a>
			</div>
		</body>
		</html>
		`)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
