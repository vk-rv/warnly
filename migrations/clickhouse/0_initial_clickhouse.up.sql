CREATE TABLE IF NOT EXISTS event
(
    `pid` UInt16 COMMENT 'Unique project identifier',
    `created_at` DateTime('UTC') COMMENT 'UTC dt',
    `deleted` UInt8 DEFAULT 0,
    `gid` UInt64,
    `retention_days` UInt8,
    `event_id` UUID COMMENT 'Unique event identifier',
    `platform` UInt8 COMMENT 'Platform identifier Go, Python, etc.',
    `env` LowCardinality(String) COMMENT 'Environment identifier (dev, stage, prod, etc)',
    `release` LowCardinality(String) COMMENT 'App version in semver',
    `ipv4` IPv4 COMMENT 'Sender ip addr version 4',
    `ipv6` IPv6 COMMENT 'Sender ip addr version 6',
    `user` String DEFAULT '',
    `user_hash` UInt64 MATERIALIZED cityHash64(user),
    `user_id` UInt64 COMMENT 'User identifier',
    `sdk_id` UInt8 COMMENT 'SDK identifier',
    `sdk_version` LowCardinality(String) COMMENT 'SDK semver version',
    `http_method` Enum8('GET' = 1, 'HEAD' = 2, 'POST' = 3, 'PUT' = 4, 'DELETE' = 5, 'CONNECT' = 6, 'OPTIONS' = 7, 'TRACE' = 8, 'PATCH' = 9, 'UNDEFINED' = 10) DEFAULT 10 COMMENT 'HTTP verbs',
    `http_referer` String DEFAULT '' COMMENT 'HTTP referer header',
    `tags.key` Array(String) COMMENT 'Tags key array',
    `tags.value` Array(String) COMMENT 'Tags value array',
    `_tags_hash_map` Array(UInt64) MATERIALIZED arrayMap((k, v) -> cityHash64(concat(replaceRegexpAll(k, '(\\=|\\\\)', '\\\\\\1'), '=', v)), tags.key, tags.value) COMMENT 'Key-val hash map',
    `contexts.key` Array(String) COMMENT 'Contexts key array',
    `contexts.value` Array(String) COMMENT 'Contexts value array',
    `primary_hash` UUID COMMENT 'Primary hash',
    `message` String COMMENT 'Message',
    `title` String COMMENT 'Title',
    `level` UInt8 COMMENT 'Log level',
    `location` String COMMENT 'Location',
    `type` UInt8 COMMENT 'Event type',
    `exception_stacks.type` Array(String) COMMENT 'Exception stack types',
    `exception_stacks.value` Array(String) COMMENT 'Exception stack values',
    `exception_frames.abs_path` Array(String) COMMENT 'Exception frame absolute path',
    `exception_frames.colno` Array(UInt32) COMMENT 'Exception frame column number',
    `exception_frames.filename` Array(String) COMMENT 'Exception frame filename',
    `exception_frames.function` Array(String) COMMENT 'Exception frame function',
    `exception_frames.lineno` Array(UInt32) COMMENT 'Exception frame line number',
    `exception_frames.in_app` Array(UInt8) COMMENT 'Exception frame in app',
    INDEX bf_tags_hash_map _tags_hash_map TYPE bloom_filter GRANULARITY 1,
    INDEX minmax_gid gid TYPE minmax GRANULARITY 1,
    INDEX bf_release release TYPE bloom_filter GRANULARITY 1
)
ENGINE = ReplacingMergeTree(deleted)
PARTITION BY (retention_days, toMonday(created_at))
ORDER BY (pid, toStartOfDay(created_at), primary_hash, cityHash64(event_id))
SAMPLE BY cityHash64(event_id)
TTL created_at + toIntervalDay(retention_days);
