package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type StaticConfig struct {
	ID        uuid.UUID         `json:"id"`
	Targets   []string          `json:"targets"`
	Labels    map[string]string `json:"labels"`
	Group     string            `json:"target_group"`
	Modules   []string          `json:"modules,omitempty"`
	UpdatedAt string            `json:"updated_at"` // unix timestamp
}

type SDTargets struct {
	Items map[uuid.UUID]StaticConfig
}

var (
	ErrInsertFailed    = errors.New("Insert failed")
	ErrDeleteFailed    = errors.New("Delete failed")
	ErrMarshlFailed    = errors.New("Marshal failed")
	ErrUnmarshalFailed = errors.New("Unmarshal failed")
	ErrIDNotFound      = errors.New("Id not found")
	ErrNoKeysFound     = errors.New("No keys found")
)

func (c *SDTargets) Insert(target StaticConfig, ctx context.Context, con *redis.Client, ttl int) (uuid.UUID, error) {
	target.ID = UUIDFromStringArray(target.Targets)
	target.UpdatedAt = fmt.Sprint(time.Now().Unix())
	if target.Group == "" {
		target.Group = "inventor-default"
	}
	srt, err := json.Marshal(target)
	if err != nil {
		return target.ID, ErrMarshlFailed
	}
	result, err := con.Set(ctx, fmt.Sprint(target.ID), srt, time.Duration(ttl)*time.Second).Result()
	if err != nil {
		return target.ID, ErrInsertFailed
	}
	log.Printf("Creating %s: %s", target.ID, result)
	return target.ID, nil
}

func (c *SDTargets) Delete(id uuid.UUID, ctx context.Context, con *redis.Client) (bool, error) {
	result, err := con.Del(ctx, id.String()).Result()
	if err != nil {
		return false, ErrDeleteFailed
	}
	if result == 0 {
		return false, ErrIDNotFound
	}
	log.Printf("Deleting target uuid: %s, count: %v", id, result)
	return true, nil
}

func (c *SDTargets) Retrieve(id uuid.UUID, ctx context.Context, con *redis.Client) (StaticConfig, error) {
	result, err := con.Get(ctx, fmt.Sprint(id)).Result()
	if err != nil {
		return StaticConfig{}, ErrIDNotFound
	}
	rel := StaticConfig{}
	if err := json.Unmarshal([]byte(result), &rel); err != nil {
		return StaticConfig{}, ErrUnmarshalFailed
	}
	return rel, nil
}

func (c *SDTargets) Scan(ctx context.Context, con *redis.Client) (SDTargets, error) {
	targets := SDTargets{
		Items: make(map[uuid.UUID]StaticConfig),
	}

	var uids []uuid.UUID
	var keys []string
	iter := con.Scan(ctx, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		uid, err := uuid.Parse(iter.Val())
		if err != nil {
			log.Printf("Can't parse uuid: %s, %v", iter.Val(), err)
			continue
		}
		uids = append(uids, uid)
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return targets, err
	}
	if len(keys) == 0 {
		return targets, nil
	}

	values, err := con.MGet(ctx, keys...).Result()
	if err != nil {
		return targets, err
	}
	for i, val := range values {
		if val == nil {
			// Key expired between SCAN and MGET.
			continue
		}
		str, ok := val.(string)
		if !ok {
			log.Printf("Unexpected value type for key %s", keys[i])
			continue
		}
		var item StaticConfig
		if err := json.Unmarshal([]byte(str), &item); err != nil {
			log.Printf("Can't unmarshal target %s: %v", keys[i], err)
			continue
		}
		targets.Items[uids[i]] = item
	}

	return targets, nil
}

func UUIDFromStringArray(str []string) uuid.UUID {
	// JSON-encode rather than join with a plain separator: each element is
	// quoted, so a comma inside one address can't be confused with the
	// separator between two addresses (e.g. ["a,b"] vs ["a","b"]).
	b, _ := json.Marshal(str)
	return uuid.NewSHA1(uuid.NameSpaceDNS, b)
}
