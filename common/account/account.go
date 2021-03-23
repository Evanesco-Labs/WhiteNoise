/**
 * Description:
 * Author: Yihen.Liu
 * Create: 2019-08-01
 */
package account

import (
	"crypto/rand"
	"whitenoise/common/log"

	"github.com/libp2p/go-libp2p-core/crypto"
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
		priv, pubk, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
		if err != nil {
			return nil
		}
		err, acnt := leveldb.InsertOrUpdateAccount(pubk, priv)
		if err != nil {
			log.Error("in GetAccount, insert or update accout err:", err.Error())
			return nil
		}
		log.Info("no account, create default one successfully.")
		return acnt
	}

	return nil
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

func (this *LevelDB) InsertOrUpdateAccount(pubKey crypto.PubKey, privKey crypto.PrivKey) (error, *Account) {
	pubBytes, err := pubKey.Raw()
	if err != nil {
		log.Error("in InsertOrUpdateAccount, pubKey.Bytes() err:", err.Error())
		return err, nil
	}

	privBytes, err := privKey.Raw()
	if err != nil {
		log.Error("in InsertOrUpdateAccount, privKey.Bytes() err:", err.Error())
		return err, nil
	}
	info := &Account{pubKey: pubBytes, privKey: privBytes}
	/*	buf, err := json.Marshal(info)
		if err != nil {
			return err, nil
		}*/
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
	/*	err = json.Unmarshal(value, info)
		if err != nil {
			return nil, err
		}*/
	return info, nil
}

func (this *LevelDB) DeleteAccount(pubKey crypto.PubKey) error {
	if rawKey, err := pubKey.Raw(); err == nil {
		return this.db.Delete(rawKey)
	} else {
		return err
	}
}

func (this *Account) GetPrivKey() crypto.PrivKey {
	if this == nil {
		log.Error("account is nil, please use an valid one.")
		return nil
	}

	if privKey, err := crypto.UnmarshalRsaPrivateKey(this.privKey); err == nil {
		return privKey
	} else {
		log.Error("unmarshal RSA-2048 privKey err:", err.Error())
		return nil
	}
}
