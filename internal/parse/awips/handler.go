package internal

import (
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/pkg/awips"
	"github.com/metdatasystem/us/shared/models"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

var (
	vtecRoute = regexp.MustCompile("(MWW|FFW|CFW|TCV|RFW|FFA|SVR|TOR|SVS|SMW|MWS|NPW|WCN|WSW|EWW|FLS|FLW)")
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
	rabbit    *amqp.Channel
	dbProduct *models.AWIPSProduct
	product   *awips.Product
	log       zerolog.Logger
}

type HandlerFunc interface {
	Handle() error
}

func HandleText(text string, receivedAt time.Time, db *pgxpool.Pool, rabbit *amqp.Channel) {

	log := zlog.With().Logger()

	product, err := awips.New(text)
	if err != nil && err != awips.ErrCouldNotFindAWIPS {
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
		rabbit:  rabbit,
		log:     log,
		product: product,
	}

	handler.process(receivedAt)
}

func Handle(text string, receivedAt time.Time, wmo string, office string, awipsID string, db *pgxpool.Pool, rabbit *amqp.Channel) error {
	log := zlog.With().Logger()

	log = log.With().Str("awips", awipsID).Logger()
	log = log.With().Str("wmo", wmo).Logger()

	product, err := awips.New(text)
	if err != nil {
		if err != awips.ErrCouldNotFindAWIPS {
			log.Error().Err(err).Msg("failed to parse product")
			return fmt.Errorf("failed to parse product: %w", err)
		}
		if awipsID != "" {
			product.AWIPS, err = awips.ParseAWIPS(awipsID + "\n")
			if err != nil {
				log.Error().Err(err).Msg("failed to parse awips")
				return fmt.Errorf("failed to parse awips: %w", err)
			}
		}
	}
	if product.WMO.Original != "" {
		log = log.With().Str("wmo", product.WMO.Original).Logger()
	}

	handler := &Handler{
		db:      db,
		rabbit:  rabbit,
		log:     log,
		product: product,
	}

	handler.process(receivedAt)

	return nil
}

// Process the product matching it to any routes
func (handler *Handler) process(receivedAt time.Time) {
	product := handler.product

	for _, route := range routes {
		if route.Match(product) {
			if handler.dbProduct == nil {
				pHandler := productHandler{*handler}
				dbproduct, err := pHandler.Handle(*product, receivedAt)
				if err != nil {
					handler.log.Error().Err(err).Msg("failed to handle product")
					continue
				}
				handler.dbProduct = dbproduct
			}
			h := route.Handler(handler)
			err := h.Handle()
			if err != nil {
				handler.log.Error().Err(err).Msgf("failed to handle product with %s", route.Name)
			}
		}
	}
}
