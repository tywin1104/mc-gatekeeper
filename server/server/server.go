package server

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/urfave/negroni"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/tywin1104/mc-whitelist/broker"
	"github.com/tywin1104/mc-whitelist/config"
	"github.com/tywin1104/mc-whitelist/db"
)

// Service represents struct that deals with database level operations
type Service struct {
	dbService *db.Service
	router    *mux.Router
	broker    *broker.Service
	c         *config.Config
	logger    *logrus.Entry
}

// NewService create new mongoDb service that handles database level operations
func NewService(db *db.Service, broker *broker.Service, c *config.Config, logger *logrus.Entry) *Service {
	return &Service{
		dbService: db,
		router:    mux.NewRouter().StrictSlash(true),
		broker:    broker,
		c:         c,
		logger:    logger,
	}
}

// Listen opens up the http port for REST API and register all routes
func (svc *Service) Listen(port string, wg *sync.WaitGroup) {
	svc.routes()

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
		// Skip logging health checks as they are constantly called by kubernetes probes
		if r.URL.String() != "/health" {
			svc.logger.Debugf("%s %s (code=%d dt=%s)",
				r.Method,
				r.URL,
				m.Code,
				m.Duration,
			)
		}
	})
	endpoint := "health"
	go waitForHTTPServer(wg, port[1:], endpoint, svc.logger)
	go func() {
		if err := http.ListenAndServe(port, wrappedH); err != nil {
			svc.logger.Fatal(err)
		}
	}()
}

func waitForHTTPServer(wg *sync.WaitGroup, port, endpoint string, log *logrus.Entry) {
	var client http.Client
	for {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%s/%s", port, endpoint))
		if err == nil && resp != nil && resp.StatusCode == 200 {
			wg.Done()
			log.WithField("port", port).Infof("Http server is up and healthy")
			break
		}
	}
}

func (svc *Service) routes() {
	// Endpoints that are public accessible
	external := svc.router.PathPrefix("/api/v1/requests").Subrouter()
	external.HandleFunc("/", svc.HandleCreateRequest()).Methods("POST")
	external.HandleFunc("/{requestIdEncoded}", svc.HandleGetRequestByID()).Methods("GET")
	external.HandleFunc("/{requestIdEncoded}", svc.HandlePatchRequestByID()).Methods("PATCH").Queries("adm", "{adm}")

	// Endpoint to authenticate admin user
	auth := svc.router.PathPrefix("/api/v1/auth").Subrouter()
	auth.HandleFunc("/", svc.HandleAdminSignin()).Methods("POST")

	// Endpoints for internal(admin) consumptiono only that are wrapped by auth middleware
	internal := svc.router.PathPrefix("/api/v1/internal/requests").Subrouter()
	internal.Handle("/", negroni.New(
		negroni.HandlerFunc(svc.GetAuthMiddleware().HandlerWithNext),
		negroni.Wrap(svc.HandleGetRequests()),
	)).Methods("GET")
	internal.Handle("/{requestId}", negroni.New(
		negroni.HandlerFunc(svc.GetAuthMiddleware().HandlerWithNext),
		negroni.Wrap(svc.HandleInternalPatchRequestByID()),
	)).Methods("PATCH")

	// Server health endpoint
	svc.router.HandleFunc("/health", svc.HandleHealthCheck()).Methods("GET")
	// Recaptcha verification endpoint
	svc.router.HandleFunc("/api/v1/recaptcha/verify", svc.handleVerifyRecaptcha()).Methods("POST")
	// Endpoint to verify validity of action email
	svc.router.HandleFunc("/api/v1/verify/{requestIdEncoded}", svc.HandleVerifyMatchingTokens()).Methods("GET").Queries("adm", "{adm}")
}

//HandleHealthCheck signals the server is running
func (svc *Service) HandleHealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
