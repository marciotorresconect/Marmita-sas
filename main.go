package main

import (
	"database/sql"
	"log"
	"net/http"
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

	// Criar tabelas atualizadas
	db.Exec(`CREATE TABLE IF NOT EXISTS pedidos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cliente TEXT,
		whatsapp TEXT,
		tamanho TEXT,
		refrigerante TEXT,
		pagamento TEXT,
		status TEXT,
		valor REAL,
		pago BOOLEAN
	)`)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	// CLIENTE: Tela de Pedido
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// CLIENTE: Processar Pedido
	r.POST("/pedido", func(c *gin.Context) {
		cliente := c.PostForm("cliente")
		whatsapp := c.PostForm("whatsapp")
		tamanho := c.PostForm("tamanho")
		refri := c.PostForm("refrigerante")
		pagamento := c.PostForm("pagamento")

		valor := 20.0
		if tamanho == "Grande" { valor = 25.0 }
		if refri == "Sim" { valor += 5.0 }

		pago := true
		if pagamento == "Fiado" { pago = false }

		_, err := db.Exec("INSERT INTO pedidos (cliente, whatsapp, tamanho, refrigerante, pagamento, status, valor, pago) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			cliente, whatsapp, tamanho, refri, pagamento, "Preparando", valor, pago)

		if err != nil {
			c.String(500, "Erro ao salvar")
			return
		}
		// Redireciona para o status do próprio cliente
		c.Redirect(http.StatusFound, "/status/"+whatsapp)
	})

	// CLIENTE: Ver status do seu pedido
	r.GET("/status/:whatsapp", func(c *gin.Context) {
		whatsapp := c.Param("whatsapp")
		var p struct { Cliente, Tamanho, Status string }
		err := db.QueryRow("SELECT cliente, tamanho, status FROM pedidos WHERE whatsapp = ? ORDER BY id DESC LIMIT 1", whatsapp).Scan(&p.Cliente, &p.Tamanho, &p.Status)
		
		if err != nil {
			c.String(404, "Nenhum pedido encontrado para este número.")
			return
		}
		c.HTML(http.StatusOK, "status.html", p)
	})

	// ADMIN: Painel Principal
	r.GET("/admin", func(c *gin.Context) {
		rows, _ := db.Query("SELECT id, cliente, tamanho, status, pagamento, valor, pago FROM pedidos ORDER BY id DESC")
		var pedidos []map[string]interface{}
		for rows.Next() {
			var id int
			var cli, tam, st, pag string
			var val float64
			var pg bool
			rows.Scan(&id, &cli, &tam, &st, &pag, &val, &pg)
			pedidos = append(pedidos, map[string]interface{}{
				"ID": id, "Cliente": cli, "Tamanho": tam, "Status": st, "Pagamento": pag, "Valor": val, "Pago": pg,
			})
		}
		c.HTML(http.StatusOK, "admin.html", gin.H{"pedidos": pedidos})
	})

	// ADMIN: Atualizar Status
	r.POST("/update/:id", func(c *gin.Context) {
		db.Exec("UPDATE pedidos SET status = ? WHERE id = ?", c.PostForm("status"), c.Param("id"))
		c.Redirect(http.StatusFound, "/admin")
	})

	// ADMIN: Marcar como Pago (Fiado -> Pago)
	r.POST("/pagar/:id", func(c *gin.Context) {
		db.Exec("UPDATE pedidos SET pago = true WHERE id = ?", c.Param("id"))
		c.Redirect(http.StatusFound, "/admin")
	})

	// ADMIN: Excluir Pedido
	r.POST("/delete/:id", func(c *gin.Context) {
		db.Exec("DELETE FROM pedidos WHERE id = ?", c.Param("id"))
		c.Redirect(http.StatusFound, "/admin")
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	r.Run(":" + port)
}

