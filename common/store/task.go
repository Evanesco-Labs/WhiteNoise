package store

import (
	"fmt"
	"time"
)

type Task struct {
	Id         string
	FileHash   string
	FileName   string
	WalletAddr string
	Type       uint64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (this *SQLiteStorage) InsertTask(id, fileHash, fileName, walletAddr string, taskType uint64) (bool, error) {
	sql := fmt.Sprintf("INSERT OR REPLACE INTO %s (id, fileHash, fileName, walletAddr, type , createdAt, updatedAt) VALUES(?, ?, ?, ?, ?, ?, ?)", DSP_TASK_TABLE_NAME)
	return this.Exec(sql, id, fileHash, fileName, walletAddr, taskType, time.Now(), time.Now())
}

func (this *SQLiteStorage) UpdateTaskId(id, fileHash, fileName, walletAddr string, taskType uint64) (bool, error) {
	sql := fmt.Sprintf("UPDATE %s SET id = ?, updatedAt = ?  WHERE  fileHash = ? and walletAddr = ? and type = ? ", DSP_TASK_TABLE_NAME)
	return this.Exec(sql, id, time.Now(), fileHash, walletAddr, taskType)
}
