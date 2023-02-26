package employee

import (
	"context"

	"PowerX/internal/svc"
	"PowerX/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteEmployeeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteEmployeeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteEmployeeLogic {
	return &DeleteEmployeeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteEmployeeLogic) DeleteEmployee(req *types.DeleteEmployeeRequest) (resp *types.DeleteEmployeeReply, err error) {
	l.svcCtx.UC.Employee.DeleteEmployee(l.ctx, req.Id)
	return &types.DeleteEmployeeReply{
		Id: req.Id,
	}, nil
}
