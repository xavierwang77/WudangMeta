docker run -d --name wudangMeta-pg --network wudang-network -e POSTGRES_PASSWORD=wudang+pwd -e POSTGRES_DB=wudang_db -e POSTGRES_USER=wudang_user -p 6680:5432 postgres:latest

CREATE USER wudang_user WITH LOGIN PASSWORD 'wudang+pwd';

CREATE DATABASE wudang_db;

ALTER DATABASE wudang_db OWNER TO wudang_user;

\c wudang_db;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO wudang_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO wudang_user;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO wudang_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO wudang_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO wudang_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO wudang_user;


