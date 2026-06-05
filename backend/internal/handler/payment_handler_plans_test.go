//go:build unit

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestGetPlansIncludesIncludedGroupIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite", "file:payment_handler_get_plans?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	codexGroup, err := client.Group.Create().
		SetName("Codex").
		SetPlatform("openai").
		SetSubscriptionType("subscription").
		Save(ctx)
	require.NoError(t, err)

	claudeGroup, err := client.Group.Create().
		SetName("Claude").
		SetPlatform("anthropic").
		SetSubscriptionType("subscription").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetGroupID(codexGroup.ID).
		SetIncludedGroupIds([]int64{claudeGroup.ID}).
		SetName("Codex 专业版").
		SetDescription("Codex plus Claude route").
		SetPrice(199).
		SetValidityDays(30).
		SetValidityUnit("day").
		SetProductName("Codex 专业版").
		SetForSale(true).
		SetSortOrder(1).
		Save(ctx)
	require.NoError(t, err)

	configSvc := service.NewPaymentConfigService(client, nil, []byte("0123456789abcdef0123456789abcdef"))
	h := NewPaymentHandler(nil, configSvc, nil)

	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/plans", nil)

	h.GetPlans(ginCtx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int `json:"code"`
		Data []struct {
			ID               int64   `json:"id"`
			GroupID          int64   `json:"group_id"`
			IncludedGroupIDs []int64 `json:"included_group_ids"`
			GroupPlatform    string  `json:"group_platform"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data, 1)
	require.Equal(t, codexGroup.ID, resp.Data[0].GroupID)
	require.Equal(t, "openai", resp.Data[0].GroupPlatform)
	require.Equal(t, []int64{claudeGroup.ID}, resp.Data[0].IncludedGroupIDs)
}
