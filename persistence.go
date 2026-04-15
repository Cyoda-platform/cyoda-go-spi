package spi

import (
	"context"
	"io"
	"iter"
	"time"

)

type StoreFactory interface {
	EntityStore(ctx context.Context) (EntityStore, error)
	ModelStore(ctx context.Context) (ModelStore, error)
	KeyValueStore(ctx context.Context) (KeyValueStore, error)
	MessageStore(ctx context.Context) (MessageStore, error)
	WorkflowStore(ctx context.Context) (WorkflowStore, error)
	StateMachineAuditStore(ctx context.Context) (StateMachineAuditStore, error)
	AsyncSearchStore(ctx context.Context) (AsyncSearchStore, error)
	TransactionManager(ctx context.Context) (TransactionManager, error)
	Close() error
}

type EntityStore interface {
	Save(ctx context.Context, entity *Entity) (int64, error)
	// CompareAndSave saves the entity only if the current latest transaction ID matches expectedTxID.
	// Returns ErrConflict if the transaction ID has changed.
	CompareAndSave(ctx context.Context, entity *Entity, expectedTxID string) (int64, error)
	// SaveAll saves multiple entities, returning versions in iteration order.
	// Backends may execute saves concurrently. On error, returns the first
	// error encountered; partially-saved entities within an uncommitted
	// transaction are invisible to readers.
	SaveAll(ctx context.Context, entities iter.Seq[*Entity]) ([]int64, error)
	Get(ctx context.Context, entityID string) (*Entity, error)
	GetAsAt(ctx context.Context, entityID string, asAt time.Time) (*Entity, error)
	GetAll(ctx context.Context, modelRef ModelRef) ([]*Entity, error)
	GetAllAsAt(ctx context.Context, modelRef ModelRef, asAt time.Time) ([]*Entity, error)
	Delete(ctx context.Context, entityID string) error
	DeleteAll(ctx context.Context, modelRef ModelRef) error
	Exists(ctx context.Context, entityID string) (bool, error)
	Count(ctx context.Context, modelRef ModelRef) (int64, error)
	GetVersionHistory(ctx context.Context, entityID string) ([]EntityVersion, error)
}

type ModelStore interface {
	Save(ctx context.Context, desc *ModelDescriptor) error
	Get(ctx context.Context, modelRef ModelRef) (*ModelDescriptor, error)
	GetAll(ctx context.Context) ([]ModelRef, error)
	Delete(ctx context.Context, modelRef ModelRef) error
	Lock(ctx context.Context, modelRef ModelRef) error
	Unlock(ctx context.Context, modelRef ModelRef) error
	IsLocked(ctx context.Context, modelRef ModelRef) (bool, error)
	SetChangeLevel(ctx context.Context, modelRef ModelRef, level ChangeLevel) error
}

type KeyValueStore interface {
	Put(ctx context.Context, namespace string, key string, value []byte) error
	Get(ctx context.Context, namespace string, key string) ([]byte, error)
	Delete(ctx context.Context, namespace string, key string) error
	List(ctx context.Context, namespace string) (map[string][]byte, error)
}

type MessageStore interface {
	Save(ctx context.Context, id string, header MessageHeader, metaData MessageMetaData, payload io.Reader) error
	Get(ctx context.Context, id string) (MessageHeader, MessageMetaData, io.ReadCloser, error)
	Delete(ctx context.Context, id string) error
	DeleteBatch(ctx context.Context, ids []string) error
}

type WorkflowStore interface {
	Save(ctx context.Context, modelRef ModelRef, workflows []WorkflowDefinition) error
	Get(ctx context.Context, modelRef ModelRef) ([]WorkflowDefinition, error)
	Delete(ctx context.Context, modelRef ModelRef) error
}

type StateMachineAuditStore interface {
	Record(ctx context.Context, entityID string, event StateMachineEvent) error
	GetEvents(ctx context.Context, entityID string) ([]StateMachineEvent, error)
	GetEventsByTransaction(ctx context.Context, entityID string, transactionID string) ([]StateMachineEvent, error)
}
