package crawler

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type IndexEntry struct {
	Url    string   `json:"url"`
	Text   string   `json:"text"`
	Links  []string `json:"outLinks"`
	Images []string `json:"imageLinks"`
}

type IndexQueue interface {
	Enque(entry IndexEntry) error
	EnqueMultiple(entries []IndexEntry) error
	GetEntries(n int) ([]IndexEntry, error)
}

type RedisIndexQueue struct {
	client  *redis.Client
	ctx     context.Context
	scripts map[ScriptId]Script
}

func NewRedisIndexQueue(addr string, password string, db int, protocol int) *RedisIndexQueue {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		Protocol: protocol,
	})
	ctx := context.Background()
	return &RedisIndexQueue{client: client, ctx: ctx, scripts: map[ScriptId]Script{}}
}

// TODO: pull this and runScript out into like a script manager class, because index queue and url queue have this same logic
func (riq *RedisIndexQueue) loadScript(id ScriptId) error {
	script := Script{id: id}
	riq.scripts[id] = script
	return script.load(riq.ctx, riq.client)
}

func (riq *RedisIndexQueue) runScript(id ScriptId, keys []string, args ...interface{}) (interface{}, error) {
	script, ok := riq.scripts[id]
	if !ok {
		riq.loadScript(id)
		script = riq.scripts[id]
	}
	return script.exec(riq.ctx, riq.client, keys, args)
}

func ReuseClient(client *redis.Client) *RedisIndexQueue {
	ctx := context.Background()
	return &RedisIndexQueue{client: client, ctx: ctx, scripts: map[ScriptId]Script{}}
}

func (riq *RedisIndexQueue) Enque(entry IndexEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = riq.client.RPush(riq.ctx, "indexQueue", data).Result()
	return err
}

// TODO: apparently using rPush with interface{}... is faster, investigate
func (riq *RedisIndexQueue) EnqueMultiple(entries []IndexEntry) error {
	pipe := riq.client.Pipeline()
	for _, entry := range entries {
		pipe.RPush(riq.ctx, "indexQueue", entry)
	}
	_, err := pipe.Exec(riq.ctx)
	return err
}

func (riq *RedisIndexQueue) GetEntries(n int) ([]IndexEntry, error) {
	keys := []string{"indexQueue"}
	val, err := riq.runScript(popN, keys, n)
	if err != nil {
		return nil, err
	}
	valSlice := val.([]interface{})
	out := make([]IndexEntry, len(valSlice))
	for i, entry := range valSlice {
		err = json.Unmarshal([]byte(entry.(string)), &out[i])
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
