package contrib

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Mutex 分布式锁
type Mutex interface {
	// Lock 获取锁
	Lock(ctx context.Context) (bool, error)
	// TryLock 尝试获取锁
	TryLock(ctx context.Context, attempts int, delay time.Duration) (bool, error)
	// UnLock 释放锁
	UnLock(ctx context.Context) error
}

// redLock 基于「Redis」实现的分布式锁
type redLock struct {
	cli    *redis.Client
	key    string
	token  string
	expire time.Duration
}

func (l *redLock) Lock(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done(): // timeout or canceled
		return false, ctx.Err()
	default:
	}

	if err := l.lock(ctx); err != nil {
		return false, err
	}
	return len(l.token) != 0, nil
}

func (l *redLock) TryLock(ctx context.Context, attempts int, interval time.Duration) (bool, error) {
	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done(): // timeout or canceled
			return false, ctx.Err()
		default:
		}

		// attempt to acquire lock
		if err := l.lock(ctx); err != nil {
			return false, err
		}
		if len(l.token) != 0 {
			return true, nil
		}
		time.Sleep(interval)
	}
	return false, nil
}

func (l *redLock) UnLock(ctx context.Context) error {
	if len(l.token) == 0 {
		return nil
	}

	script := `
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
else
	return 0
end
`
	return l.cli.Eval(context.WithoutCancel(ctx), script, []string{l.key}, l.token).Err()
}

func (l *redLock) lock(ctx context.Context) error {
	token := uuid.New().String()
	ok, err := l.cli.SetNX(ctx, l.key, token, l.expire).Result()
	if err != nil {
		// 尝试GET一次：避免因redis网络错误导致误加锁
		v, _err := l.cli.Get(ctx, l.key).Result()
		if _err != nil {
			if errors.Is(_err, redis.Nil) {
				return err
			}
			return _err
		}
		if v == token {
			l.token = token
		}
		return nil
	}
	if ok {
		l.token = token
	}
	return nil
}

// RedLock 基于Redis实现的分布式锁实例
func RedLock(cli *redis.Client, key string, ttl time.Duration) Mutex {
	mutex := &redLock{
		cli:    cli,
		key:    key,
		expire: ttl,
	}
	if mutex.expire == 0 {
		mutex.expire = time.Second * 10
	}
	return mutex
}
