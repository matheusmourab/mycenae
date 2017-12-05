package cache

import (
	"github.com/uol/gobol"
	"github.com/uol/mycenae/lib/keyspace"
	"github.com/uol/mycenae/lib/memcached"
	"net/http"
)

type KeyspaceCache struct {
	memcached *memcached.Memcached
	keyspace  *keyspace.Keyspace
}

func NewKeyspaceCache(mc *memcached.Memcached, ks *keyspace.Keyspace) *KeyspaceCache {

	return &KeyspaceCache{
		memcached: mc,
		keyspace:  ks,
	}
}

func (kc *KeyspaceCache) GetKeyspace(key string) (string, bool, gobol.Error) {

	v, gerr := kc.memcached.Get("keyspace", key)
	if gerr != nil {
		return "", false, gerr
	}

	if v != nil {
		return string(v), true, nil
	}

	ks, found, gerr := kc.keyspace.GetKeyspace(key)
	if gerr != nil {
		if gerr.StatusCode() == http.StatusNotFound {
			return "", false, nil
		}
		return "", false, gerr
	}

	if !found {
		return "", false, nil
	}

	value := "false"

	if ks.TUUID {
		value = "true"
	}

	gerr = kc.memcached.Put("keyspace", key, []byte(value))
	if gerr != nil {
		return "", false, gerr
	}

	return value, true, nil
}

func (kc *KeyspaceCache) GetTsNumber(key string, CheckTSID func(esType, id string) (bool, gobol.Error)) (bool, gobol.Error) {
	return kc.getTSID("meta", "number", key, CheckTSID)
}

func (kc *KeyspaceCache) GetTsText(key string, CheckTSID func(esType, id string) (bool, gobol.Error)) (bool, gobol.Error) {
	return kc.getTSID("metatext", "text", key, CheckTSID)
}

func (kc *KeyspaceCache) getTSID(esType, bucket, key string, CheckTSID func(esType, id string) (bool, gobol.Error)) (bool, gobol.Error) {

	v, gerr := kc.memcached.Get(bucket, key)
	if gerr != nil {
		return false, gerr
	}
	if v != nil {
		return true, nil
	}

	found, gerr := CheckTSID(esType, key)
	if gerr != nil {
		return false, gerr
	}
	if !found {
		return false, nil
	}

	gerr = kc.memcached.Put(bucket, key, []byte{})
	if gerr != nil {
		return false, gerr
	}

	return true, nil
}
