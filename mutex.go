package contrib

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// DistributedMutex 分布式锁
type DistributedMutex interface {
	// Lock 获取锁
	Lock(ctx context.Context) (bool, error)
	// TryLock 尝试获取锁
	TryLock(ctx context.Context, attempts int, delay time.Duration) (bool, error)
	// UnLock 释放锁
	UnLock(ctx context.Context) error
}

// distributed 基于「Redis」实现的分布式锁
type distributed struct {
	cli    *redis.Client
	key    string
	token  string
	expire time.Duration
}

func (d *distributed) Lock(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done(): // timeout or canceled
		return false, ctx.Err()
	default:
	}

	if err := d.lock(ctx); err != nil {
		return false, err
	}
	return len(d.token) != 0, nil
}

func (d *distributed) TryLock(ctx context.Context, attempts int, interval time.Duration) (bool, error) {
	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done(): // timeout or canceled
			return false, ctx.Err()
		default:
		}

		// attempt to acquire lock
		if err := d.lock(ctx); err != nil {
			return false, err
		}
		if len(d.token) != 0 {
			return true, nil
		}
		time.Sleep(interval)
	}
	return false, nil
}

func (d *distributed) UnLock(ctx context.Context) error {
	if len(d.token) == 0 {
		return nil
	}

	script := `
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
else
	return 0
end
`
	return d.cli.Eval(context.WithoutCancel(ctx), script, []string{d.key}, d.token).Err()
}

func (d *distributed) lock(ctx context.Context) error {
	token := uuid.New().String()
	ok, err := d.cli.SetNX(ctx, d.key, token, d.expire).Result()
	if err != nil {
		// 尝试GET一次：避免因redis网络错误导致误加锁
		v, _err := d.cli.Get(ctx, d.key).Result()
		if _err != nil {
			if errors.Is(_err, redis.Nil) {
				return err
			}
			return _err
		}
		if v == token {
			d.token = token
		}
		return nil
	}
	if ok {
		d.token = token
	}
	return nil
}

// RedisMutex 基于Redis实现的分布式锁实例
func RedisMutex(cli *redis.Client, key string, ttl time.Duration) DistributedMutex {
	mutex := &distributed{
		cli:    cli,
		key:    key,
		expire: ttl,
	}
	if mutex.expire == 0 {
		mutex.expire = time.Second * 10
	}
	return mutex
}
