package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// SettingRevision 保存一次系统设置变更前的全量键值快照，用于版本历史与一键回滚。
//
// 删除策略：硬删除；由记录路径裁剪保留最近 N 条。
type SettingRevision struct {
	ent.Schema
}

func (SettingRevision) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "setting_revisions"},
	}
}

func (SettingRevision) Fields() []ent.Field {
	return []ent.Field{
		field.Text("snapshot").
			SchemaType(map[string]string{
				dialect.Postgres: "text",
			}),
		field.Int64("changed_by").
			Optional().
			Default(0),
		field.String("note").
			MaxLen(255).
			Optional().
			Default(""),
		field.Time("created_at").
			Default(time.Now).
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}
