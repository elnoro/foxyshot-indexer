create table image_descriptions
(
    file_id     text                    not null
        constraint image_descriptions_pk
            primary key,
    description text                    not null,
    public_uri  text      default ''    not null,
    created_at  timestamp default now() not null
);