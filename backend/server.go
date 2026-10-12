
package backend

import (
	"fmt"
	"net/http"
    "time"
)

func Start(port string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "healthy")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend on port %s\n", port)
	})


mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
    time.Sleep(5 * time.Second)
    fmt.Fprintln(w, "Slow response from backend on port", port)
})


	address := ":" + port
	fmt.Println("Backend starting on", address)

	if err := http.ListenAndServe(address, mux); err != nil {
		fmt.Println("Server error:", err)
	}
}
