schema "public" {
}

table "kaeman_users" {
  schema = schema.public

  column "uid" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()") // PostgreSQL only
  }
  column "nickname" {
    type = text
    null = false
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "deleted_at" {
    type = timestamptz
    null = true
  }

  primary_key {
    columns = [column.uid]
  }
  index "idx_kaeman_users_deleted_at" {
    columns = [column.deleted_at]
  }
}

table "kaeman_identities" {
  schema = schema.public

  column "uid" {
    type = uuid
    null = false
  }
  column "platform" {
    type = text
    null = false
  }
  column "identity" {
    type = text
    null = false
  }
  column "bind_from" {
    type = text
    null = false
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.platform, column.identity]
  }
  index "idx_kaeman_identities_uid" {
    columns = [column.uid]
  }
  foreign_key "fk_kaeman_identities_uid" {
    columns     = [column.uid]
    ref_columns = [table.kaeman_users.column.uid]
  }
}

table "kaeman_channel_subscribe" {
  schema = schema.public

  column "platform" {
    type = text
    null = false
  }

  column "channel_id" {
    type = text
    null = false
  }

  column "events" {
    type    = sql("text[]")
    null    = false
    default = sql("'{}'::text[]")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.platform, column.channel_id]
  }
}
