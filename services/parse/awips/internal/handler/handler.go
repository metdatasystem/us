package handler

import (
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/pkg/awips"
	dbAwips "github.com/metdatasystem/us/pkg/db/pkg/awips"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

var (
	vtecRoute = regexp.MustCompile("(MWW|FWW|CFW|TCV|RFW|FFA|SVR|TOR|SVS|SMW|MWS|NPW|WCN|WSW|EWW|FLS|FLW)")
	mcdRoute  = regexp.MustCompile("(SWOMCD)")
)

type Route struct {
	Name    string
	Match   func(product *awips.Product) bool
	Handler func(handler *Handler) HandlerFunc
}

var routes = []Route{
	// VTEC Products
	{
		Name: "VTEC Handler",
		Match: func(product *awips.Product) bool {
			return vtecRoute.MatchString(product.AWIPS.Product)
		},
		Handler: func(handler *Handler) HandlerFunc { return NewVTECHandler(handler) },
	},
}

type Handler struct {
	db        *pgxpool.Pool
	kafka     *kgo.Client
	dbProduct *dbAwips.Product
	product   *awips.Product
	log       zerolog.Logger
}

type HandlerFunc interface {
	Handle() error
}

func Handle(text string, receivedAt time.Time, db *pgxpool.Pool, kafka *kgo.Client) {

	log := zlog.With().Logger()

	product, err := awips.New(text)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse product")
		return
	}
	if product.WMO.Original != "" {
		log = log.With().Str("wmo", product.WMO.Original).Logger()
	}
	if product.AWIPS.Original != "" {
		log = log.With().Str("awips", product.AWIPS.Original).Logger()
	}

	handler := &Handler{
		db:      db,
		kafka:   kafka,
		log:     log,
		product: product,
	}

	// Process the product matching it to any routes
	for _, route := range routes {
		if route.Match(product) {
			if handler.dbProduct == nil {
				pHandler := productHandler{*handler}
				dbproduct, err := pHandler.Handle(*product, receivedAt)
				if err != nil {
					log.Error().Err(err).Msg("failed to handle product")
					continue
				}
				handler.dbProduct = dbproduct
			}
			h := route.Handler(handler)
			err := h.Handle()
			if err != nil {
				log.Error().Err(err).Msgf("failed to handle product with %s", route.Name)
			}
		}
	}
}

func (handler *Handler) ProduceKafkaMessage() {

}
