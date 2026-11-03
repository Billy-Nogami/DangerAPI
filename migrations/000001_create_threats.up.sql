CREATE TYPE threat_type AS ENUM (
    'trojan',
    'worm',
    'ransomware',
    'adware',
    'spyware',
    'other'
);

CREATE TYPE threat_severity AS ENUM (
    'low',
    'medium',
    'high',
    'critical'
);

CREATE TABLE threats (
    md5 VARCHAR(32) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type threat_type NOT NULL,
    severity threat_severity NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    categories TEXT[] NOT NULL DEFAULT '{}',
    first_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT threats_md5_format_check CHECK (md5 ~ '^[0-9a-f]{32}$'),
    CONSTRAINT threats_updated_at_check CHECK (updated_at >= created_at),
    CONSTRAINT threats_first_seen_at_check CHECK (
        first_seen_at IS NULL OR first_seen_at <= updated_at
    )
);
