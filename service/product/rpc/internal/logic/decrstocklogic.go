package logic

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/dtm-labs/dtmcli"
	"github.com/dtm-labs/dtmgrpc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"mall/service/product/rpc/internal/svc"
	"mall/service/product/rpc/product"

	"github.com/zeromicro/go-zero/core/logx"
)

type DecrStockLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecrStockLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecrStockLogic {
	return &DecrStockLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DecrStockLogic) DecrStock(in *product.DecrStockRequest) (*product.DecrStockResponse, error) {
	db, err := sqlx.NewMysql(l.svcCtx.Config.Mysql.DataSource).RawDB()
	if err != nil {
		return nil, status.Error(500, err.Error())
	}
	fmt.Println("[product-rpc]扣减库存")
	// 获取子事务屏障对象
	barrier, err := dtmgrpc.BarrierFromGrpc(l.ctx)
	if err != nil {
		return nil, status.Error(500, err.Error())
	}
	// 开启子事务屏障
	err = barrier.CallWithDB(db, func(tx *sql.Tx) error {
		// 处理具体的逻辑, 库存数量-1
		result, err := l.svcCtx.ProductModel.TxAdjustStock(l.ctx, tx, in.Id, -1)
		if err != nil {
			logx.Errorf("扣除商品数量失败. 错误:%v", err)
			return err
		}
		// 库存扣除失败
		affected, err := result.RowsAffected()
		logx.Infof("扣除商品数量. 影响行数:%d 错误:%v", affected, err)
		if err == nil && affected == 0 {
			return dtmcli.ErrFailure
		}
		return err
	})
	if err == dtmcli.ErrFailure {
		return nil, status.Error(codes.Aborted, dtmcli.ResultFailure)
	}

	if err != nil {
		return nil, err
	}

	return &product.DecrStockResponse{}, nil
}
