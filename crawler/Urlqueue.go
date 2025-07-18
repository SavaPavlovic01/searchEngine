package crawler

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

type ScriptId int

const (
	checkAndPushBatch ScriptId = iota
	checkAndPushSingle
	popN
)

var scriptNames = map[ScriptId]string{
	popN:               "popN",
	checkAndPushSingle: "checkAndPushSingle",
	checkAndPushBatch:  "checkAndPushBatch",
}

type Script struct {
	id  ScriptId
	sha string
}

func (s *Script) load(ctx context.Context, client *redis.Client) error {
	scriptContent, err := os.ReadFile(getScriptPath(s.id))
	if err != nil {
		return err
	}
	sha, err := client.ScriptLoad(ctx, string(scriptContent)).Result()
	if err != nil {
		return nil
	}
	s.sha = sha
	return nil
}

func (s *Script) exec(ctx context.Context, client *redis.Client, keys []string, args ...interface{}) (interface{}, error) {
	res, err := client.EvalSha(ctx, s.sha, keys, args...).Result()
	if err != nil && strings.Contains(err.Error(), "NOSCRIPT") {
		err = s.load(ctx, client)
		if err != nil {
			return res, err
		}
		res, err = client.EvalSha(ctx, s.sha, keys, args...).Result()
	}
	return res, err
}

func (s ScriptId) String() string {
	if name, ok := scriptNames[s]; ok {
		return name
	}
	return fmt.Sprintf("unknownScriptID(%d)", int(s))
}

type UrlQueue interface {
	Enque(url string) (bool, error)
	EnqueMultiple(url []string) (bool, error)
	GetUrls(urlCount int) ([]string, error)
}

type RedisQueue struct {
	client  *redis.Client
	ctx     context.Context
	scripts map[ScriptId]*Script
}

func NewRedisQueue(addr string, password string, db int, protocol int) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		Protocol: protocol,
	})
	ctx := context.Background()
	return &RedisQueue{client: client, ctx: ctx, scripts: map[ScriptId]*Script{}}
}

func getScriptPath(id ScriptId) string {
	return strings.Join([]string{"redisScripts/", id.String(), ".lua"}, "")
}

func (rq *RedisQueue) loadScript(id ScriptId) error {
	script := &Script{id: id}
	rq.scripts[id] = script
	return script.load(rq.ctx, rq.client)
}

func (rq *RedisQueue) runScript(id ScriptId, keys []string, args ...interface{}) (interface{}, error) {
	script, ok := rq.scripts[id]
	if !ok {
		rq.loadScript(id)
		script = rq.scripts[id]
	}
	return script.exec(rq.ctx, rq.client, keys, args)
}

func (rq *RedisQueue) Enque(url string) (bool, error) {
	keys := []string{"visited", "crawlQueue"}
	val, err := rq.runScript(checkAndPushSingle, keys, url)
	if err != nil {
		return false, err
	}
	return val.(int64) > 0, nil
}

func (rq *RedisQueue) EnqueMultiple(url []string) (bool, error) {
	keys := []string{"visited", "crawlQueue"}
	test := make([]interface{}, len(url))
	for i, s := range url {
		test[i] = s
	}
	val, err := rq.runScript(checkAndPushBatch, keys, test...)
	if err != nil {
		return false, err
	}
	return val.(int64) > 0, nil
}

func (rq *RedisQueue) GetUrls(urlCount int) ([]string, error) {
	keys := []string{"crawlQueue"}
	val, err := rq.runScript(popN, keys, urlCount)
	if err != nil {
		return nil, err
	}
	valSlice := val.([]interface{})
	out := make([]string, len(valSlice))
	for i, v := range valSlice {
		out[i] = v.(string)
	}
	return out, err
}
