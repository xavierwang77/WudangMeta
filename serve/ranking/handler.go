package ranking

import (
	"WugongMeta/cmn"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

type Handler interface {
	HandleQueryRankingList(c *gin.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// HandleQueryRankingList 处理查询排行榜请求
func (h *handler) HandleQueryRankingList(c *gin.Context) {
	var req cmn.ReqProto
	err := c.ShouldBindJSON(&req)
	if err != nil {
		z.Error("failed to bind request", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "请求参数错误，请检查是否符合请求协议",
		})
		return
	}

	switch req.Action {
	case "asset.value":
		if req.Page <= 0 {
			req.Page = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		type Filter struct {
			AssetIds []int64 `json:"assetIds,omitempty"`
		}

		var filter Filter
		if req.Filter != nil {
			err = json.Unmarshal(req.Filter, &filter)
			if err != nil {
				z.Error("failed to unmarshal filter", zap.Error(err))
				c.JSON(http.StatusOK, cmn.ReplyProto{
					Status: -1,
					Msg:    "过滤条件解析错误",
				})
				return
			}
		}

		rankingList, rowCount, err := QueryAssetRankingList(c, req.Page, req.PageSize, filter.AssetIds)
		if err != nil {
			z.Error("failed to query asset ranking list", zap.Error(err))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: -1,
				Msg:    "查询排行榜失败",
			})
			return
		}

		rankingListJson, err := json.Marshal(rankingList)
		if err != nil {
			z.Error("failed to marshal ranking list", zap.Error(err))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: -1,
				Msg:    "排行榜数据序列化失败",
			})
			return
		}

		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status:   0,
			Msg:      "查询成功",
			Data:     rankingListJson,
			RowCount: rowCount,
		})
		return
	default:
		z.Error("unknown action", zap.String("action", req.Action))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "未知的操作",
		})
		return
	}
}
