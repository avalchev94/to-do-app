package database

import (
	"database/sql"

	"github.com/avalchev94/sqlxt"
)

type ScheduledTask struct {
	Task     Task         `json:"task"`
	Schedule TaskSchedule `json:"schedule"`
}

func (t *ScheduledTask) OK() error {
	if err := t.Task.OK(); err != nil {
		return err
	}
	if err := t.Schedule.OK(); err != nil {
		return err
	}
	return nil
}

func (t *ScheduledTask) Add(db *sql.DB) error {
	transaction, err := db.Begin()
	if err != nil {
		return err
	}

	if err := t.Task.add(transaction); err != nil {
		if terr := transaction.Rollback(); terr != nil {
			return terr
		}
		return err
	}

	if err := t.Task.Labels.add(t.Task.UserID, transaction); err != nil {
		if terr := transaction.Rollback(); terr != nil {
			return terr
		}
		return err
	}

	t.Schedule.TaskID = t.Task.ID
	if err := t.Schedule.add(transaction); err != nil {
		if terr := transaction.Rollback(); terr != nil {
			return terr
		}
		return transaction.Rollback()
	}

	return transaction.Commit()
}

func (t *ScheduledTask) AddInTx(transaction *sql.Tx) error {
	if err := t.Task.add(transaction); err != nil {
		return err
	}

	t.Schedule.TaskID = t.Task.ID
	if err := t.Schedule.add(transaction); err != nil {
		return err
	}
	return nil
}

func GetScheduledTask(taskID int64, db *sql.DB) (*ScheduledTask, error) {
	query := `SELECT t.*, s.type, s.date, s.time, s.created, s.finished
						FROM tasks AS t
						JOIN task_schedule AS s ON t.id = s.task_id
						WHERE t.id = $1`

	var task ScheduledTask
	if err := sqlxt.NewScanner(db.Query(query, taskID)).Scan(&task); err != nil {
		return nil, err
	}

	return &task, nil
}

func GetScheduledTasks(userID int64, params Parameters, db *sql.DB) ([]*ScheduledTask, error) {
	query := `SELECT t.*, s.type, s.date, s.time, s.created, s.finished
	FROM tasks AS t
	JOIN task_schedule AS s ON t.id = s.task_id
	WHERE t.user_id = $1`

	if params != nil {
		if where := params.encodeSQL(map[string]string{
			"task_schedule": "s",
			"tasks":         "t",
		}); where != "" {
			query += " AND " + where
		}
	}

	var tasks []*ScheduledTask
	if err := sqlxt.NewScanner(db.Query(query, userID)).Scan(&tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}
