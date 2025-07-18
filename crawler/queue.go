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
	scripts map[ScriptId]string
}

func NewRedisQueue(addr string, password string, db int, protocol int) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		Protocol: protocol,
	})
	ctx := context.Background()
	return &RedisQueue{client: client, ctx: ctx, scripts: map[ScriptId]string{}}
}

func getScriptPath(id ScriptId) string {
	return strings.Join([]string{"redisScripts/", id.String(), ".lua"}, "")
}

func (rq *RedisQueue) loadScript(id ScriptId) error {
	scriptContent, err := os.ReadFile(getScriptPath(id))
	if err != nil {
		return err
	}
	sha, err := rq.client.ScriptLoad(rq.ctx, string(scriptContent)).Result()
	if err != nil {
		return nil
	}
	rq.scripts[id] = sha
	return nil
}

func (rq *RedisQueue) runScript(id ScriptId, keys []string, args ...interface{}) (interface{}, error) {
	res, err := rq.client.EvalSha(rq.ctx, rq.scripts[id], keys, args...).Result()
	if err != nil && strings.Contains(err.Error(), "NOSCRIPT") {
		err = rq.loadScript(id)
		if err != nil {
			return res, err
		}
		res, err = rq.client.EvalSha(rq.ctx, rq.scripts[id], keys, args).Result()
	}
	return res, err
}

func (rq *RedisQueue) loadAndRunScript(path string, keys []string, args ...interface{}) (interface{}, error) {
	scriptContent, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	script := redis.NewScript(string(scriptContent))
	return script.Run(rq.ctx, rq.client, keys, args).Result()
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
