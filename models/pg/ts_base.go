package pg

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type TSBase struct {
	Base
	clusterID int64
}

// DeleteDeleted deletes all "deleted" records.
func (ts *TSBase) DeleteDeleted(tx *sqlx.Tx, clusterID int64) error {
	if ts.table == "" {
		return errors.New("Table must not be empty.")
	}

	tx, wrapInSingleTransaction, err := ts.newTransactionIfNeeded(tx)
	if err != nil {
		return err
	}
	if tx == nil {
		return errors.New("Transaction struct must not be empty.")
	}

	now := time.Now().UTC().Unix()
	query := fmt.Sprintf("DELETE FROM %v WHERE cluster_id=$1 AND deleted < to_timestamp($2) at time zone 'utc'", ts.table)

	_, err = tx.Exec(query, clusterID, now)

	if wrapInSingleTransaction == true {
		err = tx.Commit()
	}

	return err
}
