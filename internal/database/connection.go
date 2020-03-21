package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

type operatorErr struct {
}

func newOperatorErr() operatorErr {
	return operatorErr{}
}

func (e operatorErr) Wrap(err error) error {
	return fmt.Errorf("couldn't handle db operation because of %w", err)
}

type ConnectionService struct {
	DbConn *pgxpool.Pool
}

func NewConnectionService(config *Config) (*ConnectionService, error) {
	connectionString := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		config.Host, config.Port, config.DB, config.User, config.Pass)
	c, err := pgxpool.Connect(context.Background(), connectionString)
	if err != nil {
		return nil, err
	}
	return &ConnectionService{
		DbConn:             c,
	}, nil
}

func (cs *ConnectionService) WrapIntoTransaction(ctx context.Context, f func(tx pgx.Tx) error) error {
	trans, err := cs.DbConn.Begin(ctx)
	if err != nil {
		return newOperatorErr().Wrap(err)
	}
	if err := f(trans); err != nil {
		if err := trans.Rollback(ctx); err != nil {
			log.Error(newOperatorErr().Wrap(err))
		}
		return newOperatorErr().Wrap(err)
	}
	if err := trans.Commit(ctx); err != nil {
		return newOperatorErr().Wrap(err)
	}
	return nil
}

