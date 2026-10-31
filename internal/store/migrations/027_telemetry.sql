-- Aggregates only: a route is the mux pattern, never a raw path, and no IPs or
-- user IDs are stored.
CREATE TABLE telemetry_requests (
    minute      BIGINT NOT NULL,
    route       TEXT   NOT NULL,
    status      INT    NOT NULL, -- status class: 200, 300, 400, 500
    requests    BIGINT NOT NULL,
    duration_ms BIGINT NOT NULL,
    slow        BIGINT NOT NULL,
    PRIMARY KEY (minute, route, status)
);

-- One row per minute the app was running and could reach the database. A gap
-- between rows is an outage.
CREATE TABLE telemetry_heartbeats (minute BIGINT PRIMARY KEY);

CREATE TABLE telemetry_visitors (
    day      BIGINT PRIMARY KEY, -- start of the Manila calendar day (unix seconds)
    visitors BIGINT NOT NULL
);
