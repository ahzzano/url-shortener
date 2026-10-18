-- +goose Up
CREATE TABLE urls (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    target_url TEXT NOT NULL
);

-- +goose Down
DROP TABLE urls;
