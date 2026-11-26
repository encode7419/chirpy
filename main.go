package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) apiCounter(w http.ResponseWriter, req *http.Request) {
	msg := "<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>"
	fmt.Fprintf(w, msg, cfg.fileserverHits.Load())
}

func (cfg *apiConfig) resetCounter(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
}
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnErr struct {
		Error string `json:"error"`
	}
	respBody := returnErr{
		Error: msg,
	}
	respondWithJSON(w, code, respBody)
}

func chirpProfanityFilter(msg string) string {
	profanities := map[string]string{
		"kerfuffle": "****",
		"sharbert":  "****",
		"fornax":    "****",
	}
	splitMsgs := strings.Split(msg, " ")

	for x, m := range splitMsgs {
		if _, ok := profanities[strings.ToLower(m)]; ok {
			splitMsgs[x] = "****"
		}
	}
	joinedMsgs := strings.Join(splitMsgs, " ")
	return joinedMsgs

}

func validateChirpHandler(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	type validResp struct {
		CleanedBody string `json:"cleaned_body"`
	}
	cleanedStr := chirpProfanityFilter(params.Body)
	validJson := validResp{
		CleanedBody: cleanedStr,
	}

	respondWithJSON(w, 200, validJson)
}

func main() {

	apiCfg := apiConfig{}
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(appHandler))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.apiCounter)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetCounter)
	mux.HandleFunc("/api/validate_chirp", validateChirpHandler)
	server := &http.Server{Addr: ":8080", Handler: mux}
	server.ListenAndServe()

}
