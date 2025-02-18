// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mvcc

import (
	"context"
	"sync"
	"sync/atomic"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	"go.etcd.io/etcd/server/v3/lease"
	"go.etcd.io/etcd/server/v3/storage/backend"
)

type RangeOptions struct {
	Limit int64
	Rev   int64
	Count bool
}

type RangeResult struct {
	KVs   []mvccpb.KeyValue
	Rev   int64
	Count int
}

type ReadView interface {
	// FirstRev returns the first KV revision at the time of opening the txn.
	// After a compaction, the first revision increases to the compaction
	// revision.
	FirstRev() int64

	// Rev returns the revision of the KV at the time of opening the txn.
	Rev() int64

	// Range gets the keys in the range at rangeRev.
	// The returned rev is the current revision of the KV when the operation is executed.
	// If rangeRev <=0, range gets the keys at currentRev.
	// If `end` is nil, the request returns the key.
	// If `end` is not nil and not empty, it gets the keys in range [key, range_end).
	// If `end` is not nil and empty, it gets the keys greater than or equal to key.
	// Limit limits the number of keys returned.
	// If the required rev is compacted, ErrCompacted will be returned.
	Range(ctx context.Context, key, end []byte, ro RangeOptions) (r *RangeResult, err error)
}

// TxnRead represents a read-only transaction with operations that will not
// block other read transactions.
type TxnRead interface {
	ReadView
	// End marks the transaction is complete and ready to commit.
	End()
}

type WriteView interface {
	// DeleteRange deletes the given range from the store.
	// A deleteRange increases the rev of the store if any key in the range exists.
	// The number of key deleted will be returned.
	// The returned rev is the current revision of the KV when the operation is executed.
	// It also generates one event for each key delete in the event history.
	// if the `end` is nil, deleteRange deletes the key.
	// if the `end` is not nil, deleteRange deletes the keys in range [key, range_end).
	DeleteRange(key, end []byte) (n, rev int64)

	// Put puts the given key, value into the store. Put also takes additional argument lease to
	// attach a lease to a key-value pair as meta-data. KV implementation does not validate the lease
	// id.
	// A put also increases the rev of the store, and generates one event in the event history.
	// The returned rev is the current revision of the KV when the operation is executed.
	Put(key, value []byte, lease lease.LeaseID) (rev int64)
}

// TxnWrite represents a transaction that can modify the store.
type TxnWrite interface {
	TxnRead
	WriteView
	// Changes gets the changes made since opening the write txn.
	Changes() []mvccpb.KeyValue
}

// txnReadWrite coerces a read txn to a write, panicking on any write operation.
type txnReadWrite struct{ TxnRead }

func (trw *txnReadWrite) DeleteRange(key, end []byte) (n, rev int64) { panic("unexpected DeleteRange") }
func (trw *txnReadWrite) Put(key, value []byte, lease lease.LeaseID) (rev int64) {
	panic("unexpected Put")
}
func (trw *txnReadWrite) Changes() []mvccpb.KeyValue { return nil }

func NewReadOnlyTxnWrite(txn TxnRead) TxnWrite { return &txnReadWrite{txn} }

type ReadTxMode uint32

const (
	// Use ConcurrentReadTx and the txReadBuffer is copied
	ConcurrentReadTxMode = ReadTxMode(1)
	// Use backend ReadTx and txReadBuffer is not copied
	SharedBufReadTxMode = ReadTxMode(2)
)

type KV interface {
	ReadView
	WriteView

	// Read creates a read transaction.
	Read(mode ReadTxMode, trace *traceutil.Trace) TxnRead

	// Write creates a write transaction.
	Write(trace *traceutil.Trace) TxnWrite

	// HashStorage returns HashStorage interface for KV storage.
	HashStorage() HashStorage

	// Compact frees all superseded keys with revisions less than rev.
	Compact(trace *traceutil.Trace, rev int64) (<-chan struct{}, error)

	// Commit commits outstanding txns into the underlying backend.
	Commit()

	// Restore restores the KV store from a backend.
	Restore(b backend.Backend) error
	Close() error

	Index() index
}

// WatchableKV is a KV that can be watched.
type WatchableKV interface {
	KV
	Watchable
}

// Watchable is the interface that wraps the NewWatchStream function.
type Watchable interface {
	// NewWatchStream returns a WatchStream that can be used to
	// watch events happened or happening on the KV.
	NewWatchStream() WatchStream
}

type KeyValueStore struct {
	value []byte
	key   string
}

var storeIndex = int64(0)
var indexStore = sync.Map{}
var memoryStore = sync.Map{}

type MemoryKV struct {
}

func (kv *MemoryKV) FirstRev() int64 {
	return 1
}

