-- +goose Up
-- +goose StatementBegin
create type notification_status as enum (
    'pending','processing', 'sent', 'delivered', 'failed'
);
-- +goose StatementEnd

-- +goose StatementBegin
create table if not exists notifications(
    id uuid primary key default uuid_generate_v4(),
    user_id uuid not null references users(id),
    template_id uuid not null references templates(id),
    status notification_status default 'pending',
    params jsonb default '{}',
    error_message text,
    created_at timestamp with time zone default now(),
    updated_at timestamp with time zone default now()
);

create index idx_notifications_user_id on notifications(user_id);
create index idx_notifications_status on notifications(status);
create index idx_notifications_created_at on notifications(created_at);
create index idx_notifications_params on notifications using gin(params);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop table if exists notifications;
drop type if exists notification_status;

-- +goose StatementEnd