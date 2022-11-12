alter table image_descriptions
    add last_modified timestamp without time zone default to_timestamp(0) not null;
alter table image_descriptions
    alter column last_modified drop default