package server

import (
	"net/http"

	"github.com/tywin1104/mc-whitelist/utils"
	"github.com/urfave/negroni"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/tywin1104/mc-whitelist/broker"
	"github.com/tywin1104/mc-whitelist/config"
	"github.com/tywin1104/mc-whitelist/db"
	admin_auth "github.com/tywin1104/mc-whitelist/server/auth"
)

// Service represents struct that deals with database level operations
type Service struct {
	dbService *db.Service
	router    *mux.Router
	broker    *broker.Service
	c         *config.Config
	logger    *logrus.Logger
}

// NewService create new mongoDb service that handles database level operations
func NewService(db *db.Service, broker *broker.Service, c *config.Config, logger *logrus.Logger) *Service {
	return &Service{
		dbService: db,
		router:    mux.NewRouter().StrictSlash(true),
		broker:    broker,
		c:         c,
		logger:    logger,
	}
}

// Listen opens up the http port for REST API and register all routes
func (svc *Service) Listen(port string) {
	log := svc.logger
	svc.routes()
	log.WithFields(logrus.Fields{
		"port": port,
	}).Info("The API http server starts listening")

	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PATCH"},
		AllowedHeaders: []string{"*"},
	})

	// Listen and serve
	handler := c.Handler(svc.router)

	// capture http related metrics
	wrappedH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(handler, w, r)
		svc.logger.Infof("%s %s (code=%d dt=%s)",
			r.Method,
			r.URL,
			m.Code,
			m.Duration,
		)
	})
	log.Fatal(http.ListenAndServe(port, wrappedH))
}

func (svc *Service) routes() {
	// Endpoints that are public accessible
	external := svc.router.PathPrefix("/api/v1/requests").Subrouter()
	external.HandleFunc("/", svc.handleCreateRequest()).Methods("POST")
	external.HandleFunc("/{requestIdEncoded}", svc.handleGetRequestByID()).Methods("GET")
	external.HandleFunc("/{requestIdEncoded}", svc.handlePatchRequestByID()).Methods("PATCH").Queries("adm", "{adm}")

	// Endpoint to verify validity of op token for frontend to consume
	r := svc.router.PathPrefix("/api/v1/verify").Subrouter()
	r.HandleFunc("/", svc.handleVerifyAdminToken()).Methods("GET").Queries("adm", "{adm}")

	// Endpoint to authenticate admin user
	auth := svc.router.PathPrefix("/api/v1/auth").Subrouter()
	auth.HandleFunc("/", admin_auth.AdminSigninHandler(svc.c)).Methods("POST")

	// Endpoints for internal(admin) consumptiono only
	internal := svc.router.PathPrefix("/api/v1/internal/requests").Subrouter()
	internal.Handle("/", negroni.New(
		negroni.HandlerFunc(admin_auth.AuthMiddleware.HandlerWithNext),
		negroni.Wrap(svc.handleGetRequests()),
	)).Methods("GET")
	internal.Handle("/{requestId}", negroni.New(
		negroni.HandlerFunc(admin_auth.AuthMiddleware.HandlerWithNext),
		negroni.Wrap(svc.handleInternalPatchRequestByID()),
	)).Methods("PATCH")
	internal.Handle("/{requestId}", negroni.New(
		negroni.HandlerFunc(admin_auth.AuthMiddleware.HandlerWithNext),
		negroni.Wrap(svc.handleDeleteRequestByID()),
	)).Methods("DELETE")
}

func (svc *Service) handleVerifyAdminToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get admin info from ?adm=<EncodedAdminEmail>
		keys, ok := r.URL.Query()["adm"]

		if !ok || len(keys[0]) < 1 {
			http.Error(w, "adm token is missing", http.StatusBadRequest)
			return
		}
		admToken := keys[0]
		admEmail, err := utils.DecodeAndDecrypt(admToken, svc.c.PassPhrase)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		valid := false
		for _, op := range svc.c.Ops {
			if admEmail == op {
				valid = true
			}
		}
		if !valid {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
