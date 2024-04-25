package logic

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/dtm-labs/dtmgrpc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc/status"
	"mall/service/user/rpc/types/user"

	"mall/service/order/rpc/internal/svc"
	"mall/service/order/rpc/order"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateRevertLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateRevertLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateRevertLogic {
	return &CreateRevertLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建订单-回滚操作
func (l *CreateRevertLogic) CreateRevert(in *order.CreateRequest) (*order.CreateResponse, error) {
	db, err := sqlx.NewMysql(l.svcCtx.Config.Mysql.DataSource).RawDB()
	if err != nil {
		return nil, status.Error(500, err.Error())
	}

	fmt.Println("[order-rpc]创建订单回滚操作")
	// 获取子事务屏障对象
	barrier, err := dtmgrpc.BarrierFromGrpc(l.ctx)
	if err != nil {
		return nil, status.Error(500, err.Error())
	}

	// 开启子事务屏障
	if err := barrier.CallWithDB(db, func(tx *sql.Tx) error {
		// 用户是否存在
		_, err := l.svcCtx.UserRpc.UserInfo(l.ctx, &user.UserInfoRequest{
			Id: in.Uid,
		})
		if err != nil {
			return fmt.Errorf("用户不存在")
		}
		// 订单是否存在
		orderInfo, err := l.svcCtx.OrderModel.FindOneByUid(l.ctx, in.Uid)
		if err != nil {
			return fmt.Errorf("订单不存在")
		}
		// 修改订单状态为9,标识该订单已失效并更新
		orderInfo.Status = 9
		err = l.svcCtx.OrderModel.TxUpdate(l.ctx, tx, orderInfo)
		if err != nil {
			return fmt.Errorf("订单更新失败")
		}

		return nil
	}); err != nil {
		logx.Errorf("创建订单回滚失败: %v", err)
		return nil, status.Error(500, err.Error())
	}

	return &order.CreateResponse{}, nil
}
