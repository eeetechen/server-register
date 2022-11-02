package lock

import (
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestEtcdLock_TryLock(t *testing.T) {
	cli := NewEtcd(&Conf{
		Addr:        []string{"127.0.0.1:2379"},
		DialTimeout: 5,
	})
	lock := NewEtcdLock(cli, "/get/post", 5)

	if err := lock.TryLock(); err != nil {
		zap.S().Info("1 err", err)
		return
	}
	zap.S().Info("1 success")
	go zap.S().Info(lock.TryLock())
	go zap.S().Info(lock.TryLock())
	go zap.S().Info(lock.TryLock())
	go zap.S().Info(lock.TryLock())
	go zap.S().Info(lock.TryLock())
	time.Sleep(3 * time.Second)
	zap.S().Info(1, lock.UnLock())
	go func() {
		time.Sleep(6 * time.Second)
		if err := lock.TryLock(); err != nil {
			zap.S().Info("2 err")
			return
		}
		zap.S().Info("2 success")
	}()
	go func() {
		time.Sleep(6 * time.Second)
		if err := lock.TryLock(); err != nil {
			zap.S().Info("3 err")
			return
		}
		zap.S().Info("3 success")
	}()
	select {}
}
