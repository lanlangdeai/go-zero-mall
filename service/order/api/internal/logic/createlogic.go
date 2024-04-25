package logic

import (
	"context"
	"github.com/dtm-labs/dtmgrpc"
	"google.golang.org/grpc/status"
	"mall/service/order/rpc/types/order"
	"mall/service/product/rpc/types/product"

	"github.com/zeromicro/go-zero/core/logx"
	"mall/service/order/api/internal/svc"
	"mall/service/order/api/internal/types"
)

type CreateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateLogic {
	return &CreateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateLogic) Create(req *types.CreateRequest) (resp *types.CreateResponse, err error) {
	return l.CreateByDtm(req)
	res, err := l.svcCtx.OrderRpc.Create(l.ctx, &order.CreateRequest{
		Uid:    req.Uid,
		Pid:    req.Pid,
		Amount: req.Amount,
		Status: req.Status,
	})
	if err != nil {
		return nil, err
	}

	return &types.CreateResponse{
		Id: res.Id,
	}, nil
}

// 分布式事务DTM
func (l *CreateLogic) CreateByDtm(req *types.CreateRequest) (resp *types.CreateResponse, err error) {
	orderRpcServer, err := l.svcCtx.Config.OrderRpc.BuildTarget()
	if err != nil {
		return nil, status.Error(100, "订单创建异常")
	}

	productRpcServer, err := l.svcCtx.Config.ProductRpc.BuildTarget()
	if err != nil {
		return nil, status.Error(100, "订单创建异常2")
	}

	var dtmServer = "etcd://127.0.0.1:2379/dtmservice"

	gid := dtmgrpc.MustGenGid(dtmServer)
	//创建一个saga协议的事务
	saga := dtmgrpc.NewSagaGrpc(dtmServer, gid).
		Add(productRpcServer+"/product.Product/DecrStock", productRpcServer+"/product.Product/DecrStockRevert", &product.DecrStockRequest{
			Id:  req.Pid,
			Num: 1,
		}).Add(orderRpcServer+"/order.Order/Create", orderRpcServer+"/order.Order/CreateRevert", &order.CreateRequest{
		Uid:    req.Uid,
		Pid:    req.Pid,
		Amount: req.Amount,
		Status: req.Status,
	})
	// 提交事务
	err = saga.Submit()
	if err != nil {
		logx.Errorf("事务执行失败.错误:%v", err)
		return nil, status.Error(500, err.Error())
	}
	return &types.CreateResponse{}, nil
}
