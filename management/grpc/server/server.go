package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"linkany/management/controller"
	"linkany/management/dto"
	"linkany/management/entity"
	"linkany/management/grpc/mgt"
	"linkany/management/mapper"
	"linkany/management/utils"
	"net"
	"strconv"
	"time"
)

// Server is used to implement helloworld.GreeterServer.
type Server struct {
	mgt.UnimplementedManagementServiceServer
	userController *controller.UserController
	peerController *controller.PeerController
	port           int
	tokenr         *utils.Tokener
}

type ServerConfig struct {
	Port            int
	Database        mapper.DatabaseConfig
	DataBaseService *mapper.DatabaseService
}

func NewServer(cfg *ServerConfig) *Server {
	return &Server{
		port:           cfg.Port,
		userController: controller.NewUserController(mapper.NewUserMapper(cfg.DataBaseService)),
		peerController: controller.NewPeerController(mapper.NewPeerMapper(cfg.DataBaseService)),
	}
}

func (s *Server) Login(ctx context.Context, in *mgt.ManagementMessage) (*mgt.ManagementMessage, error) {
	var req mgt.LoginRequest
	if err := proto.Unmarshal(in.Body, &req); err != nil {
		return nil, err
	}
	klog.Infof("Received username: %s, password: %s", req.Username, req.Password)

	token, err := s.userController.Login(&dto.UserDto{
		Username: req.Username,
		Password: req.Password,
	})

	if err != nil {
		return nil, err
	}

	b, err := proto.Marshal(&mgt.LoginResponse{Token: token.Token})
	if err != nil {
		return nil, err
	}

	return &mgt.ManagementMessage{
		Body: b,
	}, nil
}

// List will return a list of response
func (s *Server) List(ctx context.Context, in *mgt.ManagementMessage) (*mgt.ManagementMessage, error) {
	var req mgt.Request
	if err := proto.Unmarshal(in.Body, &req); err != nil {
		return nil, err
	}
	user, err := s.userController.Get(req.GetToken())
	if err != nil {
		return nil, err
	}
	klog.Infoln(user)
	peers, err := s.peerController.GetNetworkMap(req.AppId, strconv.Itoa(int(user.ID)))
	if err != nil {
		return nil, err
	}

	bs, err := json.Marshal(peers)
	if err != nil {
		return nil, err
	}

	return &mgt.ManagementMessage{Body: bs}, nil
}

// Watch once request, will return a stream of watched response
func (s *Server) Watch(server mgt.ManagementService_WatchServer) error {
	var err error
	var msg *mgt.ManagementMessage
	msg, err = server.Recv()
	if err != nil {
		return err
	}

	var req mgt.Request
	if err = proto.Unmarshal(msg.Body, &req); err != nil {
		return err
	}

	// create a chan for the peer
	watchChannel := CreateChannel(req.PubKey)
	klog.Infof("peer %v is now watching", req.PubKey)
	for {
		select {
		case wm := <-watchChannel:
			bs, err := proto.Marshal(wm)
			if err != nil {
				return err
			}

			msg := &mgt.ManagementMessage{PubKey: req.PubKey, Body: bs}
			if err = server.Send(msg); err != nil {
				return err
			}
		}
	}

}

// Keepalive acts as a client is living， server will send 'ping' packet to client
// client will response packet to server with in 10 seconds, if not, client is offline, otherwise onlie.
func (s *Server) Keepalive(server mgt.ManagementService_KeepaliveServer) error {
	var err error
	var msg *mgt.ManagementMessage
	var pubKey string
	var userId string

	ctx := context.Background()
	msg, err = server.Recv()
	var req mgt.Request
	if err = proto.Unmarshal(msg.Body, &req); err != nil {
		return err
	}
	pubKey = req.PubKey

	user, err := s.tokenr.Parse(req.Token)
	if err != nil {
		klog.Errorf("invalid token")
		return err
	}

	userId = fmt.Sprintf("%v", user.ID)
	// record
	var wc chan *mgt.WatchMessage
	wc = utils.NewWatchManager().Get(pubKey)
	if wc == nil {
		return fmt.Errorf("fatal error, peer has not connected to managent server")
	}

	currentPeer := &mgt.Peer{
		PublicKey: pubKey,
	}

	var online = 1
	var peers []*entity.Peer

	check := func(ctx context.Context) error {
		msg, err = server.Recv()
		if err != nil {
			klog.Errorf("peer %s connected broken, notify user's clients remove this peer", pubKey)
			peers, err = s.peerController.List(&mapper.QueryParams{
				PubKey: &pubKey,
				UserId: &userId,
				Online: &online,
			})

			if err != nil {
				klog.Errorf("list peers failed: %v", err)
			}

			s.handleKeepalive(mgt.EventType_DELETE, currentPeer, peers)
			dtoParam := &dto.PeerDto{PubKey: pubKey, Online: 0}
			_, err = s.peerController.Update(dtoParam)
			return err
		}

		var req mgt.Request
		if err = proto.Unmarshal(msg.Body, &req); err != nil {
			return err
		}

		peers, err = s.peerController.List(&mapper.QueryParams{
			PubKey: &pubKey,
			UserId: &userId,
			Online: &online,
		})

		if err != nil {
			klog.Errorf("list peers failed: %v", err)
			return err
		}

		return nil

	}

	newCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err = check(newCtx); err != nil {
		klog.Errorf("check failed: %v", err)
		return err
	} else {
		s.handleKeepalive(mgt.EventType_ADD, currentPeer, peers)
	}

	timer := time.NewTimer(30 * time.Second)
	for {
		select {
		case <-timer.C:
			if err = server.Send(&mgt.ManagementMessage{Body: []byte("ping")}); err != nil {
				klog.Errorf("send ping failed: %v", err)
				continue
			}

			var checkChannel chan interface{}
			// work
			go func() {
				if err = check(ctx); err != nil {
					klog.Errorf("check failed: %v", err)
				}
				close(checkChannel)
			}()

			// check 10s receive the response
			newCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			select {
			case <-newCtx.Done():
				//timeout
				s.handleKeepalive(mgt.EventType_DELETE, currentPeer, peers)
				dtoParam := &dto.PeerDto{PubKey: pubKey, Online: 0}
				_, err = s.peerController.Update(dtoParam)
				return nil
			case <-checkChannel:
				// online
				dtoParam := &dto.PeerDto{PubKey: pubKey, Online: 1}
				_, err = s.peerController.Update(dtoParam)
				klog.Infof("peer %v is online", pubKey)
			}
		}
	}

}

func (s *Server) handleKeepalive(eventType mgt.EventType, current *mgt.Peer, peers []*entity.Peer) {
	manager := utils.NewWatchManager()
	for _, peer := range peers {
		wc := manager.Get(peer.PublicKey)
		message := utils.NewWatchMessage(eventType, current)
		// add to channel, will send to client
		wc <- message
	}
}

func (s *Server) Start() error {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	mgt.RegisterManagementServiceServer(grpcServer, s)
	klog.Infof("Grpc server listening at %v", listen.Addr())
	return grpcServer.Serve(listen)
}
