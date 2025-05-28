package fs

import (
	"fmt"
	"sync"
	"time"
)

// LockType represents the type of lock
type LockType int

const (
	// ReadLock is a shared lock that allows multiple readers
	ReadLock LockType = iota
	// WriteLock is an exclusive lock that prevents all other access
	WriteLock
)

// LockInfo contains information about a lock
type LockInfo struct {
	Path      string        // Path being locked
	Type      LockType      // Type of lock
	Owner     string        // Identifier of the lock owner
	CreatedAt time.Time     // When the lock was created
	Timeout   time.Duration // How long until the lock expires
}

// ExplicitLockManager manages explicit locks for paths
type ExplicitLockManager struct {
	locks       map[string]*LockInfo       // Mapping of paths to locks
	mu          sync.Mutex                 // Mutex to protect the locks map
	lockWaiters map[string][]chan struct{} // Channels for waiters
	waiterMu    sync.Mutex                 // Mutex to protect the waiters map
}

// NewExplicitLockManager creates a new lock manager
func NewExplicitLockManager() *ExplicitLockManager {
	return &ExplicitLockManager{
		locks:       make(map[string]*LockInfo),
		lockWaiters: make(map[string][]chan struct{}),
	}
}

// AcquireLock attempts to acquire a lock on a path
func (lm *ExplicitLockManager) AcquireLock(path, owner string, lockType LockType, timeout time.Duration) (*LockInfo, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Check if there's an existing lock
	if existing, exists := lm.locks[path]; exists {
		// If there's a write lock, we can't acquire any type of lock
		if existing.Type == WriteLock {
			return nil, fmt.Errorf("path is write-locked by %s", existing.Owner)
		}

		// If there's a read lock and we want a read lock, that's fine
		if existing.Type == ReadLock && lockType == ReadLock {
			lock := &LockInfo{
				Path:      path,
				Type:      ReadLock,
				Owner:     owner,
				CreatedAt: time.Now(),
				Timeout:   timeout,
			}
			lm.locks[path] = lock
			return lock, nil
		}

		// If there's a read lock and we want a write lock, we can't do that
		if existing.Type == ReadLock && lockType == WriteLock {
			return nil, fmt.Errorf("path is read-locked by %s", existing.Owner)
		}
	}

	lock := &LockInfo{
		Path:      path,
		Type:      lockType,
		Owner:     owner,
		CreatedAt: time.Now(),
		Timeout:   timeout,
	}
	lm.locks[path] = lock

	return lock, nil
}

// TryAcquireLock attempts to acquire a lock without blocking
func (lm *ExplicitLockManager) TryAcquireLock(path, owner string, lockType LockType, timeout time.Duration) (*LockInfo, bool) {
	lock, err := lm.AcquireLock(path, owner, lockType, timeout)
	return lock, err == nil
}

// WaitForLock waits for a lock to become available
func (lm *ExplicitLockManager) WaitForLock(path string, waitTime time.Duration) bool {
	waiter := make(chan struct{})

	lm.waiterMu.Lock()
	if _, exists := lm.lockWaiters[path]; !exists {
		lm.lockWaiters[path] = make([]chan struct{}, 0)
	}
	lm.lockWaiters[path] = append(lm.lockWaiters[path], waiter)
	lm.waiterMu.Unlock()

	// Wait for release or timeout
	select {
	case <-waiter:
		return true
	case <-time.After(waitTime):
		lm.waiterMu.Lock()
		if waiters, exists := lm.lockWaiters[path]; exists {
			for i, w := range waiters {
				if w == waiter {
					lm.lockWaiters[path] = append(waiters[:i], waiters[i+1:]...)
					break
				}
			}
		}
		lm.waiterMu.Unlock()
		return false
	}
}

// ReleaseLock releases a lock
func (lm *ExplicitLockManager) ReleaseLock(path, owner string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lock, exists := lm.locks[path]
	if !exists {
		return fmt.Errorf("no lock found for path %s", path)
	}

	if lock.Owner != owner {
		return fmt.Errorf("lock is owned by %s, not %s", lock.Owner, owner)
	}

	delete(lm.locks, path)

	lm.waiterMu.Lock()
	defer lm.waiterMu.Unlock()

	if waiters, exists := lm.lockWaiters[path]; exists {
		for _, waiter := range waiters {
			close(waiter)
		}
		delete(lm.lockWaiters, path)
	}

	return nil
}

// CleanupExpiredLocks removes any expired locks
func (lm *ExplicitLockManager) CleanupExpiredLocks() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	for path, lock := range lm.locks {
		if lock.Timeout > 0 && now.Sub(lock.CreatedAt) > lock.Timeout {
			// Lock has expired, remove it
			delete(lm.locks, path)

			lm.waiterMu.Lock()
			if waiters, exists := lm.lockWaiters[path]; exists {
				for _, waiter := range waiters {
					close(waiter)
				}
				delete(lm.lockWaiters, path)
			}
			lm.waiterMu.Unlock()
		}
	}
}

// GetLockInfo returns information about a lock
func (lm *ExplicitLockManager) GetLockInfo(path string) (*LockInfo, bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lock, exists := lm.locks[path]
	return lock, exists
}

// IsLocked checks if a path is locked
func (lm *ExplicitLockManager) IsLocked(path string) bool {
	_, exists := lm.GetLockInfo(path)
	return exists
}

// GetAllLocks returns all current locks
func (lm *ExplicitLockManager) GetAllLocks() []*LockInfo {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	locks := make([]*LockInfo, 0, len(lm.locks))
	for _, lock := range lm.locks {
		locks = append(locks, lock)
	}

	return locks
}

// WithExplicitLocking adds explicit locking capability to the filesystem
func (fs *SimpleFS) WithExplicitLocking() *SimpleFS {
	fs.lockManager = NewExplicitLockManager()
	return fs
}

// LockFile acquires an explicit lock on a file
func (fs *SimpleFS) LockFile(path, owner string, lockType LockType, timeout time.Duration) (*LockInfo, error) {
	if fs.lockManager == nil {
		return nil, fmt.Errorf("explicit locking is not enabled")
	}

	return fs.lockManager.AcquireLock(path, owner, lockType, timeout)
}

// UnlockFile releases an explicit lock on a file
func (fs *SimpleFS) UnlockFile(path, owner string) error {
	if fs.lockManager == nil {
		return fmt.Errorf("explicit locking is not enabled")
	}

	return fs.lockManager.ReleaseLock(path, owner)
}

// IsFileLocked checks if a file has an explicit lock
func (fs *SimpleFS) IsFileLocked(path string) bool {
	if fs.lockManager == nil {
		return false
	}

	return fs.lockManager.IsLocked(path)
}

// GetFileLockInfo gets information about a file's lock
func (fs *SimpleFS) GetFileLockInfo(path string) (*LockInfo, bool) {
	if fs.lockManager == nil {
		return nil, false
	}

	return fs.lockManager.GetLockInfo(path)
}

// WaitForFileLock waits for a file's lock to be released
func (fs *SimpleFS) WaitForFileLock(path string, waitTime time.Duration) bool {
	if fs.lockManager == nil {
		return true
	}

	return fs.lockManager.WaitForLock(path, waitTime)
}
