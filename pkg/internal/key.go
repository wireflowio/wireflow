package internal

import (
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"sync"
)

type KeyManager struct {
	lock       sync.Mutex
	privateKey wgtypes.Key
}

func NewKeyManager(privateKey wgtypes.Key) *KeyManager {
	return &KeyManager{privateKey: privateKey}
}

func (km *KeyManager) UpdateKey(privateKey wgtypes.Key) {
	km.lock.Lock()
	defer km.lock.Unlock()
	km.privateKey = privateKey
}

func (km *KeyManager) GetKey() wgtypes.Key {
	km.lock.Lock()
	defer km.lock.Unlock()
	return km.privateKey
}

func (km *KeyManager) GetPublicKey() wgtypes.Key {
	km.lock.Lock()
	defer km.lock.Unlock()
	return km.privateKey.PublicKey()
}

//
//type Key wgtypes.Key
//
//type PublicKey struct {
//	Key
//}
//
//func (k Key) GetBytes() []byte {
//	return k[:]
//}
//
//type privateKey struct {
//	Key
//}
//
//func ParsePublicKey(key []byte) *PublicKey {
//	return &PublicKey{Key(key)}
//}
//
//type PairKey struct {
//	privateKey privateKey
//	PublicKey  PublicKey
//}
//
//func (k Key) String() string {
//	return base64.StdEncoding.EncodeToString(k[:])
//}
//
//func ParseKey(value string) (*Key, error) {
//	key, err := base64.StdEncoding.DecodeString(value)
//	return (*Key)(key), err
//}
//
//func NewPairKey() (*PairKey, error) {
//	privateKey, err := wgtypes.GeneratePrivateKey()
//	if err != nil {
//		return nil, err
//	}
//
//	return &PairKey{
//		privateKey: privateKey{Key(privateKey)},
//		PublicKey:  PublicKey{Key(privateKey.PublicKey())},
//	}, nil
//}
