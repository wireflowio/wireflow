package internal

import (
	"encoding/json"
	drpgrpc "linkany/drp/grpc"
)

type Offer interface {
	Marshal() (int, []byte, error)
	OfferType() OfferType
	TieBreaker() uint64
	GetNode() *NodeMessage
}

type OfferHandler interface {
	SendOffer(drpgrpc.MessageType, string, string, Offer) error
	ReceiveOffer(message *drpgrpc.DrpMessage) error
}

type OfferType int

const (
	OfferTypeDrpOffer OfferType = iota
	OfferTypeDrpOfferAnswer
	OfferTypeDirectOffer
	OfferTypeDirectOfferAnswer
	OfferTypeRelayOffer
	OfferTypeRelayAnswer
)

type ConnectionType int

const (
	DirectType ConnectionType = iota
	RelayType
	DrpType
)

func UnmarshalOffer[T Offer](data []byte, t T) (T, error) {
	err := json.Unmarshal(data, t)
	if err != nil {
		var zero T
		return zero, err
	}
	return t, nil
}