func (kv *MemoryKV) Rev() int64 {
	return 1
}

func (kv *MemoryKV) Range(ctx context.Context, key, end []byte, ro RangeOptions) (*RangeResult, error) {
	var result RangeResult
	result.Rev = 1
	startValue, startExists := indexStore.Load(string(key))
	endValue, endExists := indexStore.Load(string(end))
	if endExists && startExists {
		startIndex, ok1 := startValue.(int64)
		endIndex, ok2 := endValue.(int64)
		if !ok1 || !ok2 {
			return &result, nil
		}

		for i := startIndex; i <= endIndex; i++ {
			value, ok := memoryStore.Load(i)
			if ok {
				result.KVs = append(result.KVs, mvccpb.KeyValue{Key: []byte(value.(KeyValueStore).key), Value: value.(KeyValueStore).value})
			}
		}
	}

	return &result, nil
}

func (kv *MemoryKV) DeleteRange(key, end []byte) (n, rev int64) {
	deleted := int64(0)
	startValue, startExists := indexStore.Load(string(key))
	endValue, endExists := indexStore.Load(string(end))
	if endExists && startExists {
		startIndex, ok1 := startValue.(int64)
		endIndex, ok2 := endValue.(int64)
		if !ok1 || !ok2 {
			return deleted, 1
		}

		for i := startIndex; i <= endIndex; i++ {
			value, ok := memoryStore.LoadAndDelete(i)
			if ok {
				indexStore.Delete(value.(KeyValueStore).key)
				deleted++
			}
		}
	}

	return deleted, 1
}

func (kv *MemoryKV) Put(key, value []byte, lease lease.LeaseID) (rev int64) {
	nextIndex := atomic.AddInt64(&storeIndex, 1)
	memoryStore.Store(nextIndex, KeyValueStore{value, string(key)})
	indexStore.Store(string(key), nextIndex)
	return 1
}

func (kv *MemoryKV) Read(mode ReadTxMode, trace *traceutil.Trace) TxnRead {
	return &ReadTransaction{kv: kv}
}

func (kv *MemoryKV) Write(trace *traceutil.Trace) TxnWrite {
	return &WriteTransaction{kv: kv}
}

func (kv *MemoryKV) HashStorage() HashStorage {
	return &Hasher{}
}

func (kv *MemoryKV) Index() index {
	panic("index!")
}

func (kv *MemoryKV) NewWatchStream() WatchStream {
	panic("watch stream!")
}

func (kv *MemoryKV) Compact(trace *traceutil.Trace, rev int64) (<-chan struct{}, error) {
	done := make(chan struct{})
	close(done)
	return done, nil
}

func (kv *MemoryKV) Commit() {
}

func (kv *MemoryKV) Restore(b backend.Backend) error {
	return nil
}

func (kv *MemoryKV) Close() error {
	return nil
}

type Hasher struct {
}

func (h *Hasher) Hash() (hash uint32, revision int64, err error) {
	panic("implement hash")
}

func (h *Hasher) HashByRev(rev int64) (hash KeyValueHash, currentRev int64, err error) {
	panic("implement hashByRev")
}

func (h *Hasher) Store(valueHash KeyValueHash) {
	panic("implement store")
}

func (h *Hasher) Hashes() []KeyValueHash {
	panic("implement hashes")
}

type WriteTransaction struct {
	kv *MemoryKV
}

func (w *WriteTransaction) FirstRev() int64 {
	return w.kv.FirstRev()
}

func (w *WriteTransaction) Rev() int64 {
	return w.kv.Rev()
}

func (w *WriteTransaction) Range(ctx context.Context, key, end []byte, ro RangeOptions) (r *RangeResult, err error) {
	return w.kv.Range(ctx, key, end, ro)
}

func (w *WriteTransaction) End() {

}

func (w *WriteTransaction) DeleteRange(key, end []byte) (n, rev int64) {
	return w.kv.DeleteRange(key, end)
}

func (w *WriteTransaction) Put(key, value []byte, lease lease.LeaseID) (rev int64) {
	return w.kv.Put(key, value, lease)
}

func (w *WriteTransaction) Changes() []mvccpb.KeyValue {
	//TODO implement me
	panic("implement me")
}

type ReadTransaction struct {
	kv *MemoryKV
}

func (read *ReadTransaction) FirstRev() int64 {
	return read.kv.FirstRev()
}

func (read *ReadTransaction) Rev() int64 {
	return read.kv.Rev()
}

func (read *ReadTransaction) Range(ctx context.Context, key, end []byte, ro RangeOptions) (r *RangeResult, err error) {
	return read.kv.Range(ctx, key, end, ro)
}

func (read *ReadTransaction) End() {

}
