package demo

import (
	"context"
	"log/slog"
	"time"

	"github.com/raxisau/jackgodb"
	"github.com/uptrace/bun"
)

type StatusesDAO struct {
	*jackgodb.ModelDAO[*Statuses]
}

type Statuses struct {
	bun.BaseModel `bun:"table:reg_statuses,alias:RSTAT"`

	jackgodb.ModelRecord[*Statuses] `bun:"-" json:"-"`

	ID                  int64     `bun:"id,pk,autoincrement" json:"id,omitempty"`
	StatusCode          string    `bun:"f_status_code,unique" json:"f_status_code,omitempty"`
	StatusDescription   *string   `bun:"f_status_description" json:"f_status_description,omitempty"`
	StatusReasonDefault *string   `bun:"f_status_reason_default" json:"f_status_reason_default,omitempty"`
	StatusURL           *string   `bun:"f_status_url" json:"f_status_url,omitempty"`
	CreatedAt           time.Time `bun:"f_created_at" json:"f_created_at,omitzero"`
	Comments            *string   `bun:"f_comments" json:"f_comments,omitempty"`
}

var aliasStatuses = map[string]string{
	"id":                  "id",
	"statusCode":          "f_status_code",
	"statusDescription":   "f_status_description",
	"statusReasonDefault": "f_status_reason_default",
	"statusURL":           "f_status_url",
	"createdAt":           "f_created_at",
	"comments":            "f_comments",
}

func NewStatusesDAO(dao *jackgodb.DAO) *StatusesDAO {
	return &StatusesDAO{
		ModelDAO: jackgodb.NewModelDAO(
			dao,
			aliasStatuses,
			func() *Statuses {
				return &Statuses{}
			}),
	}
}

func (dao *StatusesDAO) NewStatuses() *Statuses {
	record := dao.New()
	return record
}

func (m *Statuses) LoadByName(statusCode string) *Statuses {

	record := m.Self
	record.SetID(0)

	if statusCode == "" {
		return nil
	}

	err := m.Dao.Db.NewSelect().
		Model(record).
		Where("f_status_code=?", statusCode).
		Limit(1).
		Scan(context.Background())

	if err != nil {
		slog.Error("1. Statuses.LoadByName", "error", err)
		return nil
	}

	return m
}
