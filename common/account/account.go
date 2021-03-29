package account

import (
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"whitenoise/common/log"
	"whitenoise/crypto"

	"github.com/syndtr/goleveldb/leveldb"
	"whitenoise/common/store"
)

const DB_DIR = "./db"

type LevelDB struct {
	db *store.LevelDBStore
}

type Account struct {
	pubKey  []byte
	privKey []byte
}

func GetAccount() *Account {
	leveldb, err := OpenLevelDB(DB_DIR)
	if err != nil {
		log.Error("in GetAccount open leveldb err:", err.Error())
	}

	account, err := leveldb.QueryDefaultAccount()
	if account != nil && err == nil {
		log.Info("get default account from leveldb")
		return account
	}

	if account == nil && err == nil {
		r := rand.Reader
		priv, err := crypto.GenerateECDSAKeyPair(r)
		if err != nil {
			return nil
		}
		err, acnt := leveldb.InsertOrUpdateAccount(priv)
		if err != nil {
			log.Error("in GetAccount, insert or update accout err:", err.Error())
			return nil
		}
		log.Info("no account, create default one successfully.")
		return acnt
	}

	return nil
}

func NewOneTimeAccount() *Account {
	r := rand.Reader
	priv, err := crypto.GenerateECDSAKeyPair(r)
	if err != nil {
		return nil
	}
	pubBytes, err := crypto.MarshallECDSAPublicKey(&priv.PublicKey)
	if err != nil {
		log.Error("in InsertOrUpdateAccount, pubKey.Bytes() err:", err.Error())
		return nil
	}

	privBytes, err := crypto.MarshallECDSAPrivateKey(priv)
	if err != nil {
		log.Error("in InsertOrUpdateAccount, privKey.Bytes() err:", err.Error())
		return nil
	}
	return &Account{pubKey: pubBytes, privKey: privBytes}
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

func (this *LevelDB) InsertOrUpdateAccount(priv *ecdsa.PrivateKey) (error, *Account) {
	pubBytes, err := crypto.MarshallECDSAPublicKey(&priv.PublicKey)
	if err != nil {
		log.Error("in InsertOrUpdateAccount, pubKey.Bytes() err:", err.Error())
		return err, nil
	}

	privBytes, err := crypto.MarshallECDSAPrivateKey(priv)
	if err != nil {
		log.Error("in InsertOrUpdateAccount, privKey.Bytes() err:", err.Error())
		return err, nil
	}
	info := &Account{pubKey: pubBytes, privKey: privBytes}
	return this.db.Put([]byte("default"), privBytes), info
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

	info := &Account{privKey: value}
	return info, nil
}

func (this *LevelDB) DeleteAccount(pubKey *ecdsa.PublicKey) error {
	if rawKey, err := crypto.MarshallECDSAPublicKey(pubKey); err == nil {
		return this.db.Delete(rawKey)
	} else {
		return err
	}
}

func (this *Account) GetP2PPrivKey() crypto2.PrivKey {
	if this == nil {
		log.Error("account is nil, please use an valid one.")
		return nil
	}
	privKey, err := crypto.UnMarshallECDSAPrivateKey(this.privKey)
	if err != nil {
		log.Error("UnMarshallECDSAPrivateKey err:", err.Error())
		return nil
	}
	privP2P, _, err := crypto.P2PKeypairFromECDSA(privKey)
	if err != nil {
		log.Error("P2PKeypairFromECDSA err:", err.Error())
	}
	return privP2P
}

func (this *Account) GetECIESPrivKey() *ecies.PrivateKey {
	if this == nil {
		log.Error("account is nil, please use an valid one.")
		return nil
	}
	privKey, err := crypto.UnMarshallECDSAPrivateKey(this.privKey)
	if err != nil {
		log.Error("UnMarshallECDSAPrivateKey err:", err.Error())
		return nil
	}
	return crypto.ECIESKeypairFromECDSA(privKey)
}

//whitenoise ID is the marshall of ecdsa publickey
func (this *Account) GetWhiteNoiseID() WhiteNoiseID {
	if this == nil {
		log.Error("account is nil, please use an valid one.")
		return nil
	}
	privKey, err := crypto.UnMarshallECDSAPrivateKey(this.privKey)
	if err != nil {
		log.Error("UnMarshallECDSAPrivateKey err:", err.Error())
		return nil
	}

	id, err := crypto.MarshallECDSAPublicKey(&privKey.PublicKey)
	if err != nil {
		log.Error("MarshallECDSAPublicKey err:", err.Error())
		return nil
	}

	return id
}
