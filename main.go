package main

import (
	"log"
	"mnmlsm/web"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	// Static files
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Routes
	mux.HandleFunc("/", web.HandleHome)
	mux.HandleFunc("/options", web.HandleOptions)
	mux.HandleFunc("/stocks", web.HandleStocks)
	mux.HandleFunc("/stocks/", web.HandleStockPages)
	mux.HandleFunc("/analytics", web.HandleAnalytics)
	mux.HandleFunc("/risk", web.HandleRisk)
	mux.HandleFunc("/rules", web.HandleRules)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
