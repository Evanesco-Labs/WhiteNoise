package account

import (
	"crypto/rand"
	"errors"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
	"github.com/Evanesco-Labs/WhiteNoise/crypto"
	"github.com/Evanesco-Labs/WhiteNoise/internal/pb"
	"github.com/golang/protobuf/proto"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"io/ioutil"
	"os"

	"github.com/Evanesco-Labs/WhiteNoise/common/store"
	"github.com/syndtr/goleveldb/leveldb"
)

const DB_DIR = "./db"

const DefaultKeyType = crypto.DefaultKeyType

type LevelDB struct {
	db *store.LevelDBStore
}

type Account struct {
	KeyType int
	pubKey  crypto.PublicKey
	privKey crypto.PrivateKey
}

func GetAccount(keyType int) *Account {
	leveldb, err := OpenLevelDB(DB_DIR)
	if err != nil {
		log.Error("in GetAccount open leveldb err:", err.Error())
	}

	account, err := leveldb.QueryDefaultAccount()
	if account != nil && err == nil && account.KeyType == keyType {
		log.Info("get default account from leveldb")
		return account
	}

	r := rand.Reader
	priv, pub, err := crypto.GenerateKeyPair(keyType, r)
	if err != nil {
		return nil
	}
	account = &Account{
		KeyType: keyType,
		pubKey:  pub,
		privKey: priv,
	}
	err = leveldb.InsertOrUpdateAccount(account)
	if err != nil {
		log.Error("in GetAccount, insert or update accout err:", err.Error())
		return nil
	}
	log.Info("no account, create default one successfully.")
	return account
}

func GetAccountFromFile(path string) *Account {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if err == os.ErrNotExist {
			log.Error("key file not exit")
		}
		return nil
	}
	if fileInfo.IsDir() {
		return nil
	}

	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		log.Error("open file err", err)
		return nil
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil
	}
	priv, err := crypto.DecodeEcdsaPriv(string(data))
	if err != nil {
		log.Error(err)
		return nil
	}
	privKey := crypto.ECDSAPrivateKey{Priv: priv}

	return &Account{
		KeyType: crypto.ECDSA,
		pubKey:  privKey.Public(),
		privKey: privKey,
	}
}

func NewOneTimeAccount(keyType int) (*Account, error) {
	r := rand.Reader
	priv, pub, err := crypto.GenerateKeyPair(keyType, r)
	if err != nil {
		return nil, err
	}
	return &Account{pubKey: pub, privKey: priv, KeyType: keyType}, nil
}

func OpenLevelDB(path string) (*LevelDB, error) {
	if ldb, err := store.NewLevelDBStore(path); err == nil {
		return NewLevelDB(ldb), nil
	} else {
		log.Error("open leveldb error:%s, setting path:%s", err.Error(), path)
		return nil, err
	}
}

func NewLevelDB(d *store.LevelDBStore) *LevelDB {
	p := &LevelDB{
		db: d,
	}
	return p
}

func (this *LevelDB) Close() {
	this.db.Close()
}

func (this *LevelDB) InsertOrUpdateAccount(acc *Account) error {
	pbAccount := pb.Account{
		Type:       int32(acc.KeyType),
		PrivateKey: acc.privKey.Bytes(),
		PublicKey:  acc.pubKey.Bytes(),
	}
	data, err := proto.Marshal(&pbAccount)
	if err != nil {
		return err
	}
	return this.db.Put([]byte("default"), data)
}

func (this *LevelDB) QueryDefaultAccount() (*Account, error) {
	return this.QueryAccount("default")
}

func (this *LevelDB) QueryAccount(label string) (*Account, error) {
	key := []byte(label)
	value, err := this.db.Get(key)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
	}
	if len(value) == 0 {
		return nil, nil
	}

	pbAccount := pb.Account{}
	err = proto.Unmarshal(value, &pbAccount)
	if err != nil {
		return nil, err
	}
	switch int(pbAccount.Type) {
	case crypto.Ed25519:
		publicKey, err := crypto.UnMarshallEd25519PublicKey(pbAccount.PublicKey)
		if err != nil {
			return nil, err
		}
		privateKey, err := crypto.UnMarshallEd25519PrivateKey(pbAccount.PrivateKey)
		if err != nil {
			return nil, err
		}
		return &Account{
			KeyType: int(pbAccount.Type),
			pubKey:  publicKey,
			privKey: privateKey,
		}, nil
	case crypto.ECDSA:
		publicKey, err := crypto.UnMarshallECDSAPublicKey(pbAccount.PublicKey)
		if err != nil {
			return nil, err
		}
		privateKey, err := crypto.UnMarshallECDSAPrivateKey(pbAccount.PrivateKey)
		if err != nil {
			return nil, err
		}
		return &Account{
			KeyType: int(pbAccount.Type),
			pubKey:  publicKey,
			privKey: privateKey,
		}, nil
	case crypto.Secpk1:
		publicKey, err := crypto.UnMarshallSecp256k1PublicKey(pbAccount.PublicKey)
		if err != nil {
			return nil, err
		}
		privateKey, err := crypto.UnMarshallSecp256k1PrivateKey(pbAccount.PrivateKey)
		if err != nil {
			return nil, err
		}
		return &Account{
			KeyType: int(pbAccount.Type),
			pubKey:  publicKey,
			privKey: privateKey,
		}, nil
	default:
		return nil, errors.New("unsupport key type")
	}
}

func (acc *Account) GetPrivateKey() crypto.PrivateKey {
	return acc.privKey
}

func (acc *Account) GetPublicKey() crypto.PublicKey {
	return acc.pubKey
}

func (acc *Account) GetP2PPrivKey() crypto2.PrivKey {
	if acc == nil {
		log.Error("account is nil, please use an valid one.")
		return nil
	}

	priv, _, err := acc.GetPrivateKey().GetP2PKeypair()
	if err != nil {
		log.Error(err)
		return nil
	}
	return priv
}
