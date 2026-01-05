package replication

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"

	"github.com/ivikasavnish/postgres-test-replay/pkg/config"
	"github.com/ivikasavnish/postgres-test-replay/pkg/wal"
)

type Listener struct {
	config      *config.Config
	conn        *pgconn.PgConn
	walWriter   *wal.LogWriter
	slotName    string
	publication string
}

func NewListener(cfg *config.Config, walWriter *wal.LogWriter) *Listener {
	return &Listener{
		config:      cfg,
		walWriter:   walWriter,
		slotName:    cfg.Replication.SlotName,
		publication: cfg.Replication.PublicationName,
	}
}

func (l *Listener) Connect(ctx context.Context) error {
	dbConfig := l.config.PrimaryDB
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?replication=database",
		dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Database)

	conn, err := pgconn.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	l.conn = conn
	return nil
}

func (l *Listener) CreateReplicationSlot(ctx context.Context) error {
	result := l.conn.Exec(ctx, fmt.Sprintf("SELECT * FROM pg_create_logical_replication_slot('%s', 'pgoutput')", l.slotName))
	_, err := result.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to create replication slot: %w", err)
	}
	return nil
}

func (l *Listener) Start(ctx context.Context) error {
	pluginArguments := []string{
		"proto_version '1'",
		fmt.Sprintf("publication_names '%s'", l.publication),
	}

	err := pglogrepl.StartReplication(ctx, l.conn, l.slotName, 0, pglogrepl.StartReplicationOptions{
		PluginArgs: pluginArguments,
	})
	if err != nil {
		return fmt.Errorf("failed to start replication: %w", err)
	}

	return l.receiveMessages(ctx)
}

func (l *Listener) receiveMessages(ctx context.Context) error {
	clientXLogPos := pglogrepl.LSN(0)
	standbyMessageTimeout := time.Second * 10
	nextStandbyMessageDeadline := time.Now().Add(standbyMessageTimeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Now().After(nextStandbyMessageDeadline) {
			err := pglogrepl.SendStandbyStatusUpdate(ctx, l.conn, pglogrepl.StandbyStatusUpdate{
				WALWritePosition: clientXLogPos,
			})
			if err != nil {
				return fmt.Errorf("failed to send standby status: %w", err)
			}
			nextStandbyMessageDeadline = time.Now().Add(standbyMessageTimeout)
		}

		ctx2, cancel := context.WithTimeout(ctx, standbyMessageTimeout)
		rawMsg, err := l.conn.ReceiveMessage(ctx2)
		cancel()

		if err != nil {
			if pgconn.Timeout(err) {
				continue
			}
			return fmt.Errorf("receive message failed: %w", err)
		}

		if errMsg, ok := rawMsg.(*pgproto3.ErrorResponse); ok {
			return fmt.Errorf("received error: %+v", errMsg)
		}

		msg, ok := rawMsg.(*pgproto3.CopyData)
		if !ok {
			continue
		}

		switch msg.Data[0] {
		case pglogrepl.PrimaryKeepaliveMessageByteID:
			pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
			if err != nil {
				return fmt.Errorf("parse keepalive failed: %w", err)
			}

			if pkm.ReplyRequested {
				nextStandbyMessageDeadline = time.Time{}
			}

		case pglogrepl.XLogDataByteID:
			xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
			if err != nil {
				return fmt.Errorf("parse xlog data failed: %w", err)
			}

			if err := l.processWALData(xld); err != nil {
				return fmt.Errorf("process WAL data failed: %w", err)
			}

			clientXLogPos = xld.WALStart + pglogrepl.LSN(len(xld.WALData))
		}
	}
}

func (l *Listener) processWALData(xld pglogrepl.XLogData) error {
	logicalMsg, err := pglogrepl.Parse(xld.WALData)
	if err != nil {
		return fmt.Errorf("parse logical message failed: %w", err)
	}

	switch msg := logicalMsg.(type) {
	case *pglogrepl.RelationMessage:
		// Handle relation messages
	case *pglogrepl.InsertMessage:
		return l.handleInsert(msg, xld.WALStart)
	case *pglogrepl.UpdateMessage:
		return l.handleUpdate(msg, xld.WALStart)
	case *pglogrepl.DeleteMessage:
		return l.handleDelete(msg, xld.WALStart)
	}

	return nil
}

func (l *Listener) handleInsert(msg *pglogrepl.InsertMessage, lsn pglogrepl.LSN) error {
	entry := &wal.WALEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		LSN:       lsn.String(),
		Operation: wal.OpInsert,
		Data:      l.tupleToMap(msg.Tuple),
	}

	return l.walWriter.WriteEntry(entry)
}

func (l *Listener) handleUpdate(msg *pglogrepl.UpdateMessage, lsn pglogrepl.LSN) error {
	entry := &wal.WALEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		LSN:       lsn.String(),
		Operation: wal.OpUpdate,
		Data:      l.tupleToMap(msg.NewTuple),
	}

	if msg.OldTuple != nil {
		entry.OldData = l.tupleToMap(msg.OldTuple)
	}

	return l.walWriter.WriteEntry(entry)
}

func (l *Listener) handleDelete(msg *pglogrepl.DeleteMessage, lsn pglogrepl.LSN) error {
	entry := &wal.WALEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		LSN:       lsn.String(),
		Operation: wal.OpDelete,
		OldData:   l.tupleToMap(msg.OldTuple),
	}

	return l.walWriter.WriteEntry(entry)
}

func (l *Listener) tupleToMap(tuple *pglogrepl.TupleData) map[string]interface{} {
	result := make(map[string]interface{})

	// Note: Using generic column names here as relation metadata is not readily available
	// in the current message context. For production use, consider maintaining a cache
	// of relation ID -> column metadata mapping from RelationMessage events.
	for i, col := range tuple.Columns {
		key := fmt.Sprintf("col_%d", i)

		switch col.DataType {
		case 'n':
			result[key] = nil
		case 't':
			result[key] = string(col.Data)
		}
	}

	return result
}

func (l *Listener) Close() error {
	if l.conn != nil {
		return l.conn.Close(context.Background())
	}
	return nil
}
