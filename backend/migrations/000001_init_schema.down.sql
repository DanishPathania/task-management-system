-- 000001_init_schema.down.sql
-- Rollback schema migrations

DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS tasks CASCADE;
DROP TABLE IF EXISTS project_members CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
DROP TABLE IF EXISTS users CASCADE;
