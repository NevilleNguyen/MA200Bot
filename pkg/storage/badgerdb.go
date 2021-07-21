package storage

import (
	"bytes"
	"encoding/gob"
	"log"

	badger "github.com/dgraph-io/badger/v3"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type BadgerDB struct {
	l  *zap.SugaredLogger
	db *badger.DB
}

func NewBadgerDB() (*BadgerDB, error) {
	l := zap.S()
	storagePath := viper.GetString("storage_path")
	if err := validation.Validate(storagePath, validation.Required); err != nil {
		l.Errorw("storage_path is required", "error", err)
		return nil, err
	}
	db, err := badger.Open(badger.DefaultOptions(storagePath))
	if err != nil {
		l.Errorw("init badger db error", "error", err)
		return nil, err
	}
	return &BadgerDB{
		l:  l,
		db: db,
	}, nil
}

func (b *BadgerDB) Set(key string, value interface{}) error {
	err := b.db.Update(func(txn *badger.Txn) error {
		dataB, err := Encode(value)
		if err != nil {
			return err
		}
		if err := txn.Set([]byte(key), dataB); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (b *BadgerDB) Get(key string, value interface{}) error {
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		var valCopy []byte
		err = item.Value(func(val []byte) error {
			valCopy = append(valCopy, val...)
			return nil
		})
		if err != nil {
			return err
		}
		if err := Decode(valCopy, value); err != nil {
			return err
		}
		return nil
	})
	return err
}

func Encode(data interface{}) ([]byte, error) {
	var bufer bytes.Buffer
	enc := gob.NewEncoder(&bufer)
	err := enc.Encode(data)
	if err != nil {
		log.Print(err)
		return []byte{}, err
	}
	return bufer.Bytes(), nil
}

// Decode decoding interface into []byte, using lib encoding/gob
func Decode(in []byte, out interface{}) error {
	buffer := bytes.NewBuffer(in)
	dec := gob.NewDecoder(buffer)
	return dec.Decode(out)
}
