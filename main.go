package main

import (
	"log"
	"net/http"

	mongoclient "github.com/4jairo/webrtcFiletransfer/db"
	"github.com/4jairo/webrtcFiletransfer/handler"
	"github.com/4jairo/webrtcFiletransfer/routes"
	routesWs "github.com/4jairo/webrtcFiletransfer/routes/ws"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	mongoclient.Connect()
	//mongoclient.Mongo.CreateTTLIndex(schema.FilesCollection)

	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()

	// files
	api.Handle("/files/remove", handler.HandleBody(routes.RemoveFilesHandler)).Methods("POST")
	api.Handle("/files/add", handler.HandleBody(routes.AddFileHandler)).Methods("POST")
	api.Handle("/files/new", handler.HandleBody(routes.NewFileHandler)).Methods("POST")
	api.HandleFunc("/files/{objId}", routes.GetFilesHandler).Methods("GET")

	// signaling
	api.Handle("/signaling/new", handler.HandleBody(routes.NewSignalingHandler)).Methods("POST")

	// ws
	api.HandleFunc("/ws/conn/{objId}", routesWs.WsHandler(routesWs.WsRoleConn))
	api.HandleFunc("/ws/host/{objId}", routesWs.WsHandler(routesWs.WsRoleHost))

	// ping
	api.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"*"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	log.Println("Listening in 0.0.0.0:8900")
	if err := http.ListenAndServe("0.0.0.0:8900", handler); err != nil {
		log.Fatalf("Error setting up listener: %v\n", err)
	}
}
