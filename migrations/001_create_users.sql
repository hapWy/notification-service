-- +goose Up
-- +goose StatementBegin
create EXTENSION if not exists "uuid-ossp";
create table if not exists users(
    id uuid primary key default uuid_generate_v4(),
    telegram_id bigint not null unique,
    is_active boolean default true,
    created_at timestamp with time zone default now(),
    updated_at timestamp with time zone default now()
);

create index idx_users on users(telegram_id);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
drop table if exists users;
-- +goose StatementEnd