-- Init tables need for db

CREATE TABLE IF NOT EXISTS user_table {
    username text NOT NULL,
    password_hash text NOT NULL,
    wallet text,
    PRIMARY KEY(username)
}
