package mapper

import (
	"linkany/management/dto"
	"linkany/management/entity"
)

// UserInterface is an interface for user mapper
type UserInterface interface {
	Login(u *dto.UserDto) (*entity.Token, error)
	Register(e *dto.UserDto) (*entity.User, error)

	//Get returns a user by token
	Get(token string) (*entity.User, error)
}

type QueryParams struct {
	PubKey   *string
	UserId   *string
	Online   *int
	Total    *int
	PageNo   *int
	PageSize *int

	filters []*kv
}

type kv struct {
	Key   string
	Value interface{}
}

func (qp *QueryParams) Generate() []*kv {
	var result []*kv
	if qp.UserId != nil {
		v := &kv{
			Key:   "user_id",
			Value: qp.UserId,
		}

		result = append(result, v)
	}

	if qp.Online != nil {
		v := &kv{
			Key:   "on_line",
			Value: qp.Online,
		}

		result = append(result, v)
	}

	return result
}

// PeerInterface is an interface for peer mapper
type PeerInterface interface {
	Register(e *dto.PeerDto) (*entity.Peer, error)
	Update(e *dto.PeerDto) (*entity.Peer, error)
	Delete(e *dto.PeerDto) error

	// GetByAppId returns a peer by appId, every client has its own appId
	GetByAppId(appId string) (*entity.Peer, error)

	// List returns a list of peers by userId，when client start up, it will call this method to get all the peers once
	// after that, it will call Watch method to get the latest peers
	List(params *QueryParams) ([]*entity.Peer, error)

	// Watch returns a channel that will be used to send the latest peers to the client
	//Watch() (<-chan *entity.Peer, error)
}

// PlanInterface is an interface for plan mapper
type PlanInterface interface {
	// List returns a list of plans
	List() ([]*entity.Plan, error)
	Get() (*entity.Plan, error)
	Page() (*entity.Plan, error)
}

type SupportInterface interface {
	// List returns a list of supports
	List() ([]*entity.Support, error)
	Get() (*entity.Support, error)
	Page() (*entity.Support, error)
	Create(e *dto.SupportDto) (*entity.Support, error)
}

// NetworkMapInterface user's network map
type NetworkMapInterface interface {
	GetNetworkMap(pubKey, userId string) (*entity.NetworkMap, error)
}
