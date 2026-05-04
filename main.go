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
	r.LoadHTMLGlob("templates/*")

	// ROTA: Tela inicial
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
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

		// Mensagem que vai para o seu WhatsApp (inclui o link de acompanhamento do cliente)
		seuNumero := "5535998022156"
		texto := fmt.Sprintf("🍱 *NOVO PEDIDO!*\n\n*Cliente:* %s\n*Marmita:* %s\n*Pagamento:* %s\n*Total:* R$ %.2f\n\n*Acompanhe o pedido aqui:* https://marmita-saas.onrender.com/status/%s",
			cliente, tamanho, pagamento, valor, whatsapp)

		linkWhats := "https://api.whatsapp.com/send?phone=" + seuNumero + "&text=" + url.QueryEscape(texto)

		// Redireciona o navegador do cliente para o WhatsApp automaticamente
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
			// Retorna página amigável ao invés de tela preta
			c.Header("Content-Type", "text/html")
			c.String(404, `<!DOCTYPE html>
<html lang="pt-br">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Pedido Não Encontrado</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { background-color: #f4f7f6; font-family: 'Poppins', sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; }
        .card { border-radius: 25px; box-shadow: 0 10px 30px rgba(0,0,0,0.1); border: none; padding: 40px; text-align: center; max-width: 400px; width: 100%; background: #ffffff; }
    </style>
</head>
<body>
    <div class="card">
        <h3 class="text-danger mb-3">⚠️ Pedido Não Encontrado</h3>
        <p class="text-muted">Nenhum pedido foi localizado para este número de WhatsApp. Por favor, certifique-se de que o pedido foi enviado ou faça um novo pedido.</p>
        <a href="/" class="btn btn-primary mt-3 w-100">Fazer Novo Pedido</a>
    </div>
</body>
</html>`)
			return
		}
		c.HTML(http.StatusOK, "status.html", p)
	})

	// ROTA: Painel Administrativo com soma dos pedidos
	r.GET("/admin", func(c *gin.Context) {
		rows, _ := db.Query("SELECT id, cliente, tamanho, status, pagamento, valor, pago FROM pedidos ORDER BY id DESC")
		var pedidos []map[string]interface{}
		var totalGeral float64

		for rows.Next() {
			var id int
			var cli, tam, st, pag string
			var val float64
			var pg bool
			rows.Scan(&id, &cli, &tam, &st, &pag, &val, &pg)
			
			totalGeral += val

			pedidos = append(pedidos, map[string]interface{}{
				"ID": id, "Cliente": cli, "Tamanho": tam, "Status": st, "Pagamento": pag, "Valor": val, "Pago": pg,
			})
		}
		c.HTML(http.StatusOK, "admin.html", gin.H{
			"pedidos":    pedidos,
			"TotalGeral": totalGeral,
		})
	})

	// --- FUNÇÕES DO ADMIN ---

	r.POST("/update/:id", func(c *gin.Context) {
		db.Exec("UPDATE pedidos SET status = ? WHERE id = ?", c.PostForm("status"), c.Param("id"))
		c.Redirect(http.StatusFound, "/admin")
	})

	r.POST("/pagar/:id", func(c *gin.Context) {
		db.Exec("UPDATE pedidos SET pago = true WHERE id = ?", c.Param("id"))
		c.Redirect(http.StatusFound, "/admin")
	})

	r.POST("/delete/:id", func(c *gin.Context) {
		db.Exec("DELETE FROM pedidos WHERE id = ?", c.Param("id"))
		c.Redirect(http.StatusFound, "/admin")
	})

	// ROTA: Redirecionamento para evitar o erro 404 caso o cliente acesse apenas o número
	r.GET("/:whatsapp", func(c *gin.Context) {
		whatsapp := c.Param("whatsapp")
		isNumber := true
		for _, ch := range whatsapp {
			if ch < '0' || ch > '9' {
				isNumber = false
				break
			}
		}
		
		if isNumber && len(whatsapp) >= 10 {
			c.Redirect(http.StatusFound, "/status/"+whatsapp)
			return
		}
		
		c.Header("Content-Type", "text/html")
		c.String(404, `<!DOCTYPE html>
<html lang="pt-br">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Página Não Encontrada</title>
    <style>
        body { background: #f4f7f6; font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; }
        .card { background: white; padding: 40px; border-radius: 20px; box-shadow: 0 10px 30px rgba(0,0,0,0.1); text-align: center; max-width: 400px; width: 100%; }
        a { text-decoration: none; display: inline-block; padding: 10px 20px; background: #0d6efd; color: white; border-radius: 5px; margin-top: 15px; }
    </style>
</head>
<body>
    <div class="card">
        <h2>Página Não Encontrada</h2>
        <p>O endereço que você está tentando acessar não existe ou o número é inválido.</p>
        <a href="/">Voltar para o Início</a>
    </div>
</body>
</html>`)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
