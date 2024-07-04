package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
)

type StaticConfig struct {
	ID        uuid.UUID         `json:"id"`
	Targets   []string          `json:"targets"`
	Labels    map[string]string `json:"labels"`
	Group     string            `json:"target_group"`
	UpdatedAt string            `json:"updated_at"` // unix timestamp
}

type SDTargets struct {
	Items map[uuid.UUID]StaticConfig
}

var (
	ErrInsertFailed = errors.New("Insert failed")
	ErrDeleteFailed = errors.New("Delete failed")
	ErrMarshlFailed = errors.New("Marshal failed")
	ErrIDNotFound   = errors.New("Id not found")
	ErrNoKeysFound  = errors.New("No keys found")
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
	json.Unmarshal([]byte(result), &rel)
	return rel, nil
}

func (c *SDTargets) Scan(ctx context.Context, con *redis.Client) (SDTargets, error) {
	var targets = SDTargets{
		Items: make(map[uuid.UUID]StaticConfig),
	}
	iter := con.Scan(ctx, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		uid, err := uuid.Parse(iter.Val())
		if err != nil {
			log.Printf("Can't parse uuid: %s, %v", iter.Val(), err)
			continue
		} else {
			targets.Items[uid], _ = c.Retrieve(uid, ctx, con)
		}
	}
	if err := iter.Err(); err != nil {
		panic(err)
	}
	return targets, nil
}

func UUIDFromStringArray(str []string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(strings.Join(str, ",")))
}
