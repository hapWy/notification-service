-- +goose Up
-- +goose StatementBegin
create table if not exists templates(
    id uuid primary key default uuid_generate_v4(),
    name varchar(255) unique not null,
    body text not null,
    created_at timestamp with time zone default now(),
    updated_at timestamp with time zone default now()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists templates;
-- +goose StatementEnd
