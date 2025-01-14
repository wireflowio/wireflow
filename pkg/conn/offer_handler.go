package conn

import (
	"linkany/pkg/internal"
)

type OfferHandler interface {
	Handle(offer internal.Offer) error
}
