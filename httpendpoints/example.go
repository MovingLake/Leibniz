package httpendpoints

import (
	"context"
	"database/sql"
	"log"
	"net/http"
)

type ExampleEndpoint struct {
}

func (e *ExampleEndpoint) Serve(ctx context.Context, db *sql.DB, rw http.ResponseWriter, rq *http.Request) {
	// Add your handler logic here
	log.Println("Example endpoint")
	rw.Write([]byte("Example endpoint"))
}
