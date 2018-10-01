package database

import (
	"database/sql"
	"errors"

	"github.com/avalchev94/sqlxt"
	"github.com/lib/pq"
)

// Task represent a single task that a user can add
type Task struct {
	ID       int64   `json:"id"`
	UserID   int64   `json:"user_id" sql:"user_id"`
	Title    string  `json:"title,omitempty"`
	Body     string  `json:"body"`
	Labels   []Label `json:"labels,omitempty" sql:"-"`
	LabelsID []int64 `json:"labels_id,omitempty" sql:"labels"`
}

// OK validates the task fields.
func (t *Task) OK() error {
	if t.UserID <= 0 {
		return errors.New("UserID is incorrect")
	}
	if len(t.Body) == 0 {
		return errors.New("task body can't be empty")
	}
	return nil
}

func (t *Task) add(tx *sql.Tx) error {
	if err := t.OK(); err != nil {
		return err
	}

	row := tx.QueryRow("INSERT INTO tasks (user_id, title, body, labels) VALUES($1,$2,$3,$4) RETURNING id",
		t.UserID, t.Title, t.Body, pq.Array(t.LabelsID))

	return row.Scan(&t.ID)
}

func (t *Task) getLabels(db *sql.DB) error {
	if len(t.LabelsID) == 0 {
		return nil
	}

	rows, err := db.Query("SELECT id, name, color FROM labels WHERE id=ANY($1)", pq.Array(t.LabelsID))
	err = sqlxt.NewScanner(rows, err).Scan(&t.Labels)
	if err == nil {
		// id duplication after successful query
		t.LabelsID = nil
	}
	return err
}

func AddTask(task *Task, schedule *TaskSchedule, db *sql.DB) error {
	transaction, err := db.Begin()
	if err != nil {
		return err
	}

	if err := task.add(transaction); err != nil {
		return transaction.Rollback()
	}

	schedule.TaskID = task.ID
	if err := schedule.add(transaction); err != nil {
		return transaction.Rollback()
	}

	return transaction.Commit()
}

func AddRepetitiveTask(task *Task, repeatance *TaskRepeatance, db *sql.DB) error {
	transaction, err := db.Begin()
	if err != nil {
		return err
	}

	if err := task.add(transaction); err != nil {
		return transaction.Rollback()
	}

	repeatance.TaskID = task.ID
	if err := repeatance.add(transaction); err != nil {
		return transaction.Rollback()
	}

	return transaction.Commit()
}

// GetTask searches for task with the specified id
func GetTask(taskID int64, db *sql.DB) (*Task, error) {
	var t Task

	scanner := sqlxt.NewScanner(db.Query("SELECT * FROM tasks WHERE id=$1", taskID))
	if err := scanner.Scan(&t); err != nil {
		return nil, err
	}

	return &t, t.getLabels(db)
}

// GetTasks returns all the tasks for some user.
func GetTasks(userID int64, db *sql.DB) ([]*Task, error) {
	var tasks []*Task

	rows, err := db.Query("SELECT * FROM tasks WHERE user_id=$1", userID)
	if err := sqlxt.NewScanner(rows, err).Scan(&tasks); err != nil {
		return nil, err
	}

	// get the labels for each task
	for _, task := range tasks {
		if err := task.getLabels(db); err != nil {
			return tasks, err
		}
	}

	return tasks, nil
}
