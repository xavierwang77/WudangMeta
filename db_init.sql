docker run -d --name WugongMeta-pg --network wugong-network -e POSTGRES_PASSWORD=wugong+pwd -e POSTGRES_DB=wugong_db -e POSTGRES_USER=wugong_user -p 6680:5432 postgres:latest

CREATE USER wugong_user WITH LOGIN PASSWORD 'wugong+pwd';

CREATE DATABASE wugong_db;

ALTER DATABASE wugong_db OWNER TO wugong_user;

\c wugong_db;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO wugong_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO wugong_user;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO wugong_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO wugong_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO wugong_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO wugong_user;


