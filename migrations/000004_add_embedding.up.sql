create extension if not exists vector;
alter table image_descriptions
    add clip_embedding vector(512)