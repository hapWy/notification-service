-- +goose Up
-- +goose StatementBegin
create table if not exists notification_logs(
    id uuid primary key default uuid_generate_v4(),
    notification_id uuid not null references notifications(id),
    old_status notification_status,
    new_status notification_status not null,
    created_at timestamp with time zone default now()
);

create index idx_notification_logs on notification_logs(notification_id);
-- +goose StatementEnd 

-- +goose Down
-- +goose StatementBegin
drop table if exists notification_logs;

-- +goose StatementEnd