package model

import (
	"context"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
)

type RegStatusesDAO struct {
	*ModelDAO[*RegStatuses]
}

type RegStatuses struct {
	bun.BaseModel `bun:"table:reg_statuses,alias:RSTAT"`

	ModelRecord[*RegStatuses] `bun:"-" json:"-"`

	ID                  int64     `bun:"id,pk,autoincrement" json:"id,omitempty"`
	StatusCode          string    `bun:"f_status_code,unique" json:"f_status_code,omitempty"`
	StatusDescription   *string   `bun:"f_status_description" json:"f_status_description,omitempty"`
	StatusReasonDefault *string   `bun:"f_status_reason_default" json:"f_status_reason_default,omitempty"`
	StatusURL           *string   `bun:"f_status_url" json:"f_status_url,omitempty"`
	CreatedAt           time.Time `bun:"f_created_at" json:"f_created_at,omitzero"`
	Comments            *string   `bun:"f_comments" json:"f_comments,omitempty"`
}

var aliasRegStatuses = map[string]string{
	"id":                  "id",
	"statusCode":          "f_status_code",
	"statusDescription":   "f_status_description",
	"statusReasonDefault": "f_status_reason_default",
	"statusURL":           "f_status_url",
	"createdAt":           "f_created_at",
	"comments":            "f_comments",
}

func (dao *DAO) NewRegStatusesDAO() *RegStatusesDAO {
	return &RegStatusesDAO{
		ModelDAO: NewModelDAO(
			dao,
			aliasRegStatuses,
			func() *RegStatuses {
				return &RegStatuses{}
			}),
	}
}

func (dao *RegStatusesDAO) NewRegStatuses() *RegStatuses {
	record := dao.New()
	return record
}

func (m *RegStatuses) LoadByName(statusCode string) *RegStatuses {

	dao := m.dao
	record := m.self
	record.SetID(0)

	if statusCode == "" {
		return nil
	}

	err := dao.db.NewSelect().
		Model(record).
		Where("f_status_code=?", statusCode).
		Limit(1).
		Scan(context.Background())

	if err != nil {
		slog.Error("1. RegStatuses.LoadByName", "error", err)
		return nil
	}

	return m
}
