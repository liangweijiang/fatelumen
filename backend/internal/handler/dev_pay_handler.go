package handler

import (
	"context"
	"net/http"
	"strconv"

	"fatelumen/backend/internal/payment"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type devPaymentSvc interface {
	HandleWebhook(ctx context.Context, provider string, payload []byte, sigHeader string) error
}

// DevPayHandler 仅 dev 环境注册：提供本地假收银台与一键完成支付，走标准 webhook 履约链路。
type DevPayHandler struct {
	svc  devPaymentSvc
	mock *payment.MockProviderRef
}

// NewDevPayHandler 创建 dev 假收银台处理器。
func NewDevPayHandler(svc devPaymentSvc, mock *payment.MockProviderRef) *DevPayHandler {
	return &DevPayHandler{svc: svc, mock: mock}
}

// Page GET /api/v1/dev/pay/:id —— 极简本地收银台页。
func (h *DevPayHandler) Page(c *gin.Context) {
	id := c.Param("id")
	html := `<!doctype html><html><head><meta charset="utf-8"><title>完成支付</title>
<style>body{font-family:sans-serif;background:#efe6d6;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
.card{background:#fff;padding:40px 48px;border-radius:16px;box-shadow:0 10px 40px rgba(0,0,0,.12);text-align:center}
h1{font-size:20px;color:#4a3f2a;margin:0 0 8px}p{color:#8a7d63;margin:0 0 24px}
button{background:#b89048;color:#fff;border:0;padding:14px 36px;border-radius:10px;font-size:16px;cursor:pointer}
.ok{color:#5c7060;font-weight:600;margin-top:18px}</style></head>
<body><div class="card"><h1>本地收银台（开发用）</h1><p>订单 #` + id + `</p>
<button onclick="pay()">确认完成支付</button><div id="r" class="ok"></div></div>
<script>function pay(){fetch(location.pathname+'/complete',{method:'POST'}).then(r=>r.json()).then(d=>{document.getElementById('r').textContent=d.code===0?'支付成功，报告已解锁，可返回查看':'失败: '+(d.msg||'')})}</script>
</body></html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// Complete POST /api/v1/dev/pay/:id/complete —— 构造已签名回调并走标准履约链路解锁报告。
func (h *DevPayHandler) Complete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid order id")
		return
	}
	payload, sig, err := h.mock.BuildCompletedEvent(id)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("dev pay build event failed",
			"err", err,
			"order_id", id,
		)
		response.Error(c, "build event failed")
		return
	}
	if err := h.svc.HandleWebhook(c.Request.Context(), "mock", payload, sig); err != nil {
		logger.FromCtx(c.Request.Context()).Error("dev pay handle webhook failed",
			"err", err,
			"order_id", id,
		)
		response.Error(c, "fulfill failed")
		return
	}
	response.OK(c, gin.H{"order_id": id, "status": "paid"})
}
