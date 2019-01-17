package cachita

import (
	"fmt"
	"github.com/vmihailenco/msgpack"
	"time"

	"github.com/mediocregopher/radix"
)

var rCache Cache

type redis struct {
	pool   *radix.Pool
	prefix string
	ttl    time.Duration
}

func Redis(addr string) (Cache, error) {
	if rCache == nil {
		var err error
		rCache, err = NewRedisCache(24*time.Hour, 10, "cachita", addr)
		if err != nil {
			return nil, err
		}
	}
	return rCache, nil
}

func NewRedisCache(ttl time.Duration, poolSize int, prefix, addr string) (Cache, error) {
	pool, err := radix.NewPool("tcp", addr, poolSize)
	if err != nil {
		return nil, err
	}

	c := &redis{
		pool:   pool,
		prefix: prefix,
		ttl:    ttl,
	}

	return c, nil
}

func (c *redis) Get(key string, i interface{}) error {
	s := i
	isInt := isInt(i)
	var data []byte
	if !isInt {
		s = &data
	}
	err := c.pool.Do(radix.FlatCmd(s, "GET", c.k(key)))
	if err != nil {
		return err
	}

	if isInt {
		i = s
		return nil
	}

	data = *s.(*[]byte)
	if data == nil {
		return ErrNotFound
	}
	return msgpack.Unmarshal(data, i)
}

func (c *redis) Put(key string, i interface{}, ttl time.Duration) error {
	s := i
	if !isInt(i) {
		data, err := msgpack.Marshal(i)
		if err != nil {
			return err
		}
		s = &data
	}

	return c.pool.Do(radix.FlatCmd(nil, "SETEX", c.k(key), calculateTtl(ttl, c.ttl).Seconds(), s))
}

func (c *redis) Incr(key string, ttl time.Duration) error {
	return c.pool.Do(radix.FlatCmd(nil, "INCR", c.k(key))) //, calculateTtl(ttl, c.ttl).Seconds()
}

func (c *redis) Invalidate(key string) error {
	return c.pool.Do(radix.FlatCmd(nil, "DEL", c.k(key)))
}

func (c *redis) Exists(key string) bool {
	var b bool
	c.pool.Do(radix.FlatCmd(&b, "EXISTS", c.k(key)))
	return b
}

func (c *redis) k(key string) string {
	return fmt.Sprintf("%s:%s", c.prefix, key)
}

func isInt(i interface{}) bool {
	switch i.(type) {
	case *int:
	case *int8:
	case *int16:
	case *int32:
	case *int64:
	case int:
	case int8:
	case int16:
	case int32:
	case int64:
	default:
		return false
	}
	return true
}
