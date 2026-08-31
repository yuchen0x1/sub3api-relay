package schema

import (
	"github.com/yuchen0x1/sub3api-relay/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// StaticProxy models an account-owned assignment of an existing proxy.
//
// The assignment capacity is deliberately fixed at ten. Runtime concurrency
// accounting remains owned by the gateway/concurrency domain and is not stored
// on this entity.
type StaticProxy struct {
	ent.Schema
}

func (StaticProxy) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_static_proxy_exits"},
	}
}

func (StaticProxy) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (StaticProxy) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		field.Int64("proxy_id"),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.Int("position").
			Default(0),
		field.Bool("enabled").
			Default(true),
		field.Int("capacity").
			Default(10).
			Comment("Fixed assignment capacity; must remain 10."),
		field.String("health_status").
			MaxLen(20).
			Default("unknown"),
		field.Int("failure_count").
			Default(0),
		field.Time("last_failure_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("last_failure_reason").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("last_checked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (StaticProxy) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("account", Account.Type).
			Unique().
			Required().
			Field("account_id"),
		edge.To("proxy", Proxy.Type).
			Unique().
			Required().
			Field("proxy_id"),
	}
}

func (StaticProxy) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id"),
		index.Fields("proxy_id"),
		index.Fields("account_id", "enabled"),
		index.Fields("account_id", "position"),
		index.Fields("health_status"),
		index.Fields("deleted_at"),
		// The active-only uniqueness rule is created as a partial index by the
		// forward migration because Ent indexes do not carry that predicate here.
		index.Fields("account_id", "proxy_id"),
	}
}
