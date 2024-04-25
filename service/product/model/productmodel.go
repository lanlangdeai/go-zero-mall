package model

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ProductModel = (*customProductModel)(nil)

type (
	// ProductModel is an interface to be customized, add more methods here,
	// and implement the added methods in customProductModel.
	ProductModel interface {
		productModel

		TxAdjustStock(ctx context.Context, tx *sql.Tx, id int64, delta int) (sql.Result, error)
	}

	customProductModel struct {
		*defaultProductModel
	}
)

// NewProductModel returns a model for the database table.
func NewProductModel(conn sqlx.SqlConn, c cache.CacheConf) ProductModel {
	return &customProductModel{
		defaultProductModel: newProductModel(conn, c),
	}
}

func (m *defaultProductModel) TxAdjustStock(ctx context.Context, tx *sql.Tx, id int64, delta int) (sql.Result, error) {
	productIdKey := fmt.Sprintf("%s%v", cacheProductIdPrefix, id)
	return m.Exec(func(conn sqlx.SqlConn) (sql.Result, error) {
		//query := fmt.Sprintf("update %s set stock=stock+? where stock >= -? and id=?", m.table)
		query := fmt.Sprintf("update %s set stock=stock+? where  id=?", m.table)
		fmt.Printf("[商品]库存变更. sql:%s, id:%d, 变更额度:%d\n", query, id, delta)
		return tx.ExecContext(ctx, query, delta, id)
	}, productIdKey)
}
