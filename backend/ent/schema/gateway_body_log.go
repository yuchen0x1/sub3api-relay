package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GatewayBodyLog stores recorded gateway request bodies for debugging and
// replay. Recording is opt-in (setting gateway_body_log_enabled, default off)
// and size-capped by the writer.
//
// 删除策略：硬删除。请求体日志是临时调试数据，可整体清理，无审计保留需求。
type GatewayBodyLog struct {
	ent.Schema
}

func (GatewayBodyLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "gateway_body_logs"},
	}
}

func (GatewayBodyLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("request_id").
			MaxLen(64).
			NotEmpty(),
		field.String("method").
			MaxLen(10).
			Default("POST"),
		field.String("path").
			MaxLen(255).
			NotEmpty(),
		field.Int64("api_key_id").
			Optional().
			Default(0),
		field.Int64("user_id").
			Optional().
			Default(0),
		field.String("model").
			MaxLen(255).
			Optional().
			Default(""),
		field.Text("body").
			SchemaType(map[string]string{
				dialect.Postgres: "text",
			}),
		field.Time("created_at").
			Default(time.Now).
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}

func (GatewayBodyLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id").Unique(),
		index.Fields("api_key_id"),
		index.Fields("user_id"),
		index.Fields("created_at"),
	}
}
